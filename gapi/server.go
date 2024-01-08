package gapi

import (
	"fmt"

	db "github.com/techschool/bank/db/sqlc"
	"github.com/techschool/bank/pb"
	"github.com/techschool/bank/token"
	"github.com/techschool/bank/util"
	"github.com/techschool/bank/worker"
)

/*
定义了一个 Server 结构，它实现了 pb.UnimplementedSimpleBankServer。
这个 Server 结构包含了配置、数据库存储、和令牌生成器等字段。
NewServer 函数是创建新的 gRPC Server 的构造函数。
它接受配置和数据库存储作为参数，并返回一个设置好的 Server 实例。
*/

// Server serves gRPC requests for our banking service.
type Server struct {
	pb.UnimplementedSimpleBankServer
	config          util.Config
	store           db.Store
	tokenMaker      token.Maker
	taskDistributor worker.TaskDistributor
}

// NewServer creates a new gRPC server.
func NewServer(config util.Config, store db.Store, taskDistributor worker.TaskDistributor) (*Server, error) {
	tokenMaker, err := token.NewPasetoMaker(config.TokenSymmetricKey)
	if err != nil {
		return nil, fmt.Errorf("cannot create token maker: %w", err)
	}

	server := &Server{
		config:          config,
		store:           store,
		tokenMaker:      tokenMaker,
		taskDistributor: taskDistributor,
	}

	return server, nil
}
