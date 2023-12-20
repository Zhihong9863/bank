package api

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	db "github.com/techschool/bank/db/sqlc"
	"github.com/techschool/bank/token"
)

/*
定义了一个transferRequest结构体，它用来表示一个转账请求的数据结构，
包含了转出账户ID (FromAccountID)、转入账户ID (ToAccountID)、转账金额 (Amount)
和货币类型 (Currency)。这些字段都使用了binding标签来指定验证规则。
*/
type transferRequest struct {
	FromAccountID int64  `json:"from_account_id" binding:"required,min=1"`
	ToAccountID   int64  `json:"to_account_id" binding:"required,min=1"`
	Amount        int64  `json:"amount" binding:"required,gt=0"`
	Currency      string `json:"currency" binding:"required,currency"`
}

/*
createTransfer函数是处理POST请求的HTTP处理函数。它会尝试从请求的JSON体中绑定（解析）数据到transferRequest结构体实例。

如果JSON绑定失败（比如请求体格式不正确或缺失必要字段），函数会返回HTTP 400（错误请求）响应。

接着，函数会验证转出和转入的账户是否存在，并且货币类型是否匹配。这是通过调用validAccount函数完成的。
*/
func (server *Server) createTransfer(ctx *gin.Context) {
	var req transferRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	//在转账API端点，代码首先验证请求的起始账户是否属于已认证的用户。
	//如果不属于，将阻止操作并返回HTTP 401错误。然后，它会验证目标账户的有效性，
	//但不检查目标账户的所有者，因为用户可以向任何账户转账。
	fromAccount, valid := server.validAccount(ctx, req.FromAccountID, req.Currency)
	if !valid {
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	if fromAccount.Owner != authPayload.Username {
		err := errors.New("from account doesn't belong to the authenticated user")
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	_, valid = server.validAccount(ctx, req.ToAccountID, req.Currency)
	if !valid {
		return
	}

	//如果所有验证都通过了，函数会构造一个TransferTxParams结构体实例，并调用TransferTx方法进行转账操作。
	arg := db.TransferTxParams{
		FromAccountID: req.FromAccountID,
		ToAccountID:   req.ToAccountID,
		Amount:        req.Amount,
	}

	/*
		ransferTx是在数据库层面处理事务的函数。如果转账操作成功，会返回转账的结果。
		如果操作失败，比如数据库错误，会返回HTTP 500（服务器内部错误）响应。
		如果转账成功，函数会返回HTTP 200响应，以及一个包含转账详情的JSON对象。
	*/
	result, err := server.store.TransferTx(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, result)

}

/*
validAccount函数会查询数据库，检查给定的账户ID是否存在，并且货币类型是否与请求中的货币类型相符。
如果账户不存在或货币类型不匹配，会返回相应的HTTP错误响应。
*/
func (server *Server) validAccount(ctx *gin.Context, accountID int64, currency string) (db.Account, bool) {
	account, err := server.store.GetAccount(ctx, accountID)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return account, false
		}

		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return account, false
	}

	if account.Currency != currency {
		err := fmt.Errorf("account [%d] currency mismatch: %s vs %s", account.ID, account.Currency, currency)
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return account, false
	}

	return account, true
}
