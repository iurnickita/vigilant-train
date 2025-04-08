// Пакет auth аутентификация
package auth

import (
	"net/http"

	"github.com/iurnickita/vigilant-train/internal/common/rand"
	"github.com/iurnickita/vigilant-train/internal/shortener/token"
)

// UserCodeKey http-ключ для кода пользователя (внутренне использование)
const UserCodeKey = "userCode"

// cookieUserToken ключ cookie для токена пользователя (внешнее использование)
const cookieUserToken = "shortenerUserToken"

// getUserCode получает/присваивает код пользователя
func getUserCode(w http.ResponseWriter, r *http.Request) (string, error) {

	// куки пользователя
	var userCode string
	tokenCookie, err := r.Cookie(cookieUserToken)
	if err != nil {
		userCode = getNewUserCode()
		tokenString, err := token.BuildJWTString(userCode)
		if err != nil {
			return "", err
		}
		tokenCookie := http.Cookie{
			Name:  cookieUserToken,
			Value: tokenString,
		}
		http.SetCookie(w, &tokenCookie)
	} else {
		userCode, err = token.GetUserCode(tokenCookie.Value)
		if err != nil {
			return "", err
		}
	}
	return userCode, nil
}

// GetNewUserCode генерирует новый код пользователя
func getNewUserCode() string {
	return rand.String(4)
}

// AuthMiddleware прослойка аутентификации для хендлеров
func AuthMiddleware(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// получение id пользователя
		userCode, err := getUserCode(w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		// записываем
		r.Header.Set(UserCodeKey, userCode)

		// передаём управление хендлеру
		h.ServeHTTP(w, r)
	}
}
