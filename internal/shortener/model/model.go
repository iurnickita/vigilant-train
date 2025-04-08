// Пакет model. Модели данных
package model

// Shortener - модель сокращенной ссылки
type Shortener struct {
	Key  ShortenerKey
	Data ShortenerData
}

// ShortenerKey - модель сокращенной ссылки. Ключ
type ShortenerKey struct {
	Code string
}

// ShortenerData - модель сокращенной ссылки. Данные
type ShortenerData struct {
	URL  string
	User string
}
