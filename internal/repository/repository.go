package repository

import (
	"context"
	"gophermart/internal/db"
)

type Repository struct {
	q db.Querier
}

func NewRepository(querier db.Querier) *Repository {
	return &Repository{
		q: querier,
	}
}

func (r *Repository) GetAllUsers(ctx context.Context) ([]db.User, error) {
	return r.q.GetAllUsers(ctx)
}

func (r *Repository) GetAllWithdrawals(ctx context.Context, userID int32) ([]db.GetAllWithdrawalsRow, error) {
	return r.q.GetAllWithdrawals(ctx, userID)
}

func (r *Repository) GetBalance(ctx context.Context, userID int32) (int64, error) {
	return r.q.GetBalance(ctx, userID)
}

func (r *Repository) GetOrdersByUsers(ctx context.Context, userID int32) ([]db.GetOrdersByUsersRow, error) {
	return r.q.GetOrdersByUsers(ctx, userID)
}

func (r *Repository) GetOrdersForAccrual(ctx context.Context, arg db.GetOrdersForAccrualParams) ([]string, error) {
	return r.q.GetOrdersForAccrual(ctx, arg)
}

func (r *Repository) GetPassword(ctx context.Context, login string) ([]db.GetPasswordRow, error) {
	return r.q.GetPassword(ctx, login)
}

func (r *Repository) GetUserIDByOrder(ctx context.Context, orderNumber string) (int32, error) {
	return r.q.GetUserIDByOrder(ctx, orderNumber)
}

func (r *Repository) GetWithdraws(ctx context.Context, userID int32) (int64, error) {
	return r.q.GetWithdraws(ctx, userID)
}

func (r *Repository) SaveOrder(ctx context.Context, arg db.SaveOrderParams) (int32, error) {
	return r.q.SaveOrder(ctx, arg)
}

func (r *Repository) SaveUser(ctx context.Context, arg db.SaveUserParams) (int32, error) {
	return r.q.SaveUser(ctx, arg)
}
func (r *Repository) UpdateOrderStatus(ctx context.Context, arg db.UpdateOrderStatusParams) (int32, error) {
	return r.q.UpdateOrderStatus(ctx, arg)
}
func (r *Repository) WithdrawDB(ctx context.Context, arg db.WithdrawDBParams) (bool, error) {
	return r.q.WithdrawDB(ctx, arg)
}
