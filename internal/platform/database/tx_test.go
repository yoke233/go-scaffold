package database

import (
	"context"
	"errors"
	"testing"

	"gorm.io/gorm"
)

func TestWithinTransactionReusesExistingTransaction(t *testing.T) {
	tx := &gorm.DB{}
	transactor := &Transactor{}

	var reused bool
	err := transactor.WithinTransaction(WithTx(context.Background(), tx), func(ctx context.Context) error {
		got, ok := FromContext(ctx)
		if !ok {
			t.Fatal("expected transaction in context")
		}
		if got != tx {
			t.Fatal("expected nested transaction to reuse outer transaction")
		}
		reused = true
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !reused {
		t.Fatal("expected callback to run")
	}
}

func TestWithinTransactionReturnsConfiguredErrorWhenDBMissing(t *testing.T) {
	transactor := &Transactor{}

	err := transactor.WithinTransaction(context.Background(), func(context.Context) error {
		return nil
	})
	if !errors.Is(err, ErrTransactorNotConfigured) {
		t.Fatalf("expected ErrTransactorNotConfigured, got %v", err)
	}
}

func TestUnitOfWorkDoReusesScopeFromExistingTransaction(t *testing.T) {
	tx := &gorm.DB{}
	uow := NewUnitOfWork(&Transactor{}, nil)

	var reused bool
	err := uow.Do(WithTx(context.Background(), tx), func(scope *Scope) error {
		if scope.DB() != tx {
			t.Fatal("expected unit of work to reuse outer transaction db")
		}
		if _, ok := FromContext(scope.Context()); !ok {
			t.Fatal("expected scope context to retain transaction")
		}
		reused = true
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !reused {
		t.Fatal("expected callback to run")
	}
}
