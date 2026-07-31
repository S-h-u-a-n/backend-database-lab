package course

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
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
		fmt.Fprintf(os.Stderr, "course tests: connect to PostgreSQL: %v\n", err)
		os.Exit(1)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		fmt.Fprintf(os.Stderr, "course tests: PostgreSQL is not ready at %s: %v\n", testDatabaseURL, err)
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
		fmt.Fprintln(os.Stderr, "course tests: expected members, courses, and enrollments tables; apply all migrations first")
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
	email := uniqueEmail("course-member")
	_, err := tx.Exec(ctx,
		`INSERT INTO members (id, name, email) VALUES ($1, $2, $3)`,
		id, name, email,
	)
	if err != nil {
		t.Fatalf("insert member fixture: %v", err)
	}
	return id, email
}

func insertCourseFixture(t *testing.T, ctx context.Context, tx pgx.Tx, title string) uuid.UUID {
	t.Helper()

	id := uuid.New()
	_, err := tx.Exec(ctx,
		`INSERT INTO courses (id, title, capacity) VALUES ($1, $2, $3)`,
		id, title, 20,
	)
	if err != nil {
		t.Fatalf("insert course fixture: %v", err)
	}
	return id
}

func insertEnrollmentFixture(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	memberID uuid.UUID,
	courseID uuid.UUID,
	status string,
) {
	t.Helper()

	_, err := tx.Exec(ctx,
		`INSERT INTO enrollments (member_id, course_id, status) VALUES ($1, $2, $3)`,
		memberID, courseID, status,
	)
	if err != nil {
		t.Fatalf("insert enrollment fixture: %v", err)
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

func TestCreateCourse(t *testing.T) {
	ctx, _, queries := beginTestTx(t)
	before := time.Now().Add(-time.Minute)

	got, err := queries.CreateCourse(ctx, CreateCourseParams{
		Title:    "Database Systems",
		Capacity: 30,
	})
	if err != nil {
		t.Fatalf("CreateCourse: %v", err)
	}

	if got.ID == uuid.Nil {
		t.Error("CreateCourse returned a zero UUID")
	}
	if got.Title != "Database Systems" || got.Capacity != 30 {
		t.Fatalf("CreateCourse returned (%q, %d), want (%q, %d)", got.Title, got.Capacity, "Database Systems", 30)
	}
	if !got.CreatedAt.Valid ||
		got.CreatedAt.Time.Before(before) ||
		got.CreatedAt.Time.After(time.Now().Add(time.Minute)) {
		t.Fatalf("CreateCourse returned unexpected created_at: %v", got.CreatedAt)
	}
}

func TestGetCourse(t *testing.T) {
	ctx, tx, queries := beginTestTx(t)
	id := insertCourseFixture(t, ctx, tx, "Get Course")

	got, err := queries.GetCourse(ctx, id)
	if err != nil {
		t.Fatalf("GetCourse: %v", err)
	}
	if got.ID != id || got.Title != "Get Course" || got.Capacity != 20 {
		t.Fatalf("GetCourse returned %+v, want id=%s title=%q capacity=20", got, id, "Get Course")
	}
}

func TestListCoursesContainsCreatedCourses(t *testing.T) {
	ctx, _, queries := beginTestTx(t)
	first, err := queries.CreateCourse(ctx, CreateCourseParams{Title: "List One", Capacity: 10})
	if err != nil {
		t.Fatalf("CreateCourse(first): %v", err)
	}
	second, err := queries.CreateCourse(ctx, CreateCourseParams{Title: "List Two", Capacity: 20})
	if err != nil {
		t.Fatalf("CreateCourse(second): %v", err)
	}

	got, err := queries.ListCourses(ctx)
	if err != nil {
		t.Fatalf("ListCourses: %v", err)
	}

	found := map[uuid.UUID]bool{}
	for _, item := range got {
		if item.ID == first.ID || item.ID == second.ID {
			found[item.ID] = true
		}
	}
	if !found[first.ID] || !found[second.ID] {
		t.Fatalf("ListCourses did not contain both created courses: first=%v second=%v", found[first.ID], found[second.ID])
	}
}

func TestUpdateCourseOnlyUpdatesRequestedCourse(t *testing.T) {
	ctx, _, queries := beginTestTx(t)
	target, err := queries.CreateCourse(ctx, CreateCourseParams{Title: "Before", Capacity: 10})
	if err != nil {
		t.Fatalf("CreateCourse(target): %v", err)
	}
	other, err := queries.CreateCourse(ctx, CreateCourseParams{Title: "Untouched", Capacity: 20})
	if err != nil {
		t.Fatalf("CreateCourse(other): %v", err)
	}

	updated, err := queries.UpdateCourse(ctx, UpdateCourseParams{
		Title:    "After",
		Capacity: 50,
		ID:       target.ID,
	})
	if err != nil {
		t.Fatalf("UpdateCourse: %v", err)
	}
	if updated.ID != target.ID || updated.Title != "After" || updated.Capacity != 50 {
		t.Fatalf("UpdateCourse returned %+v", updated)
	}

	unchanged, err := queries.GetCourse(ctx, other.ID)
	if err != nil {
		t.Fatalf("GetCourse(other): %v", err)
	}
	if unchanged.Title != other.Title || unchanged.Capacity != other.Capacity {
		t.Fatalf("UpdateCourse changed another course: got %+v want %+v", unchanged, other)
	}
}

func TestDeleteCourse(t *testing.T) {
	ctx, _, queries := beginTestTx(t)
	created, err := queries.CreateCourse(ctx, CreateCourseParams{Title: "Delete Me", Capacity: 10})
	if err != nil {
		t.Fatalf("CreateCourse: %v", err)
	}

	deleted, err := queries.DeleteCourse(ctx, created.ID)
	if err != nil {
		t.Fatalf("DeleteCourse: %v", err)
	}
	if deleted.ID != created.ID || deleted.Title != created.Title || deleted.Capacity != created.Capacity {
		t.Fatalf("DeleteCourse returned %+v, want %+v", deleted, created)
	}

	_, err = queries.GetCourse(ctx, created.ID)
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("GetCourse after delete error = %v, want pgx.ErrNoRows", err)
	}
}

func TestCreateCourseRejectsZeroCapacity(t *testing.T) {
	ctx, _, queries := beginTestTx(t)

	_, err := queries.CreateCourse(ctx, CreateCourseParams{Title: "Zero", Capacity: 0})
	requirePostgresCode(t, err, "23514")
}

func TestCreateCourseRejectsNegativeCapacity(t *testing.T) {
	ctx, _, queries := beginTestTx(t)

	_, err := queries.CreateCourse(ctx, CreateCourseParams{Title: "Negative", Capacity: -1})
	requirePostgresCode(t, err, "23514")
}

func TestGetCourseMissing(t *testing.T) {
	ctx, _, queries := beginTestTx(t)

	_, err := queries.GetCourse(ctx, uuid.New())
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("GetCourse missing error = %v, want pgx.ErrNoRows", err)
	}
}

func TestListCourseRosterFiltersAndSorts(t *testing.T) {
	ctx, tx, queries := beginTestTx(t)
	courseID := insertCourseFixture(t, ctx, tx, "Roster Course")
	firstID, firstEmail := insertMemberFixture(t, ctx, tx, "Alex")
	secondID, secondEmail := insertMemberFixture(t, ctx, tx, "Alex")
	completedID, _ := insertMemberFixture(t, ctx, tx, "Completed")
	cancelledID, _ := insertMemberFixture(t, ctx, tx, "Cancelled")

	insertEnrollmentFixture(t, ctx, tx, firstID, courseID, "enrolled")
	insertEnrollmentFixture(t, ctx, tx, secondID, courseID, "enrolled")
	insertEnrollmentFixture(t, ctx, tx, completedID, courseID, "completed")
	insertEnrollmentFixture(t, ctx, tx, cancelledID, courseID, "cancelled")

	type expectedMember struct {
		id    uuid.UUID
		email string
	}
	expected := []expectedMember{
		{id: firstID, email: firstEmail},
		{id: secondID, email: secondEmail},
	}
	sort.Slice(expected, func(i, j int) bool {
		return expected[i].id.String() < expected[j].id.String()
	})

	got, err := queries.ListCourseRoster(ctx, courseID)
	if err != nil {
		t.Fatalf("ListCourseRoster: %v", err)
	}
	if len(got) != len(expected) {
		t.Fatalf("ListCourseRoster returned %d rows, want %d: %+v", len(got), len(expected), got)
	}
	for i, want := range expected {
		if got[i].MemberID != want.id ||
			got[i].MemberName != "Alex" ||
			got[i].MemberEmail != want.email ||
			got[i].Status != "enrolled" {
			t.Errorf("ListCourseRoster[%d] = %+v, want id=%s name=Alex email=%s status=enrolled", i, got[i], want.id, want.email)
		}
		if !got[i].EnrolledAt.Valid || got[i].EnrolledAt.Time.IsZero() {
			t.Errorf("ListCourseRoster[%d] returned zero enrolled_at", i)
		}
	}
}

func TestListCourseRosterEmpty(t *testing.T) {
	ctx, tx, queries := beginTestTx(t)
	courseID := insertCourseFixture(t, ctx, tx, "Empty Roster")
	completedID, _ := insertMemberFixture(t, ctx, tx, "Former Student")
	insertEnrollmentFixture(t, ctx, tx, completedID, courseID, "completed")

	got, err := queries.ListCourseRoster(ctx, courseID)
	if err != nil {
		t.Fatalf("ListCourseRoster: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("ListCourseRoster returned %d rows, want 0", len(got))
	}
}
