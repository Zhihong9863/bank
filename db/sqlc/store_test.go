package db

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTransferTx(t *testing.T) {

	account1 := createRandomAccount(t)
	account2 := createRandomAccount(t)
	fmt.Println(">> before:", account1.Balance, account2.Balance)

	//run n concurrent transfer transaction
	n := 5
	amount := int64(10)

	errs := make(chan error)
	results := make(chan TransferTxResult)

	// run n concurrent transfer transaction
	for i := 0; i < n; i++ {
		go func() {
			result, err := testStore.TransferTx(context.Background(), TransferTxParams{
				FromAccountID: account1.ID,
				ToAccountID:   account2.ID,
				Amount:        amount,
			})

			errs <- err
			results <- result
		}()
	}

	//check results
	existed := make(map[int]bool)

	for i := 0; i < n; i++ {
		err := <-errs
		require.NoError(t, err)

		result := <-results
		require.NotEmpty(t, result)

		// check transfer
		transfer := result.Transfer
		require.NotEmpty(t, transfer)
		require.Equal(t, account1.ID, transfer.FromAccountID)
		require.Equal(t, account2.ID, transfer.ToAccountID)
		require.Equal(t, amount, transfer.Amount)
		require.NotZero(t, transfer.ID)
		require.NotZero(t, transfer.CreatedAt)

		_, err = testStore.GetTransfer(context.Background(), transfer.ID)
		require.NoError(t, err)

		// check entries
		fromEntry := result.FromEntry
		require.NotEmpty(t, fromEntry)
		require.Equal(t, account1.ID, fromEntry.AccountID)
		require.Equal(t, -amount, fromEntry.Amount)
		require.NotZero(t, fromEntry.ID)
		require.NotZero(t, fromEntry.CreatedAt)

		_, err = testStore.GetEntry(context.Background(), fromEntry.ID)
		require.NoError(t, err)

		toEntry := result.ToEntry
		require.NotEmpty(t, toEntry)
		require.Equal(t, account2.ID, toEntry.AccountID)
		require.Equal(t, amount, toEntry.Amount)
		require.NotZero(t, toEntry.ID)
		require.NotZero(t, toEntry.CreatedAt)

		_, err = testStore.GetEntry(context.Background(), toEntry.ID)
		require.NoError(t, err)

		// check accounts
		fromAccount := result.FromAccount
		require.NotEmpty(t, fromAccount)
		require.Equal(t, account1.ID, fromAccount.ID)

		toAccount := result.ToAccount
		require.NotEmpty(t, toAccount)
		require.Equal(t, account2.ID, toAccount.ID)

		// check balances
		fmt.Println(">> tx:", fromAccount.Balance, toAccount.Balance)

		diff1 := account1.Balance - fromAccount.Balance
		diff2 := toAccount.Balance - account2.Balance
		require.Equal(t, diff1, diff2)
		require.True(t, diff1 > 0)
		require.True(t, diff1%amount == 0) // 1 * amount, 2 * amount, 3 * amount, ..., n * amount

		k := int(diff1 / amount)
		require.True(t, k >= 1 && k <= n)
		require.NotContains(t, existed, k)
		existed[k] = true
	}

	// check the final updated balance
	updatedAccount1, err := testStore.GetAccount(context.Background(), account1.ID)
	require.NoError(t, err)

	updatedAccount2, err := testStore.GetAccount(context.Background(), account2.ID)
	require.NoError(t, err)

	fmt.Println(">> after:", updatedAccount1.Balance, updatedAccount2.Balance)

	require.Equal(t, account1.Balance-int64(n)*amount, updatedAccount1.Balance)
	require.Equal(t, account2.Balance+int64(n)*amount, updatedAccount2.Balance)
}

func TestTransferTxDeadlock(t *testing.T) {
	//测试首先通过 NewStore 函数创建了一个新的 Store 实例，这是后续所有数据库操作的起点。
	//然后创建了两个随机账户 account1 和 account2，并打印出它们的初始余额。

	account1 := createRandomAccount(t)
	account2 := createRandomAccount(t)
	fmt.Println(">> before:", account1.Balance, account2.Balance)

	//测试设置了 n 和 amount 变量来定义将执行的并发转账事务的数量（n）和每次转账的金额（amount）。
	//errs 是一个错误通道，用于从并发的goroutine中收集错误。
	n := 10
	amount := int64(10)
	//Channels: 使用 make(chan error) 创建了一个错误通道，它用于在goroutines之间安全地传递错误信息。
	errs := make(chan error)

	/*
		测试使用一个for循环创建 n 个goroutines来模拟并发转账。
		每个goroutine中调用 store.TransferTx，这是实际执行转账操作的函数。
		if i%2 == 1 这一行确保了每个偶数的迭代（i 为 0, 2, 4, ...）都是从 account1 到 account2 的转账，
		而每个奇数的迭代（i 为 1, 3, 5, ...）则相反，从 account2 到 account1，
		这样模拟了更真实的并发场景，其中转账方向是交替的。
	*/
	for i := 0; i < n; i++ {
		fromAccountID := account1.ID
		toAccountID := account2.ID

		if i%2 == 1 {
			fromAccountID = account2.ID
			toAccountID = account1.ID
		}

		//Goroutines（并发执行）: 使用 go 关键字来启动新的goroutines，它们是轻量级的线程，用于并发执行代码。
		//Context: 使用 context.Background() 创建了一个空的上下文，这通常用于在没有特定上下文（如取消或超时）的情况下开始一个操作。
		go func() {
			_, err := testStore.TransferTx(context.Background(), TransferTxParams{
				FromAccountID: fromAccountID,
				ToAccountID:   toAccountID,
				Amount:        amount,
			})

			errs <- err
		}()
	}

	//check results
	//在所有的goroutines启动后，一个for循环等待并收集所有goroutine的错误信息。
	//使用 require.NoError(t, err) 来断言没有发生错误。如果发生了错误，测试将会失败。
	for i := 0; i < n; i++ {
		err := <-errs
		require.NoError(t, err)
	}

	// check the final updated balance
	//在所有goroutine完成后，测试检查两个账户的最终余额，确保它们与初始余额相同，
	//因为所有的转账操作都是在两个账户之间循环进行的，所以理论上最终余额应该未发生变化。
	//使用 require.Equal(t, account1.Balance, updatedAccount1.Balance) 断言最终余额与初始余额相等。
	updatedAccount1, err := testStore.GetAccount(context.Background(), account1.ID)
	require.NoError(t, err)

	updatedAccount2, err := testStore.GetAccount(context.Background(), account2.ID)
	require.NoError(t, err)

	fmt.Println(">> after:", updatedAccount1.Balance, updatedAccount2.Balance)
	require.Equal(t, account1.Balance, updatedAccount1.Balance)
	require.Equal(t, account2.Balance, updatedAccount2.Balance)
}

/**
测试事务操作：
store_test.go 包含了一系列测试用例，这些用例专门用于测试 Store 结构体中定义的方法，特别是那些涉及事务的操作。

验证 TransferTx 方法的正确性：
测试用例如 TestTransferTx 验证 TransferTx 方法是否正确实现了资金转账的业务逻辑，包括检查转账后的账户余额、转账记录的正确性等。

模拟并发场景：
在 TestTransferTx 中，测试用例模拟了并发执行多个转账事务的场景，这是检查 Store 方法在高并发情况下能否正确管理事务的重要测试。

自动化验证：
这些测试自动化地验证了数据库操作的正确性，确保了代码更改或数据库模式更新后，应用程序的核心功能仍然按预期工作。
*/
