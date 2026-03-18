package wallet

import (
	"context"
	"log/slog"

	"project/gen/model"
	walletv1 "project/gen/wallet/v1"
	"project/internal/platform/database"

	"gorm.io/gorm"
)

type walletRepo interface {
	ExistsByUserID(ctx context.Context, userID int64) (bool, error)
	Create(ctx context.Context, userID int64) (*model.Wallet, error)
	GetByUserID(ctx context.Context, userID int64) (*model.Wallet, error)
}

type unitOfWork interface {
	Do(ctx context.Context, fn func(scope *database.Scope) error) error
}

type UseCase struct {
	repo        walletRepo
	uow         unitOfWork
	repoFactory func(db *gorm.DB) walletRepo
	logger      *slog.Logger
}

func NewUseCase(repo *Repo, uow *database.UnitOfWork, logger *slog.Logger) *UseCase {
	return &UseCase{
		repo: repo,
		uow:  uow,
		repoFactory: func(db *gorm.DB) walletRepo {
			return NewRepo(db)
		},
		logger: logger,
	}
}

func (uc *UseCase) CreateWallet(ctx context.Context, req *walletv1.CreateWalletRequest) (*walletv1.CreateWalletResponse, error) {
	var w *model.Wallet
	err := uc.uow.Do(ctx, func(scope *database.Scope) error {
		repo := database.Use(scope, uc.repoFactory)

		exists, err := repo.ExistsByUserID(scope.Context(), req.UserId)
		if err != nil {
			return err
		}
		if exists {
			return walletv1.ErrorErrorReasonWalletAlreadyExists("user %d", req.UserId)
		}

		w, err = repo.Create(scope.Context(), req.UserId)
		return err
	})
	if err != nil {
		return nil, err
	}
	return &walletv1.CreateWalletResponse{UserId: w.UserID, Balance: w.Balance}, nil
}

func (uc *UseCase) GetWallet(ctx context.Context, req *walletv1.GetWalletRequest) (*walletv1.GetWalletResponse, error) {
	w, err := uc.repo.GetByUserID(ctx, req.UserId)
	if err != nil {
		return nil, walletv1.ErrorErrorReasonWalletNotFound("user %d", req.UserId)
	}
	return &walletv1.GetWalletResponse{UserId: w.UserID, Balance: w.Balance}, nil
}
