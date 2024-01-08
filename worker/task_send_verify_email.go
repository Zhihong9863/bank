package worker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog/log"
	db "github.com/techschool/bank/db/sqlc"
	"github.com/techschool/bank/util"
)

/*
这个文件包含了发送验证邮件任务的负载结构体定义和具体的任务处理函数。

PayloadSendVerifyEmail 结构体定义了发送验证邮件任务的负载数据。
DistributeTaskSendVerifyEmail 方法用于将发送验证邮件的任务分发到队列。
ProcessTaskSendVerifyEmail 方法是具体的任务处理逻辑，它从队列中取出任务，解析负载，执行发送验证邮件的业务逻辑。

*/

const TaskSendVerifyEmail = "task:send_verify_email"

type PayloadSendVerifyEmail struct {
	Username string `json:"username"`
}

func (distributor *RedisTaskDistributor) DistributeTaskSendVerifyEmail(
	ctx context.Context,
	payload *PayloadSendVerifyEmail,
	opts ...asynq.Option,
) error {
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal task payload: %w", err)
	}

	task := asynq.NewTask(TaskSendVerifyEmail, jsonPayload, opts...)
	info, err := distributor.client.EnqueueContext(ctx, task)
	if err != nil {
		return fmt.Errorf("failed to enqueue task: %w", err)
	}

	log.Info().Str("type", task.Type()).Bytes("payload", task.Payload()).
		Str("queue", info.Queue).Int("max_retry", info.MaxRetry).Msg("enqueued task")
	return nil
}

func (processor *RedisTaskProcessor) ProcessTaskSendVerifyEmail(ctx context.Context, task *asynq.Task) error {
	var payload PayloadSendVerifyEmail
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", asynq.SkipRetry)
	}

	user, err := processor.store.GetUser(ctx, payload.Username)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	verifyEmail, err := processor.store.CreateVerifyEmail(ctx, db.CreateVerifyEmailParams{
		Username:   user.Username,
		Email:      user.Email,
		SecretCode: util.RandomString(32),
	})
	if err != nil {
		return fmt.Errorf("failed to create verify email: %w", err)
	}

	subject := "Welcome to Simple Bank"
	// TODO: replace this URL with an environment variable that points to a front-end page
	verifyUrl := fmt.Sprintf("http://localhost:8080/v1/verify_email?email_id=%d&secret_code=%s",
		verifyEmail.ID, verifyEmail.SecretCode)
	content := fmt.Sprintf(`Hello %s,<br/>
	Thank you for registering with us!<br/>
	Please <a href="%s">click here</a> to verify your email address.<br/>
	`, user.FullName, verifyUrl)
	to := []string{user.Email}

	err = processor.mailer.SendEmail(subject, content, to, nil, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to send verify email: %w", err)
	}

	log.Info().Str("type", task.Type()).Bytes("payload", task.Payload()).
		Str("email", user.Email).Msg("processed task")
	return nil
}

/*
用户注册：

用户打开Postman，并设置一个POST请求，目标是您的web服务器的/v1/create_user端点。
用户在请求的body中填入他们的用户名、全名、电子邮件和密码。
用户发送请求。


Web服务器处理：

Web服务器接收到Postman的请求。
服务器执行注册逻辑，将用户信息存储到数据库中，创建新的用户记录。
如果用户成功创建，服务器将创建一个新任务，用于发送验证电子邮件给用户。
任务包括用户的用户名，以及其他任务设置，如最大重试次数（10次）和队列名称（标记为critical）。

Redis任务队列：

创建的任务被发送到Redis服务器，它作为任务队列的后端。
任务在队列中排队等待被处理，队列按照一定的顺序（如优先级）管理所有的任务。


任务处理器：

在服务器上，一个独立的进程或协程（在Go中称为goroutine）运行着Redis任务处理器。
处理器监视任务队列，当发现有新任务时，它会取出任务并开始处理。
对于发送验证电子邮件的任务，处理器会执行发送邮件的操作，可能会调用电子邮件服务的API或使用SMTP客户端。


任务完成与日志记录：

一旦电子邮件发送任务完成，无论是成功还是失败，任务处理器都会记录一条日志。
这条日志显示任务的类型、处理的用户名以及其他相关信息，如是否成功、重试次数等。
这条日志最终出现在您的服务器控制台上，作为任务处理结果的反馈。

整个流程的设计让耗时的电子邮件发送操作在用户注册流程完成后异步执行，
不会阻塞或延迟Web服务器响应用户注册请求的速度。
这种设计同时提高了用户体验和系统的可伸缩性。如果邮件服务出现问题或者发送邮件需要较长时间，任务队列和处理器也能够保证邮件最终会被发送出去（通过重试机制）。
*/
