package user

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"project/gen/model"
	userv1 "project/gen/user/v1"
	"project/internal/platform/database"

	"gorm.io/gorm"
)

type fakeUserRepo struct {
	exists       bool
	existsErr    error
	createResult *model.User
	createErr    error
}

func (r *fakeUserRepo) ExistsByEmail(context.Context, string) (bool, error) {
	return r.exists, r.existsErr
}

func (r *fakeUserRepo) Create(context.Context, string, string) (*model.User, error) {
	return r.createResult, r.createErr
}

func (r *fakeUserRepo) GetByID(context.Context, int64) (*model.User, error) {
	return nil, nil
}

type fakeWalletQuery struct{}

func (fakeWalletQuery) GetBalanceByUserID(context.Context, int64) (int64, error) {
	return 0, nil
}

type fakeWalletWriter struct {
	called bool
	userID int64
	err    error
}

func (w *fakeWalletWriter) CreateByUserID(_ context.Context, userID int64) error {
	w.called = true
	w.userID = userID
	return w.err
}

type fakeUnitOfWork struct {
	called bool
	err    error
}

func (r *fakeUnitOfWork) Do(ctx context.Context, fn func(scope *database.Scope) error) error {
	r.called = true
	if r.err != nil {
		return r.err
	}
	return fn(database.NewScope(ctx, nil))
}

func TestCreateUserRunsInsideTransaction(t *testing.T) {
	uow := &fakeUnitOfWork{}
	repo := &fakeUserRepo{
		createResult: &model.User{ID: 1, Name: "alice", Email: "alice@example.com"},
	}
	writer := &fakeWalletWriter{}
	uc := &UseCase{
		uow: uow,
		repoFactory: func(_ *gorm.DB) userRepo {
			return repo
		},
		walletQuery:  fakeWalletQuery{},
		walletWriter: writer,
		logger:       slog.Default(),
	}

	resp, err := uc.CreateUser(context.Background(), &userv1.CreateUserRequest{
		Name:  "alice",
		Email: "alice@example.com",
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !uow.called {
		t.Fatal("expected transaction runner to be called")
	}
	if resp.Id != 1 {
		t.Fatalf("expected id 1, got %d", resp.Id)
	}
	if !writer.called || writer.userID != 1 {
		t.Fatalf("expected wallet writer to be called with user id 1, got called=%v userID=%d", writer.called, writer.userID)
	}
}

func TestCreateUserPropagatesTransactionError(t *testing.T) {
	uow := &fakeUnitOfWork{err: errors.New("tx failed")}
	uc := &UseCase{
		uow: uow,
		repoFactory: func(_ *gorm.DB) userRepo {
			return &fakeUserRepo{}
		},
		walletQuery:  fakeWalletQuery{},
		walletWriter: &fakeWalletWriter{},
		logger:       slog.Default(),
	}

	_, err := uc.CreateUser(context.Background(), &userv1.CreateUserRequest{})
	if err == nil || err.Error() != "tx failed" {
		t.Fatalf("expected transaction error, got %v", err)
	}
}

func TestCreateUserPropagatesWalletInitError(t *testing.T) {
	uc := &UseCase{
		uow: &fakeUnitOfWork{},
		repoFactory: func(_ *gorm.DB) userRepo {
			return &fakeUserRepo{
				createResult: &model.User{ID: 9, Name: "bob", Email: "bob@example.com"},
			}
		},
		walletQuery:  fakeWalletQuery{},
		walletWriter: &fakeWalletWriter{err: errors.New("wallet init failed")},
		logger:       slog.Default(),
	}

	_, err := uc.CreateUser(context.Background(), &userv1.CreateUserRequest{
		Name:  "bob",
		Email: "bob@example.com",
	})
	if err == nil || err.Error() != "wallet init failed" {
		t.Fatalf("expected wallet init error, got %v", err)
	}
}
