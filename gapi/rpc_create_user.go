package gapi

import (
	"context"

	"github.com/lib/pq"
	db "github.com/techschool/bank/db/sqlc"
	"github.com/techschool/bank/pb"
	"github.com/techschool/bank/util"
	"github.com/techschool/bank/val"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

/*
处理来自客户端的创建用户请求。它首先验证请求参数，
然后对密码进行加密处理，并创建一个新的用户记录。
如果创建过程中出现错误，比如用户名已经存在，它会返回相应的错误信息。
如果用户创建成功，则返回创建的用户信息。
*/
func (server *Server) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	violations := validateCreateUserRequest(req)
	if violations != nil {
		return nil, invalidArgumentError(violations)
	}

	hashedPassword, err := util.HashPassword(req.GetPassword())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to hash password: %s", err)
	}

	arg := db.CreateUserParams{
		// CreateUserParams: db.CreateUserParams{
		Username:       req.GetUsername(),
		HashedPassword: hashedPassword,
		FullName:       req.GetFullName(),
		Email:          req.GetEmail(),
	}
	// AfterCreate: func(user db.User) error {
	// 	taskPayload := &worker.PayloadSendVerifyEmail{
	// 		Username: user.Username,
	// 	}
	// 	opts := []asynq.Option{
	// 		asynq.MaxRetry(10),
	// 		asynq.ProcessIn(10 * time.Second),
	// 		asynq.Queue(worker.QueueCritical),
	// 	}

	// 	return server.taskDistributor.DistributeTaskSendVerifyEmail(ctx, taskPayload, opts...)
	// },

	// txResult, err := server.store.CreateUserTx(ctx, arg)

	user, err := server.store.CreateUser(ctx, arg)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code.Name() {
			case "unique_violation":
				return nil, status.Errorf(codes.AlreadyExists, "username already exists: %s", err)
			}
		}
		return nil, status.Errorf(codes.Internal, "failed to create user: %s", err)
	}

	rsp := &pb.CreateUserResponse{
		User: convertUser(user),
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
