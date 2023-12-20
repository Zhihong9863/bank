package api

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"
	db "github.com/techschool/bank/db/sqlc"
	"github.com/techschool/bank/util"
)

/*
这段代码定义了一个HTTP处理函数createUser，用于创建新用户。
这是一个典型的注册功能，接收用户名、密码、全名和电子邮件等信息。
createUserRequest结构体用于映射和验证客户端请求的JSON体。
它使用了binding标签来要求所有字段都是必须的，并且对于用户名要求是字母数字，密码至少6个字符，电子邮件要满足电子邮件格式。
*/
type createUserRequest struct {
	Username string `json:"username" binding:"required,alphanum"`
	Password string `json:"password" binding:"required,min=6"`
	FullName string `json:"full_name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
}

// userResponse结构体定义了返回给客户端的用户信息格式，不包括敏感信息如密码。
type userResponse struct {
	Username          string    `json:"username"`
	FullName          string    `json:"full_name"`
	Email             string    `json:"email"`
	PasswordChangedAt time.Time `json:"password_changed_at"`
	CreatedAt         time.Time `json:"created_at"`
}

// newUserResponse函数接收一个db.User类型的参数，然后生成一个userResponse对象，这样在创建用户后可以发送回客户端。
func newUserResponse(user db.User) userResponse {
	return userResponse{
		Username:          user.Username,
		FullName:          user.FullName,
		Email:             user.Email,
		PasswordChangedAt: user.PasswordChangedAt,
		CreatedAt:         user.CreatedAt,
	}
}

func (server *Server) createUser(ctx *gin.Context) {
	/*
		首先尝试将请求的JSON体绑定到createUserRequest结构体实例。
		如果这个绑定失败了（比如因为请求体中缺少必需字段或格式不正确），
		函数就会返回HTTP 400（错误请求）响应。
	*/
	var req createUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	//然后，代码会尝试使用util.HashPassword函数来散列密码，如果失败则返回HTTP 500（服务器内部错误）响应。
	hashedPassword, err := util.HashPassword(req.Password)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	/*
		接下来，使用请求数据构建CreateUserParams，
		并调用store.CreateUser方法来尝试在数据库中创建用户。
		如果创建用户时出现错误，如用户名已存在（违反唯一性约束），则返回HTTP 403（禁止）响应。
		如果是其他数据库错误，则返回HTTP 500。
	*/
	arg := db.CreateUserParams{
		Username:       req.Username,
		HashedPassword: hashedPassword,
		FullName:       req.FullName,
		Email:          req.Email,
	}

	user, err := server.store.CreateUser(ctx, arg)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code.Name() {
			case "unique_violation":
				ctx.JSON(http.StatusForbidden, errorResponse(err))
				return
			}
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	//如果用户成功创建，newUserResponse函数用于创建一个响应体，然后发送HTTP 200（成功）响应和用户数据。
	rsp := newUserResponse(user)
	ctx.JSON(http.StatusOK, rsp)
}

/*
定义了用户登录请求的数据结构。
Username 字段要求是字母数字且为必填项。
Password 字段要求至少有6个字符且为必填项。
*/
type loginUserRequest struct {
	Username string `json:"username" binding:"required,alphanum"`
	Password string `json:"password" binding:"required,min=6"`
}

/*
定义了登录成功后返回给客户端的数据结构。
包括会话ID、访问令牌、访问令牌过期时间、刷新令牌、刷新令牌过期时间以及用户信息。
*/
type loginUserResponse struct {
	SessionID             uuid.UUID    `json:"session_id"`
	AccessToken           string       `json:"access_token"`
	AccessTokenExpiresAt  time.Time    `json:"access_token_expires_at"`
	RefreshToken          string       `json:"refresh_token"`
	RefreshTokenExpiresAt time.Time    `json:"refresh_token_expires_at"`
	User                  userResponse `json:"user"`
}

/*
实现了用户登录流程，验证用户凭证，并提供了访问令牌和刷新令牌。
这使得用户可以在接下来的HTTP请求中使用这些令牌来验证其身份，
同时允许他们在访问令牌过期后使用刷新令牌来继续访问系统，
从而提供了一种安全且有效的身份验证和会话管理方法。

处理用户登录请求的HTTP处理函数。
尝试从请求的JSON体中绑定数据到loginUserRequest结构体实例。如果绑定失败，返回HTTP 400（错误请求）响应。
使用请求中的Username查询数据库以获取用户信息。如果用户不存在或发生其他数据库错误，返回相应的HTTP响应。
调用CheckPassword函数验证请求中的密码与数据库中散列存储的密码是否匹配。如果不匹配，返回HTTP 401（未授权）响应。
如果密码验证成功，使用tokenMaker创建新的访问令牌，并设置适当的有效期。如果创建令牌失败，返回HTTP 500（服务器内部错误）响应。
*/
func (server *Server) loginUser(ctx *gin.Context) {
	var req loginUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	user, err := server.store.GetUser(ctx, req.Username)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	err = util.CheckPassword(req.Password, user.HashedPassword)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	accessToken, accessPayload, err := server.tokenMaker.CreateToken(
		user.Username,
		user.Role,
		server.config.AccessTokenDuration,
	)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// refreshToken, refreshPayload, err := server.tokenMaker.CreateToken(
	// 	user.Username,
	// 	user.Role,
	// 	server.config.RefreshTokenDuration,
	// )
	// if err != nil {
	// 	ctx.JSON(http.StatusInternalServerError, errorResponse(err))
	// 	return
	// }

	// session, err := server.store.CreateSession(ctx, db.CreateSessionParams{
	// 	ID:           refreshPayload.ID,
	// 	Username:     user.Username,
	// 	RefreshToken: refreshToken,
	// 	UserAgent:    ctx.Request.UserAgent(),
	// 	ClientIp:     ctx.ClientIP(),
	// 	IsBlocked:    false,
	// 	ExpiresAt:    refreshPayload.ExpiredAt,
	// })
	// if err != nil {
	// 	ctx.JSON(http.StatusInternalServerError, errorResponse(err))
	// 	return
	// }

	//使用成功验证的用户信息创建loginUserResponse实例。
	//封装了访问令牌、访问令牌过期时间和用户信息。
	//返回HTTP 200（成功）响应，携带loginUserResponse实例作为JSON体
	rsp := loginUserResponse{
		// SessionID:             session.ID,
		AccessToken:          accessToken,
		AccessTokenExpiresAt: accessPayload.ExpiredAt,
		// RefreshToken:          refreshToken,
		// RefreshTokenExpiresAt: refreshPayload.ExpiredAt,
		User: newUserResponse(user),
	}
	ctx.JSON(http.StatusOK, rsp)
}
