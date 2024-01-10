package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

/**
Store 结构体封装了数据库操作，特别是那些需要在事务中执行的操作。
execTx 方法是 Store 的核心，它允许你在一个事务中安全地执行多个数据库操作。
如果中途出现错误，它会回滚事务，这意味着所有在事务中进行的更改都不会应用到数据库。
*/

type Store interface {
	Querier
	TransferTx(ctx context.Context, arg TransferTxParams) (TransferTxResult, error)
	CreateUserTx(ctx context.Context, arg CreateUserTxParams) (CreateUserTxResult, error)
	VerifyEmailTx(ctx context.Context, arg VerifyEmailTxParams) (VerifyEmailTxResult, error)
}

// SQLStore provides all functions to execute SQL queries and transactions
// Store 是一个结构体，它嵌入了 Queries 结构体（这是由 sqlc 自动生成的，提供了一系列与数据库交互的方法）。
// 它还包含了一个指向 sql.DB 的指针，sql.DB 是 Go 标准库中的一个结构体，用于表示数据库连接。
type SQLStore struct {
	connPool *pgxpool.Pool
	*Queries
}

// NewStore 是一个函数，它创建并返回一个新的 Store 实例。
// 它接受一个 *sql.DB（数据库连接）作为参数，并用这个连接初始化 Store 结构体中的 db 字段和 Queries 字段。
// NewStore creates a new store
func NewStore(connPool *pgxpool.Pool) Store {
	return &SQLStore{
		connPool: connPool,
		Queries:  New(connPool),
	}
}
