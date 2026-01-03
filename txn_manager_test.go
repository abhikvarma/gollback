package gollback

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestWithTimeout_SetsOption(t *testing.T) {
	opts := &TxnOptions{}
	WithTimeout(5 * time.Second)(opts)

	if opts.Timeout != 5*time.Second {
		t.Errorf("expected Timeout=5s, got %v", opts.Timeout)
	}
}

func TestReadOnly_SetsOption(t *testing.T) {
	opts := &TxnOptions{}
	ReadOnly()(opts)

	if !opts.ReadOnly {
		t.Error("expected ReadOnly=true, got false")
	}
}

func TestTxnBeginError_Error(t *testing.T) {
	err := TxnBeginError{Err: errors.New("connection refused")}
	expected := "failed to begin transaction: connection refused"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestTxnBeginError_Unwrap(t *testing.T) {
	underlying := errors.New("connection refused")
	err := TxnBeginError{Err: underlying}
	if !errors.Is(err, underlying) {
		t.Error("Unwrap should return underlying error")
	}
}

func TestTxnCommitError_Error(t *testing.T) {
	err := TxnCommitError{Err: errors.New("commit failed")}
	expected := "failed to commit transaction: commit failed"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestTxnCommitError_Unwrap(t *testing.T) {
	underlying := errors.New("commit failed")
	err := TxnCommitError{Err: underlying}
	if !errors.Is(err, underlying) {
		t.Error("Unwrap should return underlying error")
	}
}

func TestTxnRollbackError_Error(t *testing.T) {
	err := TxnRollbackError{
		RollBackErr: errors.New("rollback failed"),
		Cause:       errors.New("original error"),
	}
	expected := "failed to rollback transaction: rollback failed, cause: original error"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestTxnRollbackError_Unwrap(t *testing.T) {
	cause := errors.New("original error")
	err := TxnRollbackError{
		RollBackErr: errors.New("rollback failed"),
		Cause:       cause,
	}
	if !errors.Is(err, cause) {
		t.Error("Unwrap should return cause error")
	}
}

func TestNewTxnProvider_ReturnsNonNil(t *testing.T) {
	pool := &pgxpool.Pool{}
	txnManager, connGetter := NewTxnProvider(pool)

	if txnManager == nil {
		t.Error("expected non-nil TxnManager")
	}
	if connGetter == nil {
		t.Error("expected non-nil ConnGetter")
	}
}

func TestConnGetter_ReturnsPool_WhenNoTxnInContext(t *testing.T) {
	pool := &pgxpool.Pool{}
	_, connGetter := NewTxnProvider(pool)

	ctx := context.Background()
	conn := connGetter(ctx)

	if conn == nil {
		t.Error("expected non-nil Conn")
	}
	if conn != pool {
		t.Error("expected ConnGetter to return pool when no txn in context")
	}
}
