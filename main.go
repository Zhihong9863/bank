package main

import (
	"context"
	"database/sql"
	"os"

	// "log"
	"net"
	"net/http"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	_ "github.com/lib/pq"
	"github.com/rakyll/statik/fs"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/techschool/bank/api"
	db "github.com/techschool/bank/db/sqlc"
	_ "github.com/techschool/bank/doc/statik"
	"github.com/techschool/bank/gapi"
	"github.com/techschool/bank/pb"
	"github.com/techschool/bank/util"

	// "github.com/techschool/simplebank/worker"

	// "github.com/techschool/bank/worker"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/encoding/protojson"
)

/*
加载配置，连接到数据库，然后调用 runGrpcServer 函数来启动 gRPC 服务器。
runGrpcServer 函数创建了一个新的 gRPC 服务器实例，
注册了银行服务的实现，并启动了 gRPC 服务器监听特定的端口。
reflection.Register(grpcServer) 这一行启用了 gRPC 反射，
这允许工具像 evans 或 grpcurl 在运行时查询服务器支持的服务和方法。
*/

func main() {

	config, err := util.LoadConfig(".")
	if err != nil {
		log.Fatal().Err(err).Msg("cannot load config")
	}

	if config.Environment == "development" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	conn, err := sql.Open(config.DBDriver, config.DBSource)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot connect to db")
	}

	runDBMigration(config.MigrationURL, config.DBSource)

	store := db.NewStore(conn)
	go runGatewayServer(config, store)
	runGrpcServer(config, store)
}

func runDBMigration(migrationURL string, dbSource string) {
	migration, err := migrate.New(migrationURL, dbSource)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot create new migrate instance")
	}

	if err = migration.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatal().Err(err).Msg("failed to run migrate up")
	}

	log.Info().Msg("db migrated successfully")
}

func runGrpcServer(config util.Config, store db.Store) {
	server, err := gapi.NewServer(config, store)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot create server")
	}

	gprcLogger := grpc.UnaryInterceptor(gapi.GrpcLogger)
	grpcServer := grpc.NewServer(gprcLogger)
	pb.RegisterSimpleBankServer(grpcServer, server)
	reflection.Register(grpcServer)

	listener, err := net.Listen("tcp", config.GRPCServerAddress)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot create listener")
	}
	log.Printf("start gRPC server at %s", listener.Addr().String())
	err = grpcServer.Serve(listener)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot start gRPC server")
	}

}

/*
增加了通过 HTTP 访问 gRPC 服务的能力。这通过 gRPC-Gateway 实现，
它是一个反向代理，可以将 HTTP/JSON 请求转换为 gRPC 调用，
然后将服务器的 gRPC 响应转换回 HTTP/JSON。
这允许客户端既可以使用原生的 gRPC 也可以使用更通用的 HTTP 来与您的服务通信。

1.启动 gRPC-Gateway：runGatewayServer 函数启动了一个 HTTP 服务器，
该服务器使用 grpcMux 将接收到的 HTTP 请求转换为 gRPC 请求。

5.并行运行：在 main 函数中，通过 go runGatewayServer(config, store)
6.异步地运行了 HTTP 服务器，同时主线程在运行 gRPC 服务器。
*/
func runGatewayServer(config util.Config, store db.Store) {
	server, err := gapi.NewServer(config, store)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot create server")
	}

	//2.配置 JSON 解析器：使用 runtime.JSONPb 来自定义 JSON 的编组和解组行为，使其可以正确处理 proto 消息。
	jsonOption := runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
		MarshalOptions: protojson.MarshalOptions{
			UseProtoNames: true,
		},
		UnmarshalOptions: protojson.UnmarshalOptions{
			DiscardUnknown: true,
		},
	})

	grpcMux := runtime.NewServeMux(jsonOption)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	//3.注册服务处理程序：通过调用 pb.RegisterSimpleBankHandlerServer，
	//将 gRPC 服务注册到了 gRPC-Gateway，这样 HTTP 请求就可以转发到相应的 gRPC 方法。
	err = pb.RegisterSimpleBankHandlerServer(ctx, grpcMux, server)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot register handler server")
	}

	//4.HTTP 监听器：创建了一个监听特定地址的 HTTP 监听器，允许客户端通过 HTTP 协议连接到您的服务。
	mux := http.NewServeMux()
	mux.Handle("/", grpcMux)

	statikFS, err := fs.New()
	if err != nil {
		log.Fatal().Err(err).Msg("cannot create statik fs")
	}

	swaggerHandler := http.StripPrefix("/swagger/", http.FileServer(statikFS))
	mux.Handle("/swagger/", swaggerHandler)

	listener, err := net.Listen("tcp", config.HTTPServerAddress)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot create listener")
	}

	log.Printf("start HTTP gateway server at %s", listener.Addr().String())
	handler := gapi.HttpLogger(mux)
	err = http.Serve(listener, handler)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot start HTTP gateway server")
	}
}

func runGinServer(config util.Config, store db.Store) {
	server, err := api.NewServer(config, store)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot create server")
	}

	err = server.Start(config.HTTPServerAddress)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot start server")
	}
}
