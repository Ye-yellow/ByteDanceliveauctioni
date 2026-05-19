package data

import (
	"context"
	"errors"

	biz "live-auction-bid/backend/app/auction/service/internal/biz"
)

var ErrMySQLRepositoryNotConfigured = errors.New("mysql lot repository is not configured")

type MySQLLotRepository struct{}

func NewMySQLLotRepository() *MySQLLotRepository { return &MySQLLotRepository{} }

func (r *MySQLLotRepository) Save(ctx context.Context, lot *biz.Lot) error {
	return ErrMySQLRepositoryNotConfigured
}
func (r *MySQLLotRepository) FindByID(ctx context.Context, id string) (*biz.Lot, error) {
	return nil, ErrMySQLRepositoryNotConfigured
}
func (r *MySQLLotRepository) FindLiveByRoom(ctx context.Context, roomID string) (*biz.Lot, error) {
	return nil, ErrMySQLRepositoryNotConfigured
}
func (r *MySQLLotRepository) List(ctx context.Context) ([]*biz.Lot, error) {
	return nil, ErrMySQLRepositoryNotConfigured
}
