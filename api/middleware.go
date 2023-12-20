package api

//代码实现了一个中间件authMiddleware，用于在使用Gin框架构建的HTTP服务器中处理授权。
//中间件的目的是验证HTTP请求中的访问令牌（通常是JWT或PASETO）。
//Gin中间件，负责检查HTTP请求中的授权头部，验证其中的令牌，并在令牌有效时允许请求继续处理。
//这是一种常见的身份验证方法，用于保护需要授权的HTTP端点

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/techschool/bank/token"
)

// 定义了一些常量，包括
// 认证头部的键authorizationHeaderKey，
// 认证类型authorizationTypeBearer，
// 和上下文中用于存储认证有效负载的键authorizationPayloadKey。
const (
	authorizationHeaderKey  = "authorization"
	authorizationTypeBearer = "bearer"
	authorizationPayloadKey = "authorization_payload"
)

// AuthMiddleware creates a gin middleware for authorization
//tokenMaker是一个生成和验证令牌的接口。
//authMiddleware返回一个Gin处理函数，该函数将作为中间件用于验证请求。

/*
中间件首先尝试从请求头中获取authorization头部。
如果头部不存在，中间件将中止请求并返回HTTP 401（未授权）响应。
如果存在，则检查头部格式是否正确（应该包含两部分，类型和令牌）。
确认认证类型是否为bearer，这是HTTP Bearer Token的标准认证方案。
提取令牌，并使用tokenMaker验证令牌的有效性。
如果验证失败，中止请求并返回HTTP 401。
如果验证成功，将令牌有效负载存储在请求的上下文中，以便后续的处理函数可以使用。
*/
//gin.HandlerFunc是一个类型，它是对应于Gin框架中的请求处理函数的签名。
func authMiddleware(tokenMaker token.Maker) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authorizationHeader := ctx.GetHeader(authorizationHeaderKey)

		if len(authorizationHeader) == 0 {
			err := errors.New("authorization header is not provided")
			//ctx.AbortWithStatusJSON()是Gin的方法，用于中止请求处理并立即返回JSON格式的错误响应。
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
			return
		}

		fields := strings.Fields(authorizationHeader)
		if len(fields) < 2 {
			err := errors.New("invalid authorization header format")
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
			return
		}

		authorizationType := strings.ToLower(fields[0])
		if authorizationType != authorizationTypeBearer {
			err := fmt.Errorf("unsupported authorization type %s", authorizationType)
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
			return
		}

		accessToken := fields[1]
		payload, err := tokenMaker.VerifyToken(accessToken)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
			return
		}

		ctx.Set(authorizationPayloadKey, payload)
		//ctx.Next()是Gin框架中的一个方法，它表示中间件处理完成后继续执行后续的中间件或路由处理函数。
		ctx.Next()
	}
}
