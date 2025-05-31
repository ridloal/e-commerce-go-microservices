package mocks

import (
	"context"
	"database/sql" // Make sure this import is present

	"github.com/stretchr/testify/mock"
)

type MockDBTX struct {
	mock.Mock
}

func (m *MockDBTX) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	// Create a new slice for arguments to m.Called, starting with ctx and query
	callArgs := make([]interface{}, 0, 2+len(args))
	callArgs = append(callArgs, ctx, query)
	callArgs = append(callArgs, args...)

	ret := m.Called(callArgs...)

	var r0 sql.Result
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(sql.Result)
	}
	return r0, ret.Error(1)
}

func (m *MockDBTX) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	args := m.Called(ctx, query)
	var r0 *sql.Stmt
	if args.Get(0) != nil {
		r0 = args.Get(0).(*sql.Stmt)
	}
	return r0, args.Error(1)
}

func (m *MockDBTX) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	callArgs := make([]interface{}, 0, 2+len(args))
	callArgs = append(callArgs, ctx, query)
	callArgs = append(callArgs, args...)

	ret := m.Called(callArgs...)

	var r0 *sql.Rows
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*sql.Rows)
	}
	return r0, ret.Error(1)
}

func (m *MockDBTX) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	callArgs := make([]interface{}, 0, 2+len(args))
	callArgs = append(callArgs, ctx, query)
	callArgs = append(callArgs, args...)

	ret := m.Called(callArgs...)

	var r0 *sql.Row
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*sql.Row)
	}
	// QueryRowContext is expected to return a non-nil *sql.Row, even on error.
	// The error is checked by calling .Scan() on the returned *sql.Row.
	// So, if ret.Get(0) is nil (meaning you didn't mock a specific *sql.Row to return),
	// you might still need to return a placeholder *sql.Row or ensure your mock setup always provides one.
	// For many tests, if this method is called, you'd provide a mock *sql.Row.
	return r0
}

func (m *MockDBTX) Commit() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockDBTX) Rollback() error {
	args := m.Called()
	return args.Error(0)
}
