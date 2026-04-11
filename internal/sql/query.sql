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
select coalesce(sum(accrual),0)::BIGINT as sum_order_number from orders 
where user_id = $1
AND status = 'PROCESSED';

-- name: GetWithdraws :one
SELECT COALESCE(SUM(withdraw), 0)::BIGINT as sum_withdraw
FROM withdraw
WHERE user_id = $1;

-- name: WithdrawDB :one
SELECT WITHDRAW(
    input_order_number := $1,
    input_withdraw := $2,
    input_user_id := $3);

-- name: GetAllWithdrawals :many
SELECT order_number, withdraw, processed_at FROM withdraw
WHERE user_id = $1;    