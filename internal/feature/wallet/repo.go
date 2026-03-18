package wallet

import (
	"context"

	"project/gen/model"
	"project/gen/query"
	"project/internal/platform/database"

	"gorm.io/gorm"
)

type Repo struct {
	db *gorm.DB
}

func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

func (r *Repo) query(ctx context.Context) *query.Query {
	return query.Use(database.DB(ctx, r.db))
}

func (r *Repo) GetByUserID(ctx context.Context, userID int64) (*model.Wallet, error) {
	q := r.query(ctx)
	return q.Wallet.WithContext(ctx).Where(q.Wallet.UserID.Eq(userID)).First()
}

func (r *Repo) ExistsByUserID(ctx context.Context, userID int64) (bool, error) {
	q := r.query(ctx)
	count, err := q.Wallet.WithContext(ctx).Where(q.Wallet.UserID.Eq(userID)).Count()
	return count > 0, err
}

func (r *Repo) Create(ctx context.Context, userID int64) (*model.Wallet, error) {
	w := &model.Wallet{UserID: userID, Balance: 0}
	err := r.query(ctx).Wallet.WithContext(ctx).Create(w)
	return w, err
}
