package gollback

import "fmt"

type TxnBeginError struct {
	Err error
}

func (e TxnBeginError) Error() string {
	return fmt.Sprintf("failed to begin transaction: %v", e.Err)
}

func (e TxnBeginError) Unwrap() error {
	return e.Err
}

type TxnCommitError struct {
	Err error
}

func (e TxnCommitError) Error() string {
	return fmt.Sprintf("failed to commit transaction: %v", e.Err)
}

func (e TxnCommitError) Unwrap() error {
	return e.Err
}

type TxnRollbackError struct {
	RollBackErr error
	Cause       error
}

func (e TxnRollbackError) Error() string {
	return fmt.Sprintf("failed to rollback transaction: %v, cause: %v", e.RollBackErr, e.Cause)
}

func (e TxnRollbackError) Unwrap() error {
	return e.Cause
}
