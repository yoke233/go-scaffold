package user

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

func (r *Repo) Create(ctx context.Context, name, email string) (*model.User, error) {
	u := &model.User{Name: name, Email: email}
	err := r.query(ctx).User.WithContext(ctx).Create(u)
	return u, err
}

func (r *Repo) GetByID(ctx context.Context, id int64) (*model.User, error) {
	q := r.query(ctx)
	return q.User.WithContext(ctx).Where(q.User.ID.Eq(id)).First()
}

func (r *Repo) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	q := r.query(ctx)
	count, err := q.User.WithContext(ctx).Where(q.User.Email.Eq(email)).Count()
	return count > 0, err
}
