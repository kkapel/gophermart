-- name: GetAllUsers :many
SELECT * FROM USERS;

-- name: SaveUser :one
INSERT INTO USERS (login, pass_hash) VALUES ($1, $2)
RETURNING id;

-- name: GetPassword :many
SELECT pass_hash, id FROM USERS WHERE login = $1;

-- name: SaveOrder :one
INSERT INTO ORDERS(order_number, status, uploaded_at, user_id)
VALUES($1, $2, $3, $4)
RETURNING id;

-- name: GetUserIDByOrder :one
SELECT user_id FROM ORDERS WHERE order_number = $1;

-- name: GetOrdersByUsers :many
SELECT order_number, status, uploaded_at, accrual FROM orders WHERE user_id = $1;

-- name: GetOrdersForAccrual :many
SELECT order_number FROM orders WHERE status = $1
order by uploaded_at
LIMIT $2;

-- name: UpdateOrderStatus :one
UPDATE orders SET status = $1,
accrual = $2
WHERE order_number = $3
RETURNING id;

-- name: GetBalance :one
select sum(accrual) as sum_order_number from orders o where user_id = $1;

-- name: GetWithdraws :one
SELECT COALESCE(SUM(withdraw), 0)::BIGINT as sum_withdraw
FROM withdraw
WHERE order_number IN (SELECT order_number FROM ORDERS WHERE user_id = $1);