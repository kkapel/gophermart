-- name: GetAllUsers :many
SELECT * FROM USERS;

-- name: SaveUser :one
INSERT INTO USERS (login, pass_hash) VALUES ($1, $2)
RETURNING id;

-- name: GetPassword :one
SELECT pass_hash FROM USERS WHERE login = $1;

-- name: SaveOrder :one
INSERT INTO ORDERS(order_number, status, uploaded_at)
VALUES($1, $2, $3)
RETURNING id;