package dbutils

import "database/sql"

type SQLTransactional interface {
	Get(dest interface{}, query string, args ...interface{}) error
	Select(dest interface{}, query string, args ...interface{}) error
	Exec(query string, args ...any) (sql.Result, error)
	Rebind(query string) string
}

// mock sql transactional that can be customized to return certain data on certain sql scripts
type MockSQLTransactional struct {
	GetFunc    func(dest interface{}, query string, args ...interface{}) error
	SelectFunc func(dest interface{}, query string, args ...interface{}) error
	ExecFunc   func(query string, args ...any) (sql.Result, error)
	RebindFunc func(query string) string
}

// NewMockSQLTransactional creates a new instance of MockSQLTransactional with default implementations
func NewMockSQLTransactional() *MockSQLTransactional {
	return &MockSQLTransactional{
		GetFunc:    func(dest interface{}, query string, args ...interface{}) error { return nil },
		SelectFunc: func(dest interface{}, query string, args ...interface{}) error { return nil },
		ExecFunc:   func(query string, args ...any) (sql.Result, error) { return nil, nil },
		RebindFunc: func(query string) string { return query },
	}
}

// Get implements SQLTransactional.
func (m *MockSQLTransactional) Get(dest interface{}, query string, args ...interface{}) error {
	return m.GetFunc(dest, query, args...)
}

// Select implements SQLTransactional.
func (m *MockSQLTransactional) Select(dest interface{}, query string, args ...interface{}) error {
	return m.SelectFunc(dest, query, args...)
}

// Exec implements SQLTransactional.
func (m *MockSQLTransactional) Exec(query string, args ...any) (sql.Result, error) {
	return m.ExecFunc(query, args...)
}

// Rebind implements SQLTransactional.
func (m *MockSQLTransactional) Rebind(query string) string {
	return m.RebindFunc(query)
}
