package db

import (
	"context"
	"database/sql"
	"fmt"
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
	db *sql.DB
	*Queries
}

// NewStore 是一个函数，它创建并返回一个新的 Store 实例。
// 它接受一个 *sql.DB（数据库连接）作为参数，并用这个连接初始化 Store 结构体中的 db 字段和 Queries 字段。
// NewStore creates a new store
func NewStore(db *sql.DB) Store {
	return &SQLStore{
		db:      db,
		Queries: New(db),
	}
}

// // execTx executes a function within a database transaction
// //execTx 是 Store 结构体的一个方法，其主要目的是在数据库事务的上下文中安全地执行一系列数据库操作。
// //ctx context.Context：这是 Go 语言的上下文对象，它通常用于控制函数的执行。
// //它可以用于设置超时、截止时间，或者在函数执行过程中传递取消信号。

// // fn func(*Queries) error：这是一个函数类型的参数。
// // 这个函数接受一个指向 Queries 结构体的指针，并返回一个错误。
// // 这意味着你可以传递任何这样的函数给 execTx，这个函数会在事务中执行一些数据库操作，
// // 并且这些操作要么全部成功，要么（在出错时）全部不执行。
func (store *SQLStore) execTx(ctx context.Context, fn func(*Queries) error) error {
	//这一行开始一个新的数据库事务。BeginTx 方法来自 Go 的 sql 包，用于在给定的上下文（ctx）中开始一个新的事务。
	//如果事务成功开始，它返回一个事务对象 tx。如果出现错误，如数据库连接问题，它返回一个错误。
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	//此处创建了一个新的 Queries 实例，它绑定到刚刚创建的事务 tx。这意味着使用 q 执行的所有数据库操作都是在这个事务的上下文中进行的。
	q := New(tx)
	//这一行执行传入的函数 fn，将之前创建的 q 作为参数传递给它。所有在 fn 中执行的数据库操作都会在事务 tx 中进行。如果 fn 中的操作成功，它返回 nil；如果有任何操作失败，它返回一个错误。
	err = fn(q)
	//如果 fn 执行成功（err == nil），则执行 tx.Commit()，这将提交事务，即所有在事务中的操作会被永久地写入数据库。
	//如果 fn 返回错误（err != nil），则执行 tx.Rollback()，这将撤销事务，即所有在事务中的操作都不会对数据库产生影响。
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("tx err: %v, rb err: %v", err, rbErr)
		}
		return err
	}

	return tx.Commit()
}
