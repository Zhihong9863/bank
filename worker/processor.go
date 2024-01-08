package worker

import (
	"context"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog/log"
	db "github.com/techschool/bank/db/sqlc"
	"github.com/techschool/bank/mail"
	// "github.com/techschool/bank/mail"
)

/*
这部分代码定义了任务处理器的结构体和方法。RedisTaskProcessor 结构体包含 asynq.Server 以及用于业务逻辑处理的 store 和 mailer。

TaskProcessor 接口定义了 Start 和 ProcessTaskSendVerifyEmail 方法。
NewRedisTaskProcessor 函数创建一个新的 RedisTaskProcessor 实例，并配置服务器以处理特定队列和错误处理。
Start 方法设置任务处理函数和启动服务器以处理任务。
ProcessTaskSendVerifyEmail 方法处理发送验证邮件的任务。
它解析任务负载，执行业务逻辑（如获取用户信息，创建验证邮件记录，发送邮件等），
*/

const (
	QueueCritical = "critical"
	QueueDefault  = "default"
)

type TaskProcessor interface {
	Start() error
	ProcessTaskSendVerifyEmail(ctx context.Context, task *asynq.Task) error
}

type RedisTaskProcessor struct {
	server *asynq.Server
	store  db.Store
	mailer mail.EmailSender
}

/*
代码定义了任务处理器的行为，它监听Redis队列，
一旦队列中出现了任务，它就会处理这些任务。
例如，ProcessTaskSendVerifyEmail函数将处理发送验证邮件的任务。
*/
func NewRedisTaskProcessor(redisOpt asynq.RedisClientOpt, store db.Store, mailer mail.EmailSender) TaskProcessor {
	// logger := NewLogger()
	// redis.SetLogger(logger)

	server := asynq.NewServer(
		redisOpt,
		asynq.Config{
			Queues: map[string]int{
				QueueCritical: 10,
				QueueDefault:  5,
			},
			ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
				log.Error().Err(err).Str("type", task.Type()).
					Bytes("payload", task.Payload()).Msg("process task failed")
			}),
			Logger: NewLogger(),
		},
	)

	return &RedisTaskProcessor{
		server: server,
		store:  store,
		mailer: mailer,
	}
}

func (processor *RedisTaskProcessor) Start() error {
	mux := asynq.NewServeMux()

	mux.HandleFunc(TaskSendVerifyEmail, processor.ProcessTaskSendVerifyEmail)

	return processor.server.Start(mux)
}

/*
和task_send_verify_email.go差不多，另一个解释版本

即时反馈：

注册界面立即向服务器发送了您的注册信息。
服务器接收到这些信息，并开始在数据库中创建您的用户账户。
一旦服务器确认用户账户创建成功，它会立即向注册界面返回一个“注册成功”的消息。

后台任务启动：

在返回“注册成功”消息的同时，服务器也会在后台创建一个任务，这个任务是发送验证邮件给您。
这个任务被发送到Redis任务队列中，并不会影响到服务器给您的即时反馈。

邮件发送：

在后台，一个任务处理器会检查Redis任务队列，当发现您的验证邮件任务时，就会开始处理这个任务。
处理器执行发送邮件的操作，您的邮箱服务商会收到一个请求，要求发送一封验证邮件到您提供的邮箱地址。

用户体验：

对于您来说，注册过程是顺畅的，没有任何等待。您看到的是“注册成功”的消息，这时您可以继续使用网站或应用的其他部分。
通常会有一个提示，告诉您一封验证邮件已经发送到您的邮箱，让您检查邮箱并完成验证步骤。
*/
