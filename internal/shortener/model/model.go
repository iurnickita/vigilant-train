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

// Stats - статистические данные
type Stats struct {
	URLs  int `json:"urls"`
	Users int `json:"users"`
}
