package db

import "context"

// TransferTxParams contains the input parameters of the transfer transaction
type TransferTxParams struct {
	FromAccountID int64 `json:"from_account_id"`
	ToAccountID   int64 `json:"to_account_id"`
	Amount        int64 `json:"amount"`
}

// TransferTxResult is the result of the transfer transaction
type TransferTxResult struct {
	Transfer    Transfer `json:"transfer"`
	FromAccount Account  `json:"from_account"`
	ToAccount   Account  `json:"to_account"`
	FromEntry   Entry    `json:"from_entry"`
	ToEntry     Entry    `json:"to_entry"`
}

// TransferTx performs a money transfer from one account to the other.
// It creates the transfer, add account entries, and update accounts' balance within a database transaction
//创建一个转账记录。
//在转出账户创建一个负金额的条目（表示资金被取出）。
//在转入账户创建一个正金额的条目（表示资金被存入）。

// ctx context.Context: 用于传递上下文信息，比如请求的截止时间、取消信号等。
// arg TransferTxParams: 包含执行转账所需的参数，如转出账户ID、转入账户ID和转账金额。
// 返回值：TransferTxResult 包含转账操作的结果，和一个 error 值表示可能发生的错误。
func (store *SQLStore) TransferTx(ctx context.Context, arg TransferTxParams) (TransferTxResult, error) {
	var result TransferTxResult

	//启动事务：
	//TransferTx 使用 execTx 方法来确保所有步骤在一个事务中执行。如果任何步骤失败，整个事务会被回滚。
	err := store.execTx(ctx, func(q *Queries) error {
		var err error

		//创建转账记录：
		//使用 CreateTransfer 方法（由 sqlc 自动生成）创建一个转账记录。这个记录包含了转出账户、转入账户和转账金额。
		result.Transfer, err = q.CreateTransfer(ctx, CreateTransferParams{
			FromAccountID: arg.FromAccountID,
			ToAccountID:   arg.ToAccountID,
			Amount:        arg.Amount,
		})
		if err != nil {
			return err
		}

		//创建账户条目：
		//使用 CreateEntry 方法（由 sqlc 自动生成）在转出账户创建一个负金额的条目，表示资金被取出。
		result.FromEntry, err = q.CreateEntry(ctx, CreateEntryParams{
			AccountID: arg.FromAccountID,
			Amount:    -arg.Amount,
		})
		if err != nil {
			return err
		}
		/*

			在进行资金转账时，通常涉及两个账户：一个是资金转出的账户，另一个是资金转入的账户。
			如果系统同时处理多个此类转账事务，且这些事务涉及相同的账户，就可能出现死锁。

			假设有两个并发的转账操作正在执行：

			操作 A：从账户 1 转账到账户 2。
			操作 B：从账户 2 转账到账户 1。
			如果没有一致的锁定顺序，可能会出现如下情况：

			操作 A 锁定了账户 1 并准备锁定账户 2。
			同时，操作 B 锁定了账户 2 并准备锁定账户 1。
			在这种情况下，操作 A 等待操作 B 释放账户 2 的锁，而操作 B 等待操作 A 释放账户 1 的锁。
			这就是死锁，因为它们都在等待对方释放资源，而没有任何一方可以继续执行。

			现在，假设我们实施了一条规则，无论什么操作，都要先锁定ID较小的账户。这样的话：

			操作 A 将先锁定账户 1（因为 1 < 2），然后锁定账户 2。
			操作 B 也将尝试先锁定账户 1（因为 1 < 2），但因为操作 A 已经锁定了账户 1，所以它必须等待。

			在这种情况下，操作 B 会等待操作 A 完成，并不会先锁定账户 2。操作 A 完成后，会释放账户 1 和账户 2 的锁。
			然后操作 B 可以锁定账户 1 和账户 2，继续执行它的转账操作。通过这种方式，我们确保了不会有两个操作互相等待对方释放锁的情况发生。
			始终按照相同的顺序获取锁意味着不存在循环等待条件
		*/

		result.ToEntry, err = q.CreateEntry(ctx, CreateEntryParams{
			AccountID: arg.ToAccountID,
			Amount:    arg.Amount,
		})
		if err != nil {
			return err
		}

		if arg.FromAccountID < arg.ToAccountID {
			result.FromAccount, result.ToAccount, err = addMoney(ctx, q, arg.FromAccountID, -arg.Amount, arg.ToAccountID, arg.Amount)
		} else {
			result.ToAccount, result.FromAccount, err = addMoney(ctx, q, arg.ToAccountID, arg.Amount, arg.FromAccountID, -arg.Amount)
		}

		return nil
	})

	//如果所有步骤都成功完成，TransferTx 返回一个包含所有操作结果的 TransferTxResult 结构体，以及 nil 错误。
	return result, err
}

func addMoney(
	ctx context.Context,
	q *Queries,
	accountID1 int64,
	amount1 int64,
	accountID2 int64,
	amount2 int64,
) (account1 Account, account2 Account, err error) {
	account1, err = q.AddAccountBalance(ctx, AddAccountBalanceParams{
		ID:     accountID1,
		Amount: amount1,
	})
	if err != nil {
		return
	}

	account2, err = q.AddAccountBalance(ctx, AddAccountBalanceParams{
		ID:     accountID2,
		Amount: amount2,
	})
	return
}

/**
定义 Store 结构体：
Store 结构体封装了数据库交互的逻辑，特别是处理需要在数据库事务中执行的操作。它利用了 sqlc 自动生成的 Queries 结构体，使得数据库操作更加方便和安全。

execTx 方法
execTx 方法的主要职责是提供一个安全的方式来执行包裹在事务中的数据库操作。它确保了在事务中运行的所有数据库操作要么全部成功，要么在遇到错误时全部撤销。
TransferTx 方法
TransferTx 方法用于执行资金转账的业务逻辑，这包括创建转账记录、创建账户条目，以及更新账户余额等操作。所有这些步骤需要在同一个事务中完成，以保证数据的一致性和完整性。

  TransferTx 利用了 execTx 提供的事务管理功能来确保转账过程的完整性和一致性。
它将转账的所有步骤封装在一个单独的函数中，并将这个函数传递给 execTx。
execTx 负责管理事务的生命周期，包括开始事务、提交事务或在遇到错误时回滚事务。

  通过这种方式，TransferTx 可以专注于实现业务逻辑，而不必担心事务的具体细节，因为这些都由 execTx 负责管理。
这使得代码更加清晰、模块化，且易于维护。同时，它还提高了数据库操作的可靠性，因为所有操作要么一起成功，要么在遇到问题时一起撤销。
*/
