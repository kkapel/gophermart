-- name: GetAllUsers :many
SELECT * FROM USERS;

-- name: SaveUser :exec
INSERT INTO USERS (login, pass_hash) VALUES ($1, $2);