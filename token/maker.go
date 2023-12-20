package token

import (
	"time"
)

//这个接口定义了两个主要的方法：CreateToken用于创建一个新的JWT，
//VerifyToken用于验证接收到的JWT的有效性。这是一个通用的接口，可以有多种实现，
//比如使用不同的加密算法或者令牌格式。

// Maker is an interface for managing tokens
type Maker interface {
	// CreateToken creates a new token for a specific username and duration
	CreateToken(username string, role string, duration time.Duration) (string, *Payload, error)

	// VerifyToken checks if the token is valid or not
	VerifyToken(token string) (*Payload, error)
}
