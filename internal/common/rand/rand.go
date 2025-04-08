// Пакет rand
package rand

import (
	"math/rand"
	"time"
)

// Набор символов по умолчанию
const charset = "abcdefghijklmnopqrstuvwxyz" +
	"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// Числовой рандомайзер
var seededRand *rand.Rand = rand.New(
	rand.NewSource(time.Now().UnixNano()))

// Получить случайную строку из определенного набора символов
func StringWithCharset(length int, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

// Получить случайную строку из набора по умолчанию
func String(length int) string {
	return StringWithCharset(length, charset)
}
