-- name: ListNewsByTeam :many
-- Keyset page of a team's news items, pinned first then newest first. NULL
-- cursor args mean "first page" -- the NULL check makes this one static
-- query instead of the caller having to build the predicate conditionally.
SELECT n.id, n.team_id, n.author_id, n.title, n.body, n.pinned, n.created_at,
       u.name AS author_name, u.avatar_color AS author_color,
       (u.photo_object_key IS NOT NULL)::boolean AS has_photo
FROM news n
JOIN users u ON u.id = n.author_id
WHERE n.team_id = $1
  AND (
    sqlc.narg('cursor_id')::uuid IS NULL
    OR (n.pinned, n.created_at, n.id) < (
      sqlc.narg('cursor_pinned')::boolean,
      sqlc.narg('cursor_created_at')::timestamptz,
      sqlc.narg('cursor_id')::uuid
    )
  )
ORDER BY n.pinned DESC, n.created_at DESC, n.id DESC
LIMIT $2;

-- name: CountNewsByTeam :one
SELECT COUNT(*) FROM news WHERE team_id = $1;

-- name: CreateNews :one
INSERT INTO news (team_id, author_id, title, body, pinned)
VALUES ($1, $2, $3, $4, $5)
RETURNING id;

-- name: UpdateNews :execrows
UPDATE news SET
    title  = COALESCE(sqlc.narg('title'), title),
    body   = COALESCE(sqlc.narg('body'), body),
    pinned = COALESCE(sqlc.narg('pinned'), pinned)
WHERE id = $1 AND team_id = $2;

-- name: DeleteNews :execrows
DELETE FROM news WHERE id = $1 AND team_id = $2;

-- name: GetNewsByID :one
SELECT n.id, n.team_id, n.author_id, n.title, n.body, n.pinned, n.created_at,
       u.name AS author_name, u.avatar_color AS author_color,
       (u.photo_object_key IS NOT NULL)::boolean AS has_photo
FROM news n
JOIN users u ON u.id = n.author_id
WHERE n.id = $1;
