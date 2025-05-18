// Пакет auth аутентификация
package auth

import (
	"context"
	"net/http"

	"github.com/iurnickita/vigilant-train/internal/common/rand"
	"github.com/iurnickita/vigilant-train/internal/shortener/token"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type Key string

const (
	// UserCodeKey http-ключ для кода пользователя (внутренне использование)
	UserCodeKey = "userCode"
	// UserCodeKey для grpc
	UserCodeKeyGRPC Key = "userCode"
	// cookieUserToken ключ cookie для токена пользователя (внешнее использование)
	cookieUserToken = "shortenerUserToken"
	//
	metadataUserToken = "token"
)

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

// AuthMiddleware прослойка аутентификации для http хендлеров
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

// AuthUnaryInterceptor прослойка аутентификации для gRPC хендлеров
func AuthUnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {

	if info.FullMethod == "Register" {
		return handler(ctx, req)
	}

	// Получение метаданных из контекста
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		var t string
		var userCode string
		var err error
		// Чтение токена из метаданных
		values := md.Get(metadataUserToken)
		if len(values) > 0 {
			// Получение кода пользователя из токена
			t = values[0]
			userCode, err = token.GetUserCode(t)
			if err != nil {
				return nil, status.Errorf(codes.Unauthenticated, err.Error())
			}
		} else {
			// Регистрация.
			// Похоже, так не работает. Метаданные идут только в одну сторону
			// возвращать токен надо явно
			// Создана функция Register

			/* // Создание нового кода пользователя
			userCode = getNewUserCode()
			// Формирование токена
			t, err := token.BuildJWTString(userCode)
			if err != nil {
				return nil, status.Errorf(codes.Unauthenticated, err)
			}
			// Запись токена в метаданные
			md.Set(metadataUserToken, t) */

			return nil, status.Errorf(codes.Unauthenticated, "Unauthenticated. Use Register procedure")
		}
		// Запись кода пользователя в контекст для дальнейшего использования
		ctx = context.WithValue(ctx, UserCodeKeyGRPC, userCode)
	}

	return handler(ctx, req)
}

// Register получение нового токена/кода пользователя
func Register() (string, error) {
	// Создание нового кода пользователя
	userCode := getNewUserCode()
	// Формирование токена
	t, err := token.BuildJWTString(userCode)
	if err != nil {
		return "", err
	}
	return t, nil
}
