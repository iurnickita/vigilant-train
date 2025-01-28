package model

type Shortener struct {
	Key  ShortenerKey
	Data ShortenerData
}

type ShortenerKey struct {
	Code string
}

type ShortenerData struct {
	URL  string
	User string
}
