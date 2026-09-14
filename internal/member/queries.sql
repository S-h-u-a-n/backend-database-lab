-- name: CreateMember :one
INSERT INTO members(name, email)
VALUES (sqlc.arg(name), sqlc.arg(email))
RETURNING id, name, email, joined_at;

-- name: GetMember :one
SELECT id, name, email, joined_at FROM members
WHERE id = sqlc.arg(id);

-- name: ListMembers :many
SELECT id, name, email, joined_at FROM members
ORDER BY joined_at ASC, id ASC;

-- name: UpdateMember :one
UPDATE members
SET
    name = sqlc.arg(name),
    email = sqlc.arg(email)
WHERE id = sqlc.arg(id)
RETURNING id, name, email, joined_at;

-- name: DeleteMember :one
DELETE FROM members
WHERE id = sqlc.arg(id)
RETURNING id, name, email, joined_at;

-- name: ListMemberCourses :many
SELECT e.course_id AS course_id,
       c.title AS course_title,
       e.status AS status,
       e.enrolled_at AS enrolled_at
FROM members m
JOIN enrollments e
    ON m.id = e.member_id
JOIN courses c
    ON e.course_id = c.id
WHERE m.id = sqlc.arg(member_id)
ORDER BY c.title ASC, c.id ASC;

-- name: ListClassmates :many
SELECT DISTINCT m.id AS member_id,
                m.name AS member_name,
                m.email AS member_email
FROM enrollments e1
JOIN enrollments e2
    ON e2.course_id = e1.course_id
JOIN members m
    ON m.id = e2.member_id
WHERE
    e1.member_id = sqlc.arg(member_id)
    AND e1.status = 'enrolled'
    AND e2.status = 'enrolled'
    AND m.id <> e1.member_id
ORDER BY m.name ASC, m.id ASC;
