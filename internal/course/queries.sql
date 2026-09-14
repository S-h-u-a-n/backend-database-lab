-- name: CreateCourse :one
INSERT INTO courses(title, capacity)
VALUES (sqlc.arg(title), sqlc.arg(capacity))
RETURNING id, title, capacity, created_at;

-- name: GetCourse :one
SELECT id, title, capacity, created_at FROM courses
WHERE id = sqlc.arg(id);

-- name: ListCourses :many
SELECT id, title, capacity, created_at FROM courses
ORDER BY created_at ASC, id ASC;

-- name: UpdateCourse :one
UPDATE courses
SET
    title = sqlc.arg(title),
    capacity = sqlc.arg(capacity)
WHERE id = sqlc.arg(id)
RETURNING id, title, capacity, created_at;

-- name: DeleteCourse :one
DELETE FROM courses
WHERE id = sqlc.arg(id)
RETURNING id, title, capacity, created_at;

-- name: ListCourseRoster :many
SELECT m.id AS member_id,
       m.name AS member_name,
       m.email AS member_email,
       e.status AS status,
       e.enrolled_at AS enrolled_at
FROM courses c
JOIN enrollments e
    ON e.course_id = c.id
JOIN members m
    ON m.id = e.member_id
WHERE
    c.id = sqlc.arg(course_id)
    AND e.status = 'enrolled'
ORDER BY member_name ASC, member_id ASC;