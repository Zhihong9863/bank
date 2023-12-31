package gapi

import (
	db "github.com/techschool/bank/db/sqlc"
	"github.com/techschool/bank/pb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

/*
这个辅助函数将数据库层返回的 User 结构转换为 gRPC 服务需要的 pb.User 消息格式。
它主要用于在 CreateUser 和 LoginUser 方法中构造响应。
*/
func convertUser(user db.User) *pb.User {
	return &pb.User{
		Username:          user.Username,
		FullName:          user.FullName,
		Email:             user.Email,
		PasswordChangedAt: timestamppb.New(user.PasswordChangedAt),
		CreatedAt:         timestamppb.New(user.CreatedAt),
	}
}
