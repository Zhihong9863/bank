package api

/*
这段代码提供了之前代码段中路由处理函数的具体实现。
这些处理函数负责处理来自HTTP请求的数据，与数据库交互，并返回适当的响应。
*/

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	db "github.com/techschool/bank/db/sqlc"
	"github.com/techschool/bank/token"
)

type createAccountRequest struct {
	Currency string `json:"currency" binding:"required,currency"`
}

/*
createAccount 函数处理创建新银行账户的POST请求。
它首先尝试从请求的JSON正文中绑定数据到createAccountRequest结构体。
如果请求数据不符合要求（例如，缺少必要的字段），它会返回400状态码（BadRequest）和错误信息。
如果数据绑定成功，它将构造一个CreateAccountParams结构体，并调用server.store.CreateAccount方法来在数据库中创建新账户。
如果账户创建成功，它返回201状态码（Created）和账户信息；如果出现服务器内部错误，它返回500状态码（InternalServerError）和错误信息。
*/
func (server *Server) createAccount(ctx *gin.Context) {
	var req createAccountRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	//这段代码确保用户只能为自己创建账户。它从上下文中获取授权载荷（authPayload），这包含了用户名等信息，并用它来设置新账户的所有者。
	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	arg := db.CreateAccountParams{
		Owner:    authPayload.Username,
		Currency: req.Currency,
		Balance:  0,
	}

	account, err := server.store.CreateAccount(ctx, arg)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code.Name() {
			case "foreign_key_violation", "unique_violation":
				ctx.JSON(http.StatusForbidden, errorResponse(err))
				return
			}
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, account)

}

type getAccountRequest struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

/*
getAccount 函数处理获取特定银行账户信息的GET请求。
它从请求的URI中提取账户ID，并尝试将其绑定到getAccountRequest结构体。
如果URI中没有有效的ID（或者ID小于1），它会返回400状态码和错误信息。
如果URI绑定成功，它使用账户ID调用server.store.GetAccount方法来获取账户详情。
如果找到了账户，它返回200状态码和账户信息；如果账户不存在，它返回404状态码（NotFound）和错误信息；如果出现其他错误，它返回500状态码和错误信息。
*/
func (server *Server) getAccount(ctx *gin.Context) {
	var req getAccountRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	account, err := server.store.GetAccount(ctx, req.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			// 如果没有找到账户，返回404状态码
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}

		// 对于所有其他类型的错误，返回500状态码
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	//在这里，代码检查请求的账户是否属于已认证的用户。如果请求获取的账户不属于发起请求的用户，将返回一个HTTP 401（未授权）错误。
	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	if account.Owner != authPayload.Username {
		err := errors.New("account doesn't belong to the authenticated user")
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, account)
}

type listAccountRequest struct {
	PageID   int32 `form:"page_id" binding:"required,min=1"`
	PageSize int32 `form:"page_size" binding:"required,min=5,max=10"`
}

/*
listAccounts 函数处理列出银行账户的GET请求。
它从请求的查询字符串中提取分页参数，并尝试将其绑定到listAccountRequest结构体。
如果查询字符串中的分页参数不符合要求，它会返回400状态码和错误信息。
如果绑定成功，它将计算要查询的数据的偏移量，并调用server.store.ListAccounts方法来获取账户列表。
如果成功获取到账户列表，它返回200状态码和账户列表；如果出现服务器内部错误，它返回500状态码和错误信息。
*/
func (server *Server) listAccounts(ctx *gin.Context) {
	var req listAccountRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	//此代码用于列出属于已认证用户的所有账户。它使用授权载荷中的用户名来查询数据库，并返回属于该用户的所有账户。
	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	arg := db.ListAccountsParams{
		Owner:  authPayload.Username,
		Limit:  req.PageSize,
		Offset: (req.PageID - 1) * req.PageSize,
	}

	accounts, err := server.store.ListAccounts(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, accounts)
}

/*
gin 是一个用 Go 语言编写的 HTTP web 框架。它是一个高性能的框架，被设计为处理 HTTP 请求更加快速和方便。

在 RESTful API 开发中，gin 可以快速地设置端点（Endpoints），
并对 GET、POST、PUT、DELETE 等 HTTP 方法进行响应处理，
它的路由设置简单明了，并且性能优异。

在代码中，gin 被用来创建 HTTP 服务器，设置路由规则，并处理具体的 HTTP 请求。
通过 gin，您能够定义如何响应特定路径的请求、如何从请求中读取数据、以及如何返回响应数据。

这些处理函数展示了如何在gin框架中处理RESTful API请求，
并与sqlc生成的数据库操作方法结合，以执行CRUD（创建、读取、更新、删除）操作。
每个函数都使用gin.Context来访问请求数据、管理请求生命周期、设置响应状态码和返回数据。
*/
