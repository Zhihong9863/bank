package worker

import (
	"context"

	"github.com/hibiken/asynq"
)

/*
这部分代码定义了任务分发的结构体和方法。
RedisTaskDistributor 结构体有一个 asynq.Client 类型的客户端，
用于向 Redis 任务队列发送任务。

TaskDistributor 接口定义了一个 DistributeTaskSendVerifyEmail 方法，
它将发送验证邮件的任务分发到任务队列中。
NewRedisTaskDistributor 函数创建一个新的 RedisTaskDistributor 实例。
DistributeTaskSendVerifyEmail 方法将验证邮件发送的任务封装成 JSON 格式的负载，
然后通过 asynq.Client 的 EnqueueContext 方法将任务加入队列。
*/
type TaskDistributor interface {
	DistributeTaskSendVerifyEmail(
		ctx context.Context,
		payload *PayloadSendVerifyEmail,
		opts ...asynq.Option,
	) error
}

type RedisTaskDistributor struct {
	client *asynq.Client
}

func NewRedisTaskDistributor(redisOpt asynq.RedisClientOpt) TaskDistributor {
	client := asynq.NewClient(redisOpt)
	return &RedisTaskDistributor{
		client: client,
	}
}

/*
这部分的主题：
在一个web服务中实现后台工作处理机制。
使用Redis任务队列可以将长时间运行或资源密集型的任务从主应用程序流程中分离出来，
以提高性能和响应速度。

例如，考虑一个在线银行应用程序，当用户注册新账户时，他们需要验证其电子邮件地址。
发送验证邮件是一个可以异步完成的任务，因为用户不需要立即收到邮件，
而服务器在处理这一过程时也不应该阻塞其他操作。

在没有后台工作处理系统的情况下，应用程序可能直接在用户注册流程中同步发送电子邮件，
这会导致用户等待邮件发送完成，如果邮件服务器响应慢或者出现问题，用户体验将非常差。

使用Redis任务队列，应用程序可以将发送邮件的任务推送到队列中，
并立即响应用户请求，从而不会延迟用户注册流程。
然后，后台工作处理器会异步地从队列中取出任务并发送邮件，
即使邮件发送过程慢，也不会影响主应用程序的性能。

这就是这部分代码的概念：如何构建可扩展的应用程序，将耗时任务异步化，
从而提供更快的响应时间和更好的用户体验。
这种模式也使得后续的维护和扩展变得更加容易，因为后台任务处理逻辑与主应用程序逻辑解耦。
*/
