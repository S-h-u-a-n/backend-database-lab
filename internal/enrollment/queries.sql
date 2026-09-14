-- name: CreateEnrollment :one
INSERT INTO enrollments(member_id, course_id)
VALUES (sqlc.arg(member_id), sqlc.arg(course_id))
RETURNING member_id, course_id, status, enrolled_at;

-- name: GetEnrollment :one
SELECT member_id, course_id, status, enrolled_at FROM enrollments
WHERE
    member_id = sqlc.arg(member_id)
    AND course_id = sqlc.arg(course_id);

-- name: ListEnrollments :many
SELECT member_id, course_id, status, enrolled_at FROM enrollments
ORDER BY enrolled_at ASC, member_id ASC, course_id ASC;

-- name: UpdateEnrollmentStatus :one
UPDATE enrollments
SET status = sqlc.arg(status)
WHERE
    member_id = sqlc.arg(member_id)
    AND course_id = sqlc.arg(course_id)
RETURNING member_id, course_id, status, enrolled_at;

-- name: DeleteEnrollment :one
DELETE FROM enrollments
WHERE
    member_id = sqlc.arg(member_id)
    AND course_id = sqlc.arg(course_id)
RETURNING member_id, course_id, status, enrolled_at;

-- name: GetEnrollmentDetail :one
SELECT m.id AS member_id,
       m.name AS member_name,
       m.email AS member_email,
       c.id AS course_id,
       c.title AS course_title,
       c.capacity AS course_capacity,
       e.status AS status,
       e.enrolled_at AS enrolled_at
FROM enrollments e
JOIN members m
    ON m.id = e.member_id
JOIN courses c
    ON c.id = e.course_id
WHERE
    e.member_id = sqlc.arg(member_id)
    AND e.course_id = sqlc.arg(course_id);