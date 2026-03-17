package ports

import "context"

type WalletQuery interface {
	GetBalanceByUserID(ctx context.Context, userID int64) (int64, error)
}
