package config

const (
	StoreTypeVar    string = "0"
	StoreTypeFile   string = "1"
	DefaultFilename string = "shortener.json"
)

type Config struct {
	StoreType string
	Filename  string
}
