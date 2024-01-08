package db

import (
	"context"
	"errors"
	"log"

	"github.com/jackc/pgx/v5/pgtype"
)

type VerifyEmailTxParams struct {
	EmailId    int64
	SecretCode string
}

type VerifyEmailTxResult struct {
	User        User
	VerifyEmail VerifyEmail
}

func (store *SQLStore) VerifyEmailTx(ctx context.Context, arg VerifyEmailTxParams) (VerifyEmailTxResult, error) {
	var result VerifyEmailTxResult

	// 开始事务
	err := store.execTx(ctx, func(q *Queries) error {

		var err error

		// 增加日志，记录事务开始
		log.Println("Starting transaction for VerifyEmailTx with EmailId:", arg.EmailId)

		// 执行更新验证邮件的操作
		result.VerifyEmail, err = q.UpdateVerifyEmail(ctx, UpdateVerifyEmailParams{
			ID:         arg.EmailId,
			SecretCode: arg.SecretCode,
		})
		if err != nil {
			// 增加日志，记录错误信息
			log.Printf("Error updating verify email: %v\n", err)
			return err
		}

		// 检查返回的验证邮件是否有效（是否有数据被更新）
		if result.VerifyEmail.ID == 0 {
			// 如果没有更新任何记录，返回错误
			log.Printf("No verify email record updated for EmailId: %d and SecretCode: %s\n", arg.EmailId, arg.SecretCode)
			return errors.New("invalid email_id or secret_code")
		}

		// 更新用户的验证状态
		result.User, err = q.UpdateUser(ctx, UpdateUserParams{
			Username:        result.VerifyEmail.Username,
			IsEmailVerified: pgtype.Bool{Bool: true, Valid: true},
		})
		if err != nil {
			// 增加日志，记录错误信息
			log.Printf("Error updating user email verification status: %v\n", err)
			return err
		}

		// 增加日志，记录事务成功完成
		log.Println("Transaction completed successfully for VerifyEmailTx with EmailId:", arg.EmailId)
		return nil
	})

	// 检查整个事务是否成功
	if err != nil {
		// 记录事务错误
		log.Printf("Transaction failed for VerifyEmailTx with EmailId: %d, error: %v\n", arg.EmailId, err)
		return result, err
	}

	return result, nil
}

/*
设计如下的这些代码片段构成了一个电子邮件验证系统的完整后端逻辑，
使用了gRPC、Redis任务队列、邮件发送以及数据库事务。
这些组件共同协作，实现了用户注册后的电子邮件验证流程。

1. rpc_verify_email.proto（gRPC 协议定义文件）
这个文件定义了用于电子邮件验证的gRPC服务接口。
它包含了客户端发送验证请求所需的VerifyEmailRequest消息格式和
服务器响应的VerifyEmailResponse消息格式。

2. task_send_verify_email.go（发送验证邮件任务）
这个文件定义了PayloadSendVerifyEmail结构体，用于封装发送验证邮件的任务数据。
它包含两个关键方法：

DistributeTaskSendVerifyEmail：将任务分发到Redis队列。
ProcessTaskSendVerifyEmail：从队列中提取任务并处理，比如发送验证邮件。

3. rpc_verify_email.go（gRPC API实现）
这个文件实现了VerifyEmail gRPC服务端点。它处理客户端的验证请求，验证请求参数，
然后调用数据库事务来确认电子邮件验证。

VerifyEmail：接收gRPC请求，调用验证逻辑。
validateVerifyEmailRequest：验证请求参数的有效性。

4. tx_verify_email.go（数据库验证事务）
这个文件包含处理电子邮件验证逻辑的数据库事务。

VerifyEmailTxParams：事务的输入参数。
VerifyEmailTxResult：事务的输出结果。
VerifyEmailTx：实际执行更新数据库中用户电子邮件验证状态的函数。

设计思路
gRPC协议定义(rpc_verify_email.proto)：定义了验证电子邮件所需的接口，
允许客户端通过gRPC调用服务。

发送验证邮件(task_send_verify_email.go)：当用户注册时，
服务器会调用DistributeTaskSendVerifyEmail方法将发送邮件的任务放入Redis队列。

处理验证请求(rpc_verify_email.go)：当用户点击邮件中的验证链接时，
客户端会通过gRPC调用VerifyEmail方法。

数据库事务(tx_verify_email.go)：VerifyEmail方法会调用VerifyEmailTx，
在一个数据库事务中更新用户的验证状态。

代码流程概述
用户注册后，服务器创建发送验证邮件的任务，并将其分发到Redis队列。
用户点击邮件中的链接发起验证请求。
服务器接收gRPC验证请求，并调用VerifyEmailTx来执行数据库事务，验证电子邮件。
数据库事务确认用户电子邮件的验证状态，并返回结果给服务器。
服务器将验证结果返回给客户端。

总结
这些代码构建了一个安全、可靠且可扩展的电子邮件验证系统。
系统能够在用户注册时异步发送验证邮件，并处理用户通过验证链接发起的验证请求。
这种设计不仅提高了用户体验（通过异步发送邮件），
而且还确保了系统的安全性（通过数据库事务来管理验证状态）。
通过gRPC接口的使用，这个系统可以轻松地与其他服务或客户端进行通信，
实现了良好的服务端与客户端分离，便于后续维护和开发。
*/
