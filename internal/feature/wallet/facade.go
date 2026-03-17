package wallet

import "context"

// Facade implements ports.WalletQuery, exposing wallet data to other domains.
type Facade struct {
	repo *Repo
}

func NewFacade(repo *Repo) *Facade {
	return &Facade{repo: repo}
}

func (f *Facade) GetBalanceByUserID(ctx context.Context, userID int64) (int64, error) {
	w, err := f.repo.GetByUserID(ctx, userID)
	if err != nil {
		return 0, err
	}
	return w.Balance, nil
}
