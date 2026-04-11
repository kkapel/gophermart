create or replace function WITHDRAW(
	input_order_number varchar,
	input_withdraw bigint,
	input_user_id integer 
)
returns boolean as $$
declare
	sum_accrual BIGINT;
	sum_withdraw BIGINT;
	difference_accrual BIGINT;
begin
	SELECT coalesce(sum(accrual), 0) into sum_accrual from orders
	WHERE user_id = input_user_id
	AND status = 'PROCESSED';

	SELECT coalesce(sum(withdraw), 0) into sum_withdraw from withdraw
	WHERE user_id = input_user_id;

	-- Вычисляем разницу
	difference_accrual = sum_accrual - sum_withdraw - input_withdraw;

	if difference_accrual < 0 then
		return false;
	end if;

	insert into withdraw (order_number, withdraw, processed_at, user_id)
	values (input_order_number, input_withdraw, CURRENT_TIMESTAMP, input_user_id);

	return true;
	
end;
$$ language plpgsql;
