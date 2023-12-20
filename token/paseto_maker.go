package token

import (
	"fmt"
	"time"

	"github.com/aead/chacha20poly1305"
	"github.com/o1egl/paseto"
)

//代码提供了PASETO（Platform-Agnostic Security Tokens）版本的令牌制造和验证机制。
//PASETO是JWT的一个安全的替代品，它提供了更加严格的安全保证。

// PasetoMaker is a PASETO token maker
// 它实现了Maker接口，使用PASETO来创建和验证令牌。
// 它包含paseto.V2的实例，这是PASETO库中实现V2版本协议的结构体。
// 它还包含一个对称密钥，用于加密和解密令牌。
type PasetoMaker struct {
	paseto       *paseto.V2
	symmetricKey []byte
}

// NewPasetoMaker creates a new PasetoMaker
// 这是一个工厂函数，用于创建PasetoMaker的实例。
// 它验证传入的对称密钥是否符合chacha20poly1305.KeySize的长度要求，
// 这是使用Chacha20-Poly1305算法进行加密所需的密钥长度。
func NewPasetoMaker(symmetricKey string) (Maker, error) {
	if len(symmetricKey) != chacha20poly1305.KeySize {
		return nil, fmt.Errorf("invalid key size: must be exactly %d characters", chacha20poly1305.KeySize)
	}

	maker := &PasetoMaker{
		paseto:       paseto.NewV2(),
		symmetricKey: []byte(symmetricKey),
	}

	return maker, nil
}

// CreateToken creates a new token for a specific username and duration
// 它创建一个新的PASETO令牌，使用给定的用户名、角色和持续时间。
// 调用NewPayload来创建一个新的令牌负载，包含用户信息和有效期限。
// 使用PASETO的Encrypt方法和对称密钥加密负载，生成令牌。
func (maker *PasetoMaker) CreateToken(username string, role string, duration time.Duration) (string, *Payload, error) {
	payload, err := NewPayload(username, role, duration)
	if err != nil {
		return "", payload, err
	}

	token, err := maker.paseto.Encrypt(maker.symmetricKey, payload, nil)
	return token, payload, err
}

// VerifyToken checks if the token is valid or not
// 它解析并验证一个PASETO令牌的有效性。
// 使用PASETO的Decrypt方法和对称密钥解密令牌，获取负载数据。
// 调用负载的Valid方法检查令牌是否过期或无效。
func (maker *PasetoMaker) VerifyToken(token string) (*Payload, error) {
	payload := &Payload{}

	err := maker.paseto.Decrypt(token, maker.symmetricKey, payload, nil)
	if err != nil {
		return nil, ErrInvalidToken
	}

	err = payload.Valid()
	if err != nil {
		return nil, err
	}

	return payload, nil
}

/*
总体来说，这部分代码将Maker接口的JWT实现替换为PASETO实现，
以提供一个安全性更高的令牌管理方案。PASETO通过提供对称加密和解密来确保令牌的安全性，
同时通过负载中的ExpiredAt字段保证令牌的有效期限。
这使得PasetoMaker成为构建安全认证系统的一个可靠组件。
*/
