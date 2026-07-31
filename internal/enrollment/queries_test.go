package enrollment

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const testDatabaseURL = "postgresql://postgres:password@localhost:5432/postgres?sslmode=disable"

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, testDatabaseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "enrollment tests: connect to PostgreSQL: %v\n", err)
		os.Exit(1)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		fmt.Fprintf(os.Stderr, "enrollment tests: PostgreSQL is not ready at %s: %v\n", testDatabaseURL, err)
		os.Exit(1)
	}

	var schemaReady bool
	err = pool.QueryRow(ctx, `
		SELECT to_regclass('public.members') IS NOT NULL
		   AND to_regclass('public.courses') IS NOT NULL
		   AND to_regclass('public.enrollments') IS NOT NULL
	`).Scan(&schemaReady)
	if err != nil || !schemaReady {
		pool.Close()
		fmt.Fprintln(os.Stderr, "enrollment tests: expected members, courses, and enrollments tables; apply all migrations first")
		os.Exit(1)
	}

	testPool = pool
	code := m.Run()
	pool.Close()
	os.Exit(code)
}

func beginTestTx(t *testing.T) (context.Context, pgx.Tx, *Queries) {
	t.Helper()

	ctx := context.Background()
	tx, err := testPool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin test transaction: %v", err)
	}
	t.Cleanup(func() {
		_ = tx.Rollback(context.Background())
	})
	return ctx, tx, New(tx)
}

func uniqueEmail(prefix string) string {
	return fmt.Sprintf("%s-%s@example.com", prefix, uuid.NewString())
}

func insertMemberFixture(t *testing.T, ctx context.Context, tx pgx.Tx, name string) (uuid.UUID, string) {
	t.Helper()

	id := uuid.New()
	email := uniqueEmail("enrollment-member")
	_, err := tx.Exec(ctx,
		`INSERT INTO members (id, name, email) VALUES ($1, $2, $3)`,
		id, name, email,
	)
	if err != nil {
		t.Fatalf("insert member fixture: %v", err)
	}
	return id, email
}

func insertCourseFixture(t *testing.T, ctx context.Context, tx pgx.Tx, title string, capacity int32) uuid.UUID {
	t.Helper()

	id := uuid.New()
	_, err := tx.Exec(ctx,
		`INSERT INTO courses (id, title, capacity) VALUES ($1, $2, $3)`,
		id, title, capacity,
	)
	if err != nil {
		t.Fatalf("insert course fixture: %v", err)
	}
	return id
}

func createEnrollmentFixture(
	t *testing.T,
	ctx context.Context,
	queries *Queries,
	memberID uuid.UUID,
	courseID uuid.UUID,
) {
	t.Helper()

	_, err := queries.CreateEnrollment(ctx, CreateEnrollmentParams{
		MemberID: memberID,
		CourseID: courseID,
	})
	if err != nil {
		t.Fatalf("CreateEnrollment fixture: %v", err)
	}
}

func requirePostgresCode(t *testing.T, err error, code string) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected PostgreSQL error %s, got nil", code)
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		t.Fatalf("expected PostgreSQL error %s, got %T: %v", code, err, err)
	}
	if pgErr.Code != code {
		t.Fatalf("expected PostgreSQL error %s, got %s: %v", code, pgErr.Code, err)
	}
}

func TestCreateEnrollment(t *testing.T) {
	ctx, tx, queries := beginTestTx(t)
	memberID, _ := insertMemberFixture(t, ctx, tx, "Create Student")
	courseID := insertCourseFixture(t, ctx, tx, "Create Course", 10)
	before := time.Now().Add(-time.Minute)

	got, err := queries.CreateEnrollment(ctx, CreateEnrollmentParams{
		MemberID: memberID,
		CourseID: courseID,
	})
	if err != nil {
		t.Fatalf("CreateEnrollment: %v", err)
	}

	if got.MemberID != memberID || got.CourseID != courseID {
		t.Fatalf("CreateEnrollment returned ids (%s, %s), want (%s, %s)", got.MemberID, got.CourseID, memberID, courseID)
	}
	if got.Status != "enrolled" {
		t.Fatalf("CreateEnrollment status = %q, want enrolled", got.Status)
	}
	if got.EnrolledAt.Before(before) || got.EnrolledAt.After(time.Now().Add(time.Minute)) {
		t.Fatalf("CreateEnrollment returned unexpected enrolled_at: %v", got.EnrolledAt)
	}
}

func TestGetEnrollment(t *testing.T) {
	ctx, tx, queries := beginTestTx(t)
	memberID, _ := insertMemberFixture(t, ctx, tx, "Get Student")
	courseID := insertCourseFixture(t, ctx, tx, "Get Course", 10)
	createEnrollmentFixture(t, ctx, queries, memberID, courseID)

	got, err := queries.GetEnrollment(ctx, GetEnrollmentParams{
		MemberID: memberID,
		CourseID: courseID,
	})
	if err != nil {
		t.Fatalf("GetEnrollment: %v", err)
	}
	if got.MemberID != memberID || got.CourseID != courseID || got.Status != "enrolled" {
		t.Fatalf("GetEnrollment returned %+v", got)
	}
}

func TestListEnrollmentsContainsCreatedEnrollments(t *testing.T) {
	ctx, tx, queries := beginTestTx(t)
	firstMemberID, _ := insertMemberFixture(t, ctx, tx, "List Student One")
	secondMemberID, _ := insertMemberFixture(t, ctx, tx, "List Student Two")
	courseID := insertCourseFixture(t, ctx, tx, "List Course", 10)
	createEnrollmentFixture(t, ctx, queries, firstMemberID, courseID)
	createEnrollmentFixture(t, ctx, queries, secondMemberID, courseID)

	got, err := queries.ListEnrollments(ctx)
	if err != nil {
		t.Fatalf("ListEnrollments: %v", err)
	}

	found := map[uuid.UUID]bool{}
	for _, item := range got {
		if item.CourseID == courseID && (item.MemberID == firstMemberID || item.MemberID == secondMemberID) {
			found[item.MemberID] = true
		}
	}
	if !found[firstMemberID] || !found[secondMemberID] {
		t.Fatalf("ListEnrollments did not contain both created enrollments: first=%v second=%v", found[firstMemberID], found[secondMemberID])
	}
}

func TestUpdateEnrollmentStatusOnlyUpdatesRequestedEnrollment(t *testing.T) {
	ctx, tx, queries := beginTestTx(t)
	memberID, _ := insertMemberFixture(t, ctx, tx, "Update Student")
	targetCourseID := insertCourseFixture(t, ctx, tx, "Target Course", 10)
	otherCourseID := insertCourseFixture(t, ctx, tx, "Other Course", 10)
	createEnrollmentFixture(t, ctx, queries, memberID, targetCourseID)
	createEnrollmentFixture(t, ctx, queries, memberID, otherCourseID)

	updated, err := queries.UpdateEnrollmentStatus(ctx, UpdateEnrollmentStatusParams{
		Status:   "completed",
		MemberID: memberID,
		CourseID: targetCourseID,
	})
	if err != nil {
		t.Fatalf("UpdateEnrollmentStatus: %v", err)
	}
	if updated.MemberID != memberID || updated.CourseID != targetCourseID || updated.Status != "completed" {
		t.Fatalf("UpdateEnrollmentStatus returned %+v", updated)
	}

	other, err := queries.GetEnrollment(ctx, GetEnrollmentParams{
		MemberID: memberID,
		CourseID: otherCourseID,
	})
	if err != nil {
		t.Fatalf("GetEnrollment(other): %v", err)
	}
	if other.Status != "enrolled" {
		t.Fatalf("UpdateEnrollmentStatus changed another enrollment to %q", other.Status)
	}
}

func TestDeleteEnrollment(t *testing.T) {
	ctx, tx, queries := beginTestTx(t)
	memberID, _ := insertMemberFixture(t, ctx, tx, "Delete Student")
	courseID := insertCourseFixture(t, ctx, tx, "Delete Course", 10)
	createEnrollmentFixture(t, ctx, queries, memberID, courseID)

	deleted, err := queries.DeleteEnrollment(ctx, DeleteEnrollmentParams{
		MemberID: memberID,
		CourseID: courseID,
	})
	if err != nil {
		t.Fatalf("DeleteEnrollment: %v", err)
	}
	if deleted.MemberID != memberID || deleted.CourseID != courseID {
		t.Fatalf("DeleteEnrollment returned %+v", deleted)
	}

	_, err = queries.GetEnrollment(ctx, GetEnrollmentParams{
		MemberID: memberID,
		CourseID: courseID,
	})
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("GetEnrollment after delete error = %v, want pgx.ErrNoRows", err)
	}
}

func TestCreateEnrollmentRejectsDuplicatePair(t *testing.T) {
	ctx, tx, queries := beginTestTx(t)
	memberID, _ := insertMemberFixture(t, ctx, tx, "Duplicate Student")
	courseID := insertCourseFixture(t, ctx, tx, "Duplicate Course", 10)
	createEnrollmentFixture(t, ctx, queries, memberID, courseID)

	_, err := queries.CreateEnrollment(ctx, CreateEnrollmentParams{
		MemberID: memberID,
		CourseID: courseID,
	})
	requirePostgresCode(t, err, "23505")
}

func TestCreateEnrollmentRejectsMissingMember(t *testing.T) {
	ctx, tx, queries := beginTestTx(t)
	courseID := insertCourseFixture(t, ctx, tx, "Foreign Key Course", 10)

	_, err := queries.CreateEnrollment(ctx, CreateEnrollmentParams{
		MemberID: uuid.New(),
		CourseID: courseID,
	})
	requirePostgresCode(t, err, "23503")
}

func TestCreateEnrollmentRejectsMissingCourse(t *testing.T) {
	ctx, tx, queries := beginTestTx(t)
	memberID, _ := insertMemberFixture(t, ctx, tx, "Foreign Key Student")

	_, err := queries.CreateEnrollment(ctx, CreateEnrollmentParams{
		MemberID: memberID,
		CourseID: uuid.New(),
	})
	requirePostgresCode(t, err, "23503")
}

func TestUpdateEnrollmentStatusRejectsInvalidStatus(t *testing.T) {
	ctx, tx, queries := beginTestTx(t)
	memberID, _ := insertMemberFixture(t, ctx, tx, "Invalid Status Student")
	courseID := insertCourseFixture(t, ctx, tx, "Invalid Status Course", 10)
	createEnrollmentFixture(t, ctx, queries, memberID, courseID)

	_, err := queries.UpdateEnrollmentStatus(ctx, UpdateEnrollmentStatusParams{
		Status:   "waiting",
		MemberID: memberID,
		CourseID: courseID,
	})
	requirePostgresCode(t, err, "23514")
}

func TestGetEnrollmentMissing(t *testing.T) {
	ctx, _, queries := beginTestTx(t)

	_, err := queries.GetEnrollment(ctx, GetEnrollmentParams{
		MemberID: uuid.New(),
		CourseID: uuid.New(),
	})
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("GetEnrollment missing error = %v, want pgx.ErrNoRows", err)
	}
}

func TestGetEnrollmentDetail(t *testing.T) {
	ctx, tx, queries := beginTestTx(t)
	memberID, email := insertMemberFixture(t, ctx, tx, "Detail Student")
	courseID := insertCourseFixture(t, ctx, tx, "Detail Course", 42)
	createEnrollmentFixture(t, ctx, queries, memberID, courseID)

	got, err := queries.GetEnrollmentDetail(ctx, GetEnrollmentDetailParams{
		MemberID: memberID,
		CourseID: courseID,
	})
	if err != nil {
		t.Fatalf("GetEnrollmentDetail: %v", err)
	}

	if got.MemberID != memberID ||
		got.MemberName != "Detail Student" ||
		got.MemberEmail != email ||
		got.CourseID != courseID ||
		got.CourseTitle != "Detail Course" ||
		got.CourseCapacity != 42 ||
		got.Status != "enrolled" {
		t.Fatalf("GetEnrollmentDetail returned %+v", got)
	}
	if got.EnrolledAt.IsZero() {
		t.Fatal("GetEnrollmentDetail returned zero enrolled_at")
	}
}

func TestGetEnrollmentDetailMissing(t *testing.T) {
	ctx, _, queries := beginTestTx(t)

	_, err := queries.GetEnrollmentDetail(ctx, GetEnrollmentDetailParams{
		MemberID: uuid.New(),
		CourseID: uuid.New(),
	})
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("GetEnrollmentDetail missing error = %v, want pgx.ErrNoRows", err)
	}
}

func TestDeletingMemberCascadesEnrollment(t *testing.T) {
	ctx, tx, queries := beginTestTx(t)
	memberID, _ := insertMemberFixture(t, ctx, tx, "Cascade Member")
	courseID := insertCourseFixture(t, ctx, tx, "Cascade Member Course", 10)
	createEnrollmentFixture(t, ctx, queries, memberID, courseID)

	tag, err := tx.Exec(ctx, `DELETE FROM members WHERE id = $1`, memberID)
	if err != nil {
		t.Fatalf("delete member fixture: %v", err)
	}
	if tag.RowsAffected() != 1 {
		t.Fatalf("deleted %d members, want 1", tag.RowsAffected())
	}

	_, err = queries.GetEnrollment(ctx, GetEnrollmentParams{
		MemberID: memberID,
		CourseID: courseID,
	})
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("GetEnrollment after member delete error = %v, want pgx.ErrNoRows", err)
	}
}

func TestDeletingCourseCascadesEnrollment(t *testing.T) {
	ctx, tx, queries := beginTestTx(t)
	memberID, _ := insertMemberFixture(t, ctx, tx, "Cascade Course Student")
	courseID := insertCourseFixture(t, ctx, tx, "Cascade Course", 10)
	createEnrollmentFixture(t, ctx, queries, memberID, courseID)

	tag, err := tx.Exec(ctx, `DELETE FROM courses WHERE id = $1`, courseID)
	if err != nil {
		t.Fatalf("delete course fixture: %v", err)
	}
	if tag.RowsAffected() != 1 {
		t.Fatalf("deleted %d courses, want 1", tag.RowsAffected())
	}

	_, err = queries.GetEnrollment(ctx, GetEnrollmentParams{
		MemberID: memberID,
		CourseID: courseID,
	})
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("GetEnrollment after course delete error = %v, want pgx.ErrNoRows", err)
	}
}
