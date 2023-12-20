package api

/*
这段代码定义了一个简单的 HTTP 服务器，用于处理银行服务相关的网络请求。它使用了 gin 框架来简化路由和请求处理
*/

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	db "github.com/techschool/bank/db/sqlc"
	"github.com/techschool/bank/token"
	"github.com/techschool/bank/util"
)

// Server serves HTTP requests for our banking service.
/*
Server 是一个结构体，它包含了两个字段：
store: 这是 db.Store 类型的一个字段，它是一个接口，定义了一系列与数据库交互的方法。这些方法可能包括添加账户余额、创建账户和转账等操作。
router: 这是 *gin.Engine 类型的字段，它是 gin 框架的核心，用于处理 HTTP 请求和路由。
*/
type Server struct {
	config     util.Config
	store      db.Store
	tokenMaker token.Maker
	router     *gin.Engine
}

// NewServer creates a new HTTP server and set up routing.
// NewServer 函数接受一个 db.Store 接口类型的参数，并返回一个 *Server 指针。这个函数初始化了一个 Server 结构体实例，并设置了相关的路由处理。
func NewServer(config util.Config, store db.Store) (*Server, error) {
	tokenMaker, err := token.NewPasetoMaker(config.TokenSymmetricKey)
	if err != nil {
		return nil, fmt.Errorf("cannot create token maker: %w", err)
	}

	server := &Server{
		config:     config,
		store:      store,
		tokenMaker: tokenMaker,
	}

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("currency", validCurrency)
	}

	server.setupRouter()
	return server, nil
}

func (server *Server) setupRouter() {
	router := gin.Default()

	router.POST("/users", server.createUser)
	router.POST("/users/login", server.loginUser)

	authRoutes := router.Group("/").Use(authMiddleware(server.tokenMaker))
	authRoutes.POST("/accounts", server.createAccount)
	authRoutes.GET("/accounts/:id", server.getAccount)
	authRoutes.GET("/accounts", server.listAccounts)

	authRoutes.POST("/transfers", server.createTransfer)

	server.router = router
}

// Start runs the HTTP server on a specific address.
/*
Start 方法接受一个字符串类型的 address 参数，表示服务器监听的地址和端口（
例如 "0.0.0.0:8080"）。它通过调用 router.Run 方法来启动 HTTP 服务器。
如果服务器启动成功，这个方法会一直运行直到服务器停止；如果启动失败，它会返回一个错误。
*/
func (server *Server) Start(address string) error {
	return server.router.Run(address)
}

/*
errorResponse 函数接受一个 error 类型的参数，并返回一个 gin.H 类型，
这是一个 map[string]interface{} 的别名。它用于创建一个 JSON 响应，
其中包含了错误信息。这个函数通常在处理 HTTP 请求时遇到错误时被调用，以返回错误详情给客户端。
*/
func errorResponse(err error) gin.H {
	return gin.H{"error": err.Error()}
}
