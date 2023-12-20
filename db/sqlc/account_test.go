package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/techschool/bank/util"
)

/**
在每个测试用例中，require 是 testify 包的一个组件，用于断言测试期望。
例如，require.NoError(t, err) 确保没有错误发生，如果有，则测试失败。
require.Equal(t, expected, actual) 确保实际值与预期值相等，如果不相等，则测试失败。
*/

// 这是一个辅助函数，用于在测试中创建一个具有随机属性的账户。它使用 account.sql.go 中的 CreateAccount 方法来实际创建账户。
func createRandomAccount(t *testing.T) Account {

	user := createRandomUser(t)

	arg := CreateAccountParams{
		Owner:    user.Username,
		Balance:  util.RandomMoney(),
		Currency: util.RandomCurrency(),
	}

	account, err := testQueries.CreateAccount(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, account)

	require.Equal(t, arg.Owner, account.Owner)
	require.Equal(t, arg.Balance, account.Balance)
	require.Equal(t, arg.Currency, account.Currency)

	require.NotZero(t, account.ID)
	require.NotZero(t, account.CreatedAt)

	return account
}

// 这是一个测试用例，它调用 createRandomAccount 辅助函数来测试账户创建的功能。
func TestCreateAccount(t *testing.T) {
	createRandomAccount(t)
}

// 这个测试用例首先使用 createRandomAccount 函数创建一个账户，
// 然后尝试通过 GetAccount 方法检索这个账户，以验证检索功能是否按预期工作。
func TestGetAccount(t *testing.T) {
	account1 := createRandomAccount(t)
	account2, err := testQueries.GetAccount(context.Background(), account1.ID)
	require.NoError(t, err)
	require.NotEmpty(t, account2)

	require.Equal(t, account1.ID, account2.ID)
	require.Equal(t, account1.Owner, account2.Owner)
	require.Equal(t, account1.Balance, account2.Balance)
	require.Equal(t, account1.Currency, account2.Currency)
	require.WithinDuration(t, account1.CreatedAt, account2.CreatedAt, time.Second)
}

// 这个测试用例首先创建一个账户，然后更新它的余额。它测试了 UpdateAccount 方法，并验证返回的账户信息是否反映了更新。
func TestUpdateAccount(t *testing.T) {
	account1 := createRandomAccount(t)

	arg := UpdateAccountParams{
		ID:      account1.ID,
		Balance: util.RandomMoney(),
	}

	account2, err := testQueries.UpdateAccount(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, account2)

	require.Equal(t, account1.ID, account2.ID)
	require.Equal(t, account1.Owner, account2.Owner)
	require.Equal(t, arg.Balance, account2.Balance)
	require.Equal(t, account1.Currency, account2.Currency)
	require.WithinDuration(t, account1.CreatedAt, account2.CreatedAt, time.Second)
}

// 这个测试用例创建一个账户，然后删除它，最后尝试再次检索该账户，以确认它已被删除。它测试了 DeleteAccount 方法的功能。
func TestDeleteAccount(t *testing.T) {
	account1 := createRandomAccount(t)
	err := testQueries.DeleteAccount(context.Background(), account1.ID)
	require.NoError(t, err)

	account2, err := testQueries.GetAccount(context.Background(), account1.ID)
	require.Error(t, err)
	require.EqualError(t, err, sql.ErrNoRows.Error())
	require.Empty(t, account2)
}

// 这个测试用例创建多个账户，并使用 ListAccounts 方法检索一部分账户，测试分页功能是否正常工作。
func TestListAccounts(t *testing.T) {

	var lastAccount Account
	for i := 0; i < 10; i++ {
		lastAccount = createRandomAccount(t)
	}

	arg := ListAccountsParams{
		Owner:  lastAccount.Owner,
		Limit:  5,
		Offset: 0,
	}

	accounts, err := testQueries.ListAccounts(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, accounts)

	for _, account := range accounts {
		require.NotEmpty(t, account)
		require.Equal(t, lastAccount.Owner, account.Owner)
	}
}
