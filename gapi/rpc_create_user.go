package gapi

import (
	"context"
	"log"
	"time"

	"github.com/hibiken/asynq"
	db "github.com/techschool/bank/db/sqlc"
	"github.com/techschool/bank/pb"
	"github.com/techschool/bank/util"
	"github.com/techschool/bank/val"
	"github.com/techschool/bank/worker"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

/*
处理来自客户端的创建用户请求。它首先验证请求参数，
然后对密码进行加密处理，并创建一个新的用户记录。
如果创建过程中出现错误，比如用户名已经存在，它会返回相应的错误信息。
如果用户创建成功，则返回创建的用户信息。
在成功创建用户记录后，它使用taskDistributor将发送验证邮件的任务添加到Redis队列。
这说明在用户注册流程中集成了异步任务的创建。
*/
func (server *Server) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	log.Println("Start CreateUser method")
	violations := validateCreateUserRequest(req)
	if violations != nil {
		return nil, invalidArgumentError(violations)
	}

	hashedPassword, err := util.HashPassword(req.GetPassword())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to hash password: %s", err)
	}

	arg := db.CreateUserTxParams{
		CreateUserParams: db.CreateUserParams{
			Username:       req.GetUsername(),
			HashedPassword: hashedPassword,
			FullName:       req.GetFullName(),
			Email:          req.GetEmail(),
		},
		AfterCreate: func(user db.User) error {
			taskPayload := &worker.PayloadSendVerifyEmail{
				Username: user.Username,
			}
			opts := []asynq.Option{
				asynq.MaxRetry(10),
				/*
					使用asynq.ProcessIn(10 * time.Second)来延迟异步任务的执行确实有助于
					确保数据库事务有足够的时间完全提交，尤其是在涉及复杂操作或多个步骤的事务中。

					这里的关键点是延迟异步任务（如发送验证邮件）的执行，
					可以确保在任务被拾取和执行之前，所有相关的数据库操作（如用户记录的创建）都已经成功完成并提交。
				*/
				asynq.ProcessIn(10 * time.Second),
				asynq.Queue(worker.QueueCritical),
			}

			return server.taskDistributor.DistributeTaskSendVerifyEmail(ctx, taskPayload, opts...)
		},
	}

	txResult, err := server.store.CreateUserTx(ctx, arg)

	if err != nil {
		log.Printf("CreateUserTx error: %v", err)
		return nil, err
	}
	log.Println("CreateUser method completed successfully")

	rsp := &pb.CreateUserResponse{
		User: convertUser(txResult.User),
	}
	return rsp, nil
}

func validateCreateUserRequest(req *pb.CreateUserRequest) (violations []*errdetails.BadRequest_FieldViolation) {
	if err := val.ValidateUsername(req.GetUsername()); err != nil {
		violations = append(violations, fieldViolation("username", err))
	}

	if err := val.ValidatePassword(req.GetPassword()); err != nil {
		violations = append(violations, fieldViolation("password", err))
	}

	if err := val.ValidateFullName(req.GetFullName()); err != nil {
		violations = append(violations, fieldViolation("full_name", err))
	}

	if err := val.ValidateEmail(req.GetEmail()); err != nil {
		violations = append(violations, fieldViolation("email", err))
	}

	return violations
}

/*
首先，在这个项目中我们的策略是这个

后验证注册流程：

用户填写信息并提交注册。
系统即刻创建用户账户，显示注册成功。
系统异步发送验证邮件。
用户登录邮箱，点击验证链接完成验证。
这种流程的好处是用户体验流畅，用户无需等待邮件到来就可以完成注册。
但缺点是，如果邮件验证失败，可能会导致账户被滥用或需要额外的手段来处理未验证的账户。

而关于延迟的补充（asynq.ProcessIn(10 * time.Second),）：

用户体验：

后验证注册流程允许用户在不等待验证邮件到达的情况下完成注册，这确实提供了更流畅的用户体验。

延迟发送邮件：

设置asynq.ProcessIn(10 * time.Second)是为了延迟任务被处理的时间，而不是设置重试的间隔。这意味着，
当用户注册后，系统会在10秒后开始尝试发送验证邮件。
这个延迟有助于确保所有相关的数据库操作（如创建用户记录）都已经完成，从而在发送邮件时可以确保数据的一致性。

邮件发送失败和重试：

如果邮件发送失败，asynq任务队列提供的重试机制可以自动尝试重新发送。
在设置中，asynq.MaxRetry(10)意味着系统最多会尝试发送邮件10次。
重试间隔通常会随着尝试次数的增加而增长，
这意味着系统不会无限快速地重试，而是会在尝试之间留出更多时间。

处理未验证账户：

即便设置了重试，仍然存在邮件永远不被用户接收或用户不点击验证链接的情况。这通常需要额外的逻辑来处理，例如：
为未验证的用户账户设置一个有效期，在一定时间后如果未验证则自动删除或禁用。
提供一个用户界面让用户可以要求重新发送验证邮件。
定期清理长时间未验证的账户。

慢慢消费消息队列：

如果担心邮件服务或任务处理器在高负载下的表现，可以考虑实现更复杂的消费策略，比如限制处理速率，或者在检测到邮件服务负载过高时动态调整任务处理的优先级或速率。
总之，设置任务的延迟执行时间和重试策略是确保邮件最终能够被发送的重要机制。
但它们并不直接解决所有的邮件验证问题，而是作为一个更大的用户账户管理策略的一部分。
可能还需要考虑如何处理那些即使在多次重试后仍然未能验证的账户。
*/
