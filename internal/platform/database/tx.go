package database

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"

	"gorm.io/gorm"
)

type txContextKey struct{}

type txState struct {
	db *gorm.DB
	id uint64
}

type Transactor struct {
	db *gorm.DB
}

type Scope struct {
	ctx context.Context
	db  *gorm.DB
}

type UnitOfWork struct {
	tx     *Transactor
	logger *slog.Logger
}

var ErrTransactorNotConfigured = errors.New("database transactor is not configured")
var txSequence atomic.Uint64

func NewTransactor(db *gorm.DB) *Transactor {
	return &Transactor{db: db}
}

func NewUnitOfWork(tx *Transactor, logger *slog.Logger) *UnitOfWork {
	if logger == nil {
		logger = slog.Default()
	}
	return &UnitOfWork{
		tx:     tx,
		logger: logger.With("component", "database.uow"),
	}
}

func NewScope(ctx context.Context, db *gorm.DB) *Scope {
	return &Scope{ctx: ctx, db: db}
}

func (t *Transactor) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := FromContext(ctx); ok {
		return fn(ctx)
	}
	if t == nil || t.db == nil {
		return ErrTransactorNotConfigured
	}
	return t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(withTxState(ctx, tx, txSequence.Add(1)))
	})
}

func (u *UnitOfWork) Do(ctx context.Context, fn func(scope *Scope) error) error {
	if state, ok := fromState(ctx); ok {
		u.logger.Debug("tx.reuse", "tx_id", state.id)
		return fn(NewScope(ctx, state.db))
	}

	var txID uint64
	err := u.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		state, ok := fromState(txCtx)
		if !ok {
			u.logger.Warn("tx.scope_missing")
			return fn(NewScope(txCtx, nil))
		}
		txID = state.id
		u.logger.Debug("tx.begin", "tx_id", txID)
		err := fn(NewScope(txCtx, state.db))
		if err != nil {
			u.logger.Warn("tx.rollback", "tx_id", txID, "error", err)
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	u.logger.Debug("tx.commit", "tx_id", txID)
	return nil
}

func (s *Scope) Context() context.Context {
	return s.ctx
}

func (s *Scope) DB() *gorm.DB {
	return s.db
}

func Use[T any](scope *Scope, factory func(*gorm.DB) T) T {
	return factory(scope.db)
}

func WithTx(ctx context.Context, tx *gorm.DB) context.Context {
	if state, ok := fromState(ctx); ok {
		return withTxState(ctx, tx, state.id)
	}
	return withTxState(ctx, tx, 0)
}

func DB(ctx context.Context, fallback *gorm.DB) *gorm.DB {
	if tx, ok := FromContext(ctx); ok {
		return tx
	}
	return fallback
}

func FromContext(ctx context.Context) (*gorm.DB, bool) {
	state, ok := fromState(ctx)
	if !ok {
		return nil, false
	}
	return state.db, true
}

func withTxState(ctx context.Context, tx *gorm.DB, id uint64) context.Context {
	return context.WithValue(ctx, txContextKey{}, txState{
		db: tx,
		id: id,
	})
}

func fromState(ctx context.Context) (txState, bool) {
	state, ok := ctx.Value(txContextKey{}).(txState)
	return state, ok
}
