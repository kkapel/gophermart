-- name: GetAllUsers :many
SELECT * FROM USERS;

-- name: SaveUser :one
INSERT INTO USERS (login, pass_hash) VALUES ($1, $2)
RETURNING id;