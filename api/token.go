package api

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	db "github.com/techschool/bank/db/sqlc"
)

/*
定义了一个处理函数renewAccessToken，它的目的是根据一个有效的refresh token
来续签一个新的access token。这个功能是安全的用户会话管理和身份验证流程的一部分，
与server.go提到的登录流程和会话管理相关
*/

type renewAccessTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type renewAccessTokenResponse struct {
	AccessToken          string    `json:"access_token"`
	AccessTokenExpiresAt time.Time `json:"access_token_expires_at"`
}

func (server *Server) renewAccessToken(ctx *gin.Context) {
	var req renewAccessTokenRequest
	//接收refresh token: 函数首先尝试从HTTP请求的JSON体中提取refresh_token字段，这个令牌是之前登录流程中发给用户的。
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	//验证refresh token: 使用server.tokenMaker.VerifyToken方法来验证refresh token的有效性。
	//如果验证失败（比如令牌过期或篡改），会返回HTTP 401（未授权）响应。
	refreshPayload, err := server.tokenMaker.VerifyToken(req.RefreshToken)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	//检索会话: 使用refresh token的ID从数据库中获取相应的会话信息。如果找不到会话或发生其他错误，返回相应的HTTP错误响应。
	session, err := server.store.GetSession(ctx, refreshPayload.ID)
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	//会话状态检查: 检查会话是否被阻止，用户是否匹配，以及refresh token是否与存储的token一致。
	//如果有任何问题，返回HTTP 401（未授权）响应。
	if session.IsBlocked {
		err := fmt.Errorf("blocked session")
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	if session.Username != refreshPayload.Username {
		err := fmt.Errorf("incorrect session user")
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	if session.RefreshToken != req.RefreshToken {
		err := fmt.Errorf("mismatched session token")
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	if time.Now().After(session.ExpiresAt) {
		err := fmt.Errorf("expired session")
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	//创建新的access token: 如果所有验证步骤都通过，使用用户信息和配置的有效期创建新的access token
	accessToken, accessPayload, err := server.tokenMaker.CreateToken(
		refreshPayload.Username,
		refreshPayload.Role,
		server.config.AccessTokenDuration,
	)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	//发送响应: 返回新的access token及其过期时间给用户
	rsp := renewAccessTokenResponse{
		AccessToken:          accessToken,
		AccessTokenExpiresAt: accessPayload.ExpiredAt,
	}
	ctx.JSON(http.StatusOK, rsp)
}

/*
这个续签流程与server.go中的代码紧密相关，因为它实现了使用refresh token来获取新的access token的功能，
这是在用户的access token过期后继续访问系统的一种方式。这也是一种安全措施，
因为即使access token被盗用，它们很快就会过期，而且只有持有refresh token的用户才能请求新的access token。

在实际的用户身份验证和会话管理流程中，access token通常用于短期访问（如一个小时），
而refresh token具有更长的有效期（如一周），并且只能用于获取新的access token。
这样，即使access token被盗，攻击者也无法长期访问系统，
因为他们没有refresh token来获取新的access token。
同时，这也减少了用户因token过期而需要频繁重新登录的不便。
*/
