package user

import (
	"context"
	"log/slog"

	"project/gen/model"
	userv1 "project/gen/user/v1"
	"project/internal/domain/ports"
	"project/internal/platform/database"

	"gorm.io/gorm"
)

type userRepo interface {
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	Create(ctx context.Context, name, email string) (*model.User, error)
	GetByID(ctx context.Context, id int64) (*model.User, error)
}

type unitOfWork interface {
	Do(ctx context.Context, fn func(scope *database.Scope) error) error
}

type UseCase struct {
	repo         userRepo
	uow          unitOfWork
	repoFactory  func(db *gorm.DB) userRepo
	walletQuery  ports.WalletQuery
	walletWriter ports.WalletWriter
	logger       *slog.Logger
}

func NewUseCase(repo *Repo, uow *database.UnitOfWork, wq ports.WalletQuery, ww ports.WalletWriter, logger *slog.Logger) *UseCase {
	return &UseCase{
		repo: repo,
		uow:  uow,
		repoFactory: func(db *gorm.DB) userRepo {
			return NewRepo(db)
		},
		walletQuery:  wq,
		walletWriter: ww,
		logger:       logger,
	}
}

func (uc *UseCase) CreateUser(ctx context.Context, req *userv1.CreateUserRequest) (*userv1.CreateUserResponse, error) {
	var u *model.User
	err := uc.uow.Do(ctx, func(scope *database.Scope) error {
		repo := database.Use(scope, uc.repoFactory)

		exists, err := repo.ExistsByEmail(scope.Context(), req.Email)
		if err != nil {
			return err
		}
		if exists {
			return userv1.ErrorErrorReasonUserAlreadyExists("email %s", req.Email)
		}

		u, err = repo.Create(scope.Context(), req.Name, req.Email)
		if err != nil {
			return err
		}

		return uc.walletWriter.CreateByUserID(scope.Context(), u.ID)
	})
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
