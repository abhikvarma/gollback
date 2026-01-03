package gollback

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Conn interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, arguments ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, arguments ...any) pgx.Row
}

type ConnGetter func(ctx context.Context) Conn

type TxnOptions struct {
	Timeout  time.Duration
	ReadOnly bool
}

type TxnOption func(*TxnOptions)

func WithTimeout(d time.Duration) TxnOption {
	return func(opts *TxnOptions) {
		opts.Timeout = d
	}
}

func ReadOnly() TxnOption {
	return func(opts *TxnOptions) {
		opts.ReadOnly = true
	}
}

type TxnManager interface {
	RunInTxn(ctx context.Context, fn func(ctx context.Context) error, opts ...TxnOption) error
}

type txnKey struct{}

type txnProvider struct {
	pool *pgxpool.Pool
}

func NewTxnProvider(pool *pgxpool.Pool) (TxnManager, ConnGetter) {
	tp := &txnProvider{pool: pool}
	return tp, func(ctx context.Context) Conn {
		if txn, ok := ctx.Value(txnKey{}).(pgx.Tx); ok {
			return txn
		}
		return pool
	}
}

func (tp *txnProvider) RunInTxn(ctx context.Context, fn func(ctx context.Context) error, opts ...TxnOption) error {
	if _, ok := ctx.Value(txnKey{}).(pgx.Tx); ok {
		slog.Warn("RunInTxn called inside existing transaction, reusing")
		return fn(ctx)
	}

	options := &TxnOptions{}
	for _, opt := range opts {
		opt(options)
	}

	if options.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, options.Timeout)
		defer cancel()
	}

	txOpts := pgx.TxOptions{}
	if options.ReadOnly {
		txOpts.AccessMode = pgx.ReadOnly
	}

	txn, err := tp.pool.BeginTx(ctx, txOpts)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			slog.Warn("transaction begin failed due to time out")
		}
		return TxnBeginError{Err: err}
	}

	ctx = context.WithValue(ctx, txnKey{}, txn)

	if err := fn(ctx); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			slog.Warn("transaction closing due to timeout")
		}

		rbCtx, rbCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer rbCancel()

		if rbErr := txn.Rollback(rbCtx); rbErr != nil {
			return TxnRollbackError{RollBackErr: rbErr, Cause: err}
		}
		return err
	}

	if err := txn.Commit(ctx); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			slog.Warn("transaction commit failed due to time out")
		}
		return TxnCommitError{Err: err}
	}
	return nil
}
