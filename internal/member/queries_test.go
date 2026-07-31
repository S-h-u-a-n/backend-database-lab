package member

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"testing"
	"time"

	"database-training/databaseutil"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

const testDatabaseURL = "postgresql://postgres:password@localhost:5432/postgres?sslmode=disable"

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, testDatabaseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "member tests: connect to PostgreSQL: %v\n", err)
		os.Exit(1)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		fmt.Fprintf(os.Stderr, "member tests: PostgreSQL is not ready at %s: %v\n", testDatabaseURL, err)
		os.Exit(1)
	}
	if err := databaseutil.MigrationUp("file://../database/migrations", testDatabaseURL, zap.NewNop()); err != nil {
		pool.Close()
		fmt.Fprintf(os.Stderr, "member tests: apply migrations: %v\n", err)
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
		fmt.Fprintln(os.Stderr, "member tests: migrations did not create the members, courses, and enrollments tables")
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
	email := uniqueEmail("member-fixture")
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

func TestCreateMember(t *testing.T) {
	ctx, _, queries := beginTestTx(t)
	before := time.Now().Add(-time.Minute)
	email := uniqueEmail("create")

	got, err := queries.CreateMember(ctx, CreateMemberParams{
		Name:  "Alice",
		Email: email,
	})
	if err != nil {
		t.Fatalf("CreateMember: %v", err)
	}

	if got.ID == uuid.Nil {
		t.Error("CreateMember returned a zero UUID")
	}
	if got.Name != "Alice" || got.Email != email {
		t.Fatalf("CreateMember returned (%q, %q), want (%q, %q)", got.Name, got.Email, "Alice", email)
	}
	if !got.JoinedAt.Valid ||
		got.JoinedAt.Time.Before(before) ||
		got.JoinedAt.Time.After(time.Now().Add(time.Minute)) {
		t.Fatalf("CreateMember returned unexpected joined_at: %v", got.JoinedAt)
	}
}

func TestGetMember(t *testing.T) {
	ctx, tx, queries := beginTestTx(t)
	id, email := insertMemberFixture(t, ctx, tx, "Bob")

	got, err := queries.GetMember(ctx, id)
	if err != nil {
		t.Fatalf("GetMember: %v", err)
	}
	if got.ID != id || got.Name != "Bob" || got.Email != email {
		t.Fatalf("GetMember returned %+v, want id=%s name=Bob email=%s", got, id, email)
	}
}

func TestListMembersContainsCreatedMembers(t *testing.T) {
	ctx, _, queries := beginTestTx(t)
	first, err := queries.CreateMember(ctx, CreateMemberParams{Name: "List One", Email: uniqueEmail("list-one")})
	if err != nil {
		t.Fatalf("CreateMember(first): %v", err)
	}
	second, err := queries.CreateMember(ctx, CreateMemberParams{Name: "List Two", Email: uniqueEmail("list-two")})
	if err != nil {
		t.Fatalf("CreateMember(second): %v", err)
	}

	got, err := queries.ListMembers(ctx)
	if err != nil {
		t.Fatalf("ListMembers: %v", err)
	}

	found := map[uuid.UUID]bool{}
	for _, item := range got {
		if item.ID == first.ID || item.ID == second.ID {
			found[item.ID] = true
		}
	}
	if !found[first.ID] || !found[second.ID] {
		t.Fatalf("ListMembers did not contain both created members: first=%v second=%v", found[first.ID], found[second.ID])
	}
}

func TestUpdateMemberOnlyUpdatesRequestedMember(t *testing.T) {
	ctx, _, queries := beginTestTx(t)
	target, err := queries.CreateMember(ctx, CreateMemberParams{Name: "Before", Email: uniqueEmail("before")})
	if err != nil {
		t.Fatalf("CreateMember(target): %v", err)
	}
	other, err := queries.CreateMember(ctx, CreateMemberParams{Name: "Untouched", Email: uniqueEmail("untouched")})
	if err != nil {
		t.Fatalf("CreateMember(other): %v", err)
	}
	updatedEmail := uniqueEmail("after")

	updated, err := queries.UpdateMember(ctx, UpdateMemberParams{
		Name:  "After",
		Email: updatedEmail,
		ID:    target.ID,
	})
	if err != nil {
		t.Fatalf("UpdateMember: %v", err)
	}
	if updated.ID != target.ID || updated.Name != "After" || updated.Email != updatedEmail {
		t.Fatalf("UpdateMember returned %+v", updated)
	}

	unchanged, err := queries.GetMember(ctx, other.ID)
	if err != nil {
		t.Fatalf("GetMember(other): %v", err)
	}
	if unchanged.Name != other.Name || unchanged.Email != other.Email {
		t.Fatalf("UpdateMember changed another member: got %+v want %+v", unchanged, other)
	}
}

func TestDeleteMember(t *testing.T) {
	ctx, _, queries := beginTestTx(t)
	created, err := queries.CreateMember(ctx, CreateMemberParams{Name: "Delete Me", Email: uniqueEmail("delete")})
	if err != nil {
		t.Fatalf("CreateMember: %v", err)
	}

	deleted, err := queries.DeleteMember(ctx, created.ID)
	if err != nil {
		t.Fatalf("DeleteMember: %v", err)
	}
	if deleted.ID != created.ID || deleted.Name != created.Name || deleted.Email != created.Email {
		t.Fatalf("DeleteMember returned %+v, want %+v", deleted, created)
	}

	_, err = queries.GetMember(ctx, created.ID)
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("GetMember after delete error = %v, want pgx.ErrNoRows", err)
	}
}

func TestCreateMemberRejectsDuplicateEmail(t *testing.T) {
	ctx, _, queries := beginTestTx(t)
	email := uniqueEmail("duplicate")

	_, err := queries.CreateMember(ctx, CreateMemberParams{Name: "First", Email: email})
	if err != nil {
		t.Fatalf("CreateMember(first): %v", err)
	}
	_, err = queries.CreateMember(ctx, CreateMemberParams{Name: "Second", Email: email})
	requirePostgresCode(t, err, "23505")
}

func TestGetMemberMissing(t *testing.T) {
	ctx, _, queries := beginTestTx(t)

	_, err := queries.GetMember(ctx, uuid.New())
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("GetMember missing error = %v, want pgx.ErrNoRows", err)
	}
}

func TestListMemberCoursesIncludesHistoryAndSorts(t *testing.T) {
	ctx, tx, queries := beginTestTx(t)
	memberID, _ := insertMemberFixture(t, ctx, tx, "History Student")

	type expectedCourse struct {
		id     uuid.UUID
		title  string
		status string
	}
	expected := []expectedCourse{
		{id: insertCourseFixture(t, ctx, tx, "Gamma"), title: "Gamma", status: "enrolled"},
		{id: insertCourseFixture(t, ctx, tx, "Alpha"), title: "Alpha", status: "completed"},
		{id: insertCourseFixture(t, ctx, tx, "Alpha"), title: "Alpha", status: "cancelled"},
	}
	for _, item := range expected {
		insertEnrollmentFixture(t, ctx, tx, memberID, item.id, item.status)
	}
	sort.Slice(expected, func(i, j int) bool {
		if expected[i].title != expected[j].title {
			return expected[i].title < expected[j].title
		}
		return expected[i].id.String() < expected[j].id.String()
	})

	got, err := queries.ListMemberCourses(ctx, memberID)
	if err != nil {
		t.Fatalf("ListMemberCourses: %v", err)
	}
	if len(got) != len(expected) {
		t.Fatalf("ListMemberCourses returned %d rows, want %d", len(got), len(expected))
	}
	for i, want := range expected {
		if got[i].CourseID != want.id || got[i].CourseTitle != want.title || got[i].Status != want.status {
			t.Errorf("ListMemberCourses[%d] = %+v, want id=%s title=%q status=%q", i, got[i], want.id, want.title, want.status)
		}
		if !got[i].EnrolledAt.Valid || got[i].EnrolledAt.Time.IsZero() {
			t.Errorf("ListMemberCourses[%d] returned zero enrolled_at", i)
		}
	}
}

func TestListMemberCoursesEmpty(t *testing.T) {
	ctx, tx, queries := beginTestTx(t)
	memberID, _ := insertMemberFixture(t, ctx, tx, "No Courses")

	got, err := queries.ListMemberCourses(ctx, memberID)
	if err != nil {
		t.Fatalf("ListMemberCourses: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("ListMemberCourses returned %d rows, want 0", len(got))
	}
}

func TestListClassmatesFiltersDeduplicatesAndSorts(t *testing.T) {
	ctx, tx, queries := beginTestTx(t)
	requestedID, _ := insertMemberFixture(t, ctx, tx, "Requested")
	firstID, firstEmail := insertMemberFixture(t, ctx, tx, "Alex")
	secondID, secondEmail := insertMemberFixture(t, ctx, tx, "Alex")
	cancelledID, _ := insertMemberFixture(t, ctx, tx, "Cancelled")
	inactiveCourseID, _ := insertMemberFixture(t, ctx, tx, "Inactive Course")

	firstCourse := insertCourseFixture(t, ctx, tx, "Course One")
	secondCourse := insertCourseFixture(t, ctx, tx, "Course Two")
	thirdCourse := insertCourseFixture(t, ctx, tx, "Course Three")

	insertEnrollmentFixture(t, ctx, tx, requestedID, firstCourse, "enrolled")
	insertEnrollmentFixture(t, ctx, tx, requestedID, secondCourse, "enrolled")
	insertEnrollmentFixture(t, ctx, tx, requestedID, thirdCourse, "cancelled")
	insertEnrollmentFixture(t, ctx, tx, firstID, firstCourse, "enrolled")
	insertEnrollmentFixture(t, ctx, tx, firstID, secondCourse, "enrolled")
	insertEnrollmentFixture(t, ctx, tx, secondID, firstCourse, "enrolled")
	insertEnrollmentFixture(t, ctx, tx, cancelledID, firstCourse, "cancelled")
	insertEnrollmentFixture(t, ctx, tx, inactiveCourseID, thirdCourse, "enrolled")

	type expectedClassmate struct {
		id    uuid.UUID
		email string
	}
	expected := []expectedClassmate{
		{id: firstID, email: firstEmail},
		{id: secondID, email: secondEmail},
	}
	sort.Slice(expected, func(i, j int) bool {
		return expected[i].id.String() < expected[j].id.String()
	})

	got, err := queries.ListClassmates(ctx, requestedID)
	if err != nil {
		t.Fatalf("ListClassmates: %v", err)
	}
	if len(got) != len(expected) {
		t.Fatalf("ListClassmates returned %d rows, want %d: %+v", len(got), len(expected), got)
	}
	for i, want := range expected {
		if got[i].MemberID != want.id || got[i].MemberName != "Alex" || got[i].MemberEmail != want.email {
			t.Errorf("ListClassmates[%d] = %+v, want id=%s name=Alex email=%s", i, got[i], want.id, want.email)
		}
	}
}
