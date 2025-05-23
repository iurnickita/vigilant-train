// Пакет token. Обработка JWT-токена
package token

import (
	"fmt"

	"github.com/golang-jwt/jwt/v4"
)

// Claims - утверждения. Тело токена
type Claims struct {
	jwt.RegisteredClaims
	UserCode string
}

// const TOKEN_EXP = time.Hour * 3

const secretKey = "supersecretkey"

// BuildJWTString формирует токен с кодом пользователя
func BuildJWTString(UserCode string) (string, error) {
	// создаём новый токен с алгоритмом подписи HS256 и утверждениями — Claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		/* RegisteredClaims: jwt.RegisteredClaims{
		    // когда создан токен
		    ExpiresAt: jwt.NewNumericDate(time.Now().Add(TOKEN_EXP)),
		}, */
		// собственное утверждение
		UserCode: UserCode,
	})

	// создаём строку токена
	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", err
	}

	// возвращаем строку токена
	return tokenString, nil
}

// GetUserCode читает код пользователя из токена
func GetUserCode(tokenString string) (string, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims,
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(secretKey), nil
		})
	if err != nil {
		return "", err
	}

	if !token.Valid {
		fmt.Println("Token is not valid")
		return "", err
	}

	fmt.Println("Token os valid")
	return claims.UserCode, nil
}
