package token

import (
	"errors"
	"fmt"
	"time"

	"github.com/dgrijalva/jwt-go"
)

const minSecretKeySize = 32

/*
这是Maker接口的一个具体实现。它使用HMAC SHA-256算法和一个秘密密钥来签名JWT。
NewJWTMaker函数用于创建JWTMaker的新实例。它需要一个足够长的密钥来确保安全性。
CreateToken方法用给定的用户名、角色和时长生成一个新的JWT，同时返回令牌内容和可能的错误。
VerifyToken方法检查令牌的签名是否有效，并验证令牌是否已经过期。

代码实现了JWT的生成和验证机制。在用户认证的场景中，服务器可以使用JWTMaker来生成包含用户信息的JWT，
并将其发给客户端。客户端后续的请求会携带这个JWT，服务器再使用VerifyToken来校验这个JWT，
以确认用户的身份和请求的合法性。这个过程确保了服务器和客户端之间安全的信息传递，
并且能够有效地处理用户的认证状态。

Maker接口：Payload结构体：JWTMaker结构体
Maker接口定义了操作JWT的标准方法集。
Payload结构体定义了JWT中要包含的数据和验证逻辑。
JWTMaker结构体实现了Maker接口，提供了使用HMAC SHA-256算法签名JWT的具体逻辑。

这三部分共同工作，提供了一个完整的JWT创建和验证的解决方案。
当需要生成一个新的JWT时，JWTMaker的CreateToken会被调用，
它创建一个新的Payload实例并对其进行签名。
当需要验证一个JWT时，JWTMaker的VerifyToken会被调用，它解析JWT，
验证签名，并检查Payload的有效性。
*/
// JWTMaker is a JSON Web Token maker
type JWTMaker struct {
	secretKey string
}

// NewJWTMaker creates a new JWTMaker
func NewJWTMaker(secretKey string) (Maker, error) {
	if len(secretKey) < minSecretKeySize {
		return nil, fmt.Errorf("invalid key size: must be at least %d characters", minSecretKeySize)
	}
	return &JWTMaker{secretKey}, nil
}

// CreateToken creates a new token for a specific username and duration
func (maker *JWTMaker) CreateToken(username string, role string, duration time.Duration) (string, *Payload, error) {
	payload, err := NewPayload(username, role, duration)
	if err != nil {
		return "", payload, err
	}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)
	token, err := jwtToken.SignedString([]byte(maker.secretKey))
	return token, payload, err
}

// VerifyToken checks if the token is valid or not
func (maker *JWTMaker) VerifyToken(token string) (*Payload, error) {
	keyFunc := func(token *jwt.Token) (interface{}, error) {
		_, ok := token.Method.(*jwt.SigningMethodHMAC)
		if !ok {
			return nil, ErrInvalidToken
		}
		return []byte(maker.secretKey), nil
	}

	jwtToken, err := jwt.ParseWithClaims(token, &Payload{}, keyFunc)
	if err != nil {
		verr, ok := err.(*jwt.ValidationError)
		if ok && errors.Is(verr.Inner, ErrExpiredToken) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	payload, ok := jwtToken.Claims.(*Payload)
	if !ok {
		return nil, ErrInvalidToken
	}

	return payload, nil
}
