package storage

import (
	"fmt"
)

type (
	NotFoundError struct {
		Err  error
	}
	AlreadyExistsError struct {
		Err       error
	}
	StatementPSQLError struct {
		Err error
	}
	ExecutionPSQLError struct {
		Err error
	}
)

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s: not found in storage\n", e.Err.Error())
}

func (e *AlreadyExistsError) Error() string {
	return fmt.Sprintf("%s: already exists in storage\n", e.Err.Error())
}

func (e *StatementPSQLError) Error() string {
	return fmt.Sprintf("%s: could not compile statement\n", e.Err.Error())
}

func (e *ExecutionPSQLError) Error() string {
	return fmt.Sprintf("%s: could not query", e.Err.Error())
}

func (e *NotFoundError) Unwrap() error {
	return e.Err
}

func (e *AlreadyExistsError) Unwrap() error {
	return e.Err
}

func (e *StatementPSQLError) Unwrap() error {
	return e.Err
}

func (e *ExecutionPSQLError) Unwrap() error {
	return e.Err
}
