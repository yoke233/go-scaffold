package wallet

import (
	"context"

	"project/gen/model"
	"project/gen/query"

	"gorm.io/gorm"
)

type Repo struct {
	db *gorm.DB
	q  *query.Query
}

func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db, q: query.Use(db)}
}

func (r *Repo) GetByUserID(ctx context.Context, userID int64) (*model.Wallet, error) {
	return r.q.Wallet.WithContext(ctx).Where(r.q.Wallet.UserID.Eq(userID)).First()
}

func (r *Repo) Create(ctx context.Context, userID int64) (*model.Wallet, error) {
	w := &model.Wallet{UserID: userID, Balance: 0}
	err := r.q.Wallet.WithContext(ctx).Create(w)
	return w, err
}
