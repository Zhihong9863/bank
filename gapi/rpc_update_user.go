package gapi

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/techschool/bank/db/sqlc"
	"github.com/techschool/bank/pb"
	"github.com/techschool/bank/util"
	"github.com/techschool/bank/val"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

/*
定义了一个 UpdateUser 方法，该方法处理用户更新信息的请求。
这个方法是 Server 结构体的一部分，该结构体实现了由 .proto 文件定义的 gRPC 服务接口

这段代码是典型的 gRPC 服务端处理逻辑，涉及到请求验证、数据库操作、错误处理和响应构建。
*/

/*
输入参数: 接收一个 context.Context 和一个 *pb.UpdateUserRequest 参数。
UpdateUserRequest 是由 protobuf 文件定义的消息类型，它携带更新用户所需的信息。
*/
func (server *Server) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UpdateUserResponse, error) {

	authPayload, err := server.authorizeUser(ctx, []string{util.BankerRole, util.DepositorRole})
	if err != nil {
		return nil, unauthenticatedError(err)
	}
	log.Printf("UpdateUser called with request: %v", req)

	/*
		验证请求: validateUpdateUserRequest 函数检查请求是否有效，
		包括用户名、密码、全名和电子邮件的格式。如果有任何验证错误，
		它会返回一个包含所有字段违规详情的 BadRequest_FieldViolation 列表。
	*/

	violations := validateUpdateUserRequest(req)
	if violations != nil {
		return nil, invalidArgumentError(violations)
	}

	if authPayload.Role != util.BankerRole && authPayload.Username != req.GetUsername() {
		return nil, status.Errorf(codes.PermissionDenied, "cannot update other user's info")
	}

	/*
		更新操作: 方法构建了一个 db.UpdateUserParams 结构体，
		用于将请求数据转换为数据库操作所需的格式。
		如果请求中包含了密码，它会先进行哈希处理。
	*/

	arg := db.UpdateUserParams{
		Username: req.GetUsername(),
		FullName: pgtype.Text{
			String: req.GetFullName(),
			Valid:  req.FullName != nil,
		},
		Email: pgtype.Text{
			String: req.GetEmail(),
			Valid:  req.Email != nil,
		},
	}

	if req.Password != nil {
		hashedPassword, err := util.HashPassword(req.GetPassword())
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to hash password: %s", err)
		}

		arg.HashedPassword = pgtype.Text{
			String: hashedPassword,
			Valid:  true,
		}

		arg.PasswordChangedAt = pgtype.Timestamptz{
			Time:  time.Now(),
			Valid: true,
		}
	}

	/*
		数据库更新: 调用 server.store.UpdateUser 方法来执行数据库更新操作。
		store 是 Server 结构体的一个字段，它实现了数据库操作的接口。
	*/

	user, err := server.store.UpdateUser(ctx, arg)
	if err != nil {
		log.Printf("UpdateUser error: %v", err)
		if errors.Is(err, db.ErrRecordNotFound) {
			return nil, status.Errorf(codes.NotFound, "user not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to update user: %s", err)
	}

	/*
		响应: 如果用户更新成功，它会返回一个 *pb.UpdateUserResponse，其中包含了更新后的用户信息。
	*/
	log.Println("UpdateUser method completed successfully")
	rsp := &pb.UpdateUserResponse{
		User: convertUser(user),
	}
	return rsp, nil
}

// 这是一个辅助函数，用于验证传入的 UpdateUserRequest。它检查每个字段是否符合特定的验证规则，比如用户名是否为空或格式错误，密码是否符合安全要求等。
func validateUpdateUserRequest(req *pb.UpdateUserRequest) (violations []*errdetails.BadRequest_FieldViolation) {
	if err := val.ValidateUsername(req.GetUsername()); err != nil {
		violations = append(violations, fieldViolation("username", err))
	}

	if req.Password != nil {
		if err := val.ValidatePassword(req.GetPassword()); err != nil {
			violations = append(violations, fieldViolation("password", err))
		}
	}

	if req.FullName != nil {
		if err := val.ValidateFullName(req.GetFullName()); err != nil {
			violations = append(violations, fieldViolation("full_name", err))
		}
	}

	if req.Email != nil {
		if err := val.ValidateEmail(req.GetEmail()); err != nil {
			violations = append(violations, fieldViolation("email", err))
		}
	}

	return violations
}
