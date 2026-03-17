package user

import (
	"context"
	"log/slog"

	userv1 "project/gen/user/v1"
	"project/internal/domain/ports"
)

type UseCase struct {
	repo        *Repo
	walletQuery ports.WalletQuery
	logger      *slog.Logger
}

func NewUseCase(repo *Repo, wq ports.WalletQuery, logger *slog.Logger) *UseCase {
	return &UseCase{repo: repo, walletQuery: wq, logger: logger}
}

func (uc *UseCase) CreateUser(ctx context.Context, req *userv1.CreateUserRequest) (*userv1.CreateUserResponse, error) {
	exists, err := uc.repo.ExistsByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, userv1.ErrorErrorReasonUserAlreadyExists("email %s", req.Email)
	}

	u, err := uc.repo.Create(ctx, req.Name, req.Email)
	if err != nil {
		return nil, err
	}

	return &userv1.CreateUserResponse{
		Id:    u.ID,
		Name:  u.Name,
		Email: u.Email,
	}, nil
}

func (uc *UseCase) GetUser(ctx context.Context, req *userv1.GetUserRequest) (*userv1.GetUserResponse, error) {
	u, err := uc.repo.GetByID(ctx, req.Id)
	if err != nil {
		return nil, userv1.ErrorErrorReasonUserNotFound("user %d", req.Id)
	}

	// Cross-domain call: get wallet balance via ports interface
	balance, _ := uc.walletQuery.GetBalanceByUserID(ctx, u.ID)

	return &userv1.GetUserResponse{
		Id:            u.ID,
		Name:          u.Name,
		Email:         u.Email,
		WalletBalance: balance,
	}, nil
}
