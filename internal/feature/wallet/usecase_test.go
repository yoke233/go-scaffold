package wallet

import (
	"context"
	"log/slog"
	"testing"

	"project/gen/model"
	walletv1 "project/gen/wallet/v1"
	"project/internal/platform/database"

	"gorm.io/gorm"
)

type fakeWalletRepo struct {
	exists       bool
	existsErr    error
	createResult *model.Wallet
	createErr    error
}

func (r *fakeWalletRepo) ExistsByUserID(context.Context, int64) (bool, error) {
	return r.exists, r.existsErr
}

func (r *fakeWalletRepo) Create(context.Context, int64) (*model.Wallet, error) {
	return r.createResult, r.createErr
}

func (r *fakeWalletRepo) GetByUserID(context.Context, int64) (*model.Wallet, error) {
	return nil, nil
}

type fakeUnitOfWork struct {
	called bool
}

func (r *fakeUnitOfWork) Do(ctx context.Context, fn func(scope *database.Scope) error) error {
	r.called = true
	return fn(database.NewScope(ctx, nil))
}

func TestCreateWalletRunsInsideTransaction(t *testing.T) {
	uow := &fakeUnitOfWork{}
	repo := &fakeWalletRepo{
		createResult: &model.Wallet{UserID: 7, Balance: 0},
	}
	uc := &UseCase{
		uow: uow,
		repoFactory: func(_ *gorm.DB) walletRepo {
			return repo
		},
		logger: slog.Default(),
	}

	resp, err := uc.CreateWallet(context.Background(), &walletv1.CreateWalletRequest{UserId: 7})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !uow.called {
		t.Fatal("expected transaction runner to be called")
	}
	if resp.UserId != 7 {
		t.Fatalf("expected user id 7, got %d", resp.UserId)
	}
}

func TestCreateWalletRejectsDuplicateWallet(t *testing.T) {
	uc := &UseCase{
		uow: &fakeUnitOfWork{},
		repoFactory: func(_ *gorm.DB) walletRepo {
			return &fakeWalletRepo{exists: true}
		},
		logger: slog.Default(),
	}

	_, err := uc.CreateWallet(context.Background(), &walletv1.CreateWalletRequest{UserId: 7})
	if err == nil {
		t.Fatal("expected duplicate wallet error")
	}
}
