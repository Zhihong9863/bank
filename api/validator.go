package api

import (
	"github.com/go-playground/validator/v10"
	"github.com/techschool/bank/util"
)

/*
自定义验证器validCurrency 检查字段的值是否为字符串以及该字符串是否为受支持的货币代码。
然后可以将该验证器注册到 validator 包中，并用于验证用 currency 标记注释的结构体字段

	也就是先在server.go中将其注册到验证器实例	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
			v.RegisterValidation("currency", validCurrency)
		}

然后就可以用了 Currency      string `json:"currency" binding:"required,currency"`
*/
var validCurrency validator.Func = func(fieldLevel validator.FieldLevel) bool {
	if currency, ok := fieldLevel.Field().Interface().(string); ok {
		return util.IsSupportedCurrency(currency)
	}
	return false
}
