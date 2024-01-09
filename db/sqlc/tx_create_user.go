package db

import (
	"context"
	"log"
)

/*
这个文件定义了一个CreateUserTx函数，它封装了创建新用户记录的操作。
这个函数接受一个结构体CreateUserTxParams，
其中包含创建用户所需的参数和一个回调函数AfterCreate。

函数首先在数据库中创建用户记录。

成功创建用户后，它会调用AfterCreate函数，可以在这个函数中实现发送验证邮件的逻辑。
*/
type CreateUserTxParams struct {
	CreateUserParams
	AfterCreate func(user User) error
}

type CreateUserTxResult struct {
	User User
}

func (store *SQLStore) CreateUserTx(ctx context.Context, arg CreateUserTxParams) (CreateUserTxResult, error) {
	log.Println("Start CreateUserTx")
	var result CreateUserTxResult

	err := store.execTx(ctx, func(q *Queries) error {
		var err error

		result.User, err = q.CreateUser(ctx, arg.CreateUserParams)
		if err != nil {
			log.Printf("CreateUser error: %v", err)
			return err
		}

		//在事务完成后，还能够安全地发送异步任务到Redis队列中，如发送验证邮件的任务
		//当CreateUserTx函数在数据库事务中成功创建了用户之后，它会调用AfterCreate
		//具体这个异步体现在gapi文件夹下的rpc create user.go里面的CreateUser(ctx context.Context, req *pb.CreateUserRequest)

		return arg.AfterCreate(result.User)
	})

	return result, err
}

/*
新增的tx代码是为了演示如何在数据库事务内发送异步任务到Redis。
具体来说，每个文件都定义了一个涉及数据库事务的函数，
create transfer这两个函数分别用于创建用户和执行资金转账。
*/
