package user

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

func (r *Repo) Create(ctx context.Context, name, email string) (*model.User, error) {
	u := &model.User{Name: name, Email: email}
	err := r.q.User.WithContext(ctx).Create(u)
	return u, err
}

func (r *Repo) GetByID(ctx context.Context, id int64) (*model.User, error) {
	return r.q.User.WithContext(ctx).Where(r.q.User.ID.Eq(id)).First()
}

func (r *Repo) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	count, err := r.q.User.WithContext(ctx).Where(r.q.User.Email.Eq(email)).Count()
	return count > 0, err
}
