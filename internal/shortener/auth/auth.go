package auth

import (
	"net/http"

	"github.com/iurnickita/vigilant-train/internal/common/rand"
	"github.com/iurnickita/vigilant-train/internal/shortener/token"
)

const UserCodeKey = "userCode"
const cookieUserToken = "shortenerUserToken"

func getUserCode(w http.ResponseWriter, r *http.Request) (string, error) {

	// куки пользователя
	var userCode string
	tokenCookie, err := r.Cookie(cookieUserToken)
	if err != nil {
		userCode = GetNewUserCode()
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

func GetNewUserCode() string {
	return rand.String(4)
}

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
