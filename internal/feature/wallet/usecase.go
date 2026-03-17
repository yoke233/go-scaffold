package wallet

import (
	"context"
	"log/slog"

	walletv1 "project/gen/wallet/v1"
)

type UseCase struct {
	repo   *Repo
	logger *slog.Logger
}

func NewUseCase(repo *Repo, logger *slog.Logger) *UseCase {
	return &UseCase{repo: repo, logger: logger}
}

func (uc *UseCase) CreateWallet(ctx context.Context, req *walletv1.CreateWalletRequest) (*walletv1.CreateWalletResponse, error) {
	w, err := uc.repo.Create(ctx, req.UserId)
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
