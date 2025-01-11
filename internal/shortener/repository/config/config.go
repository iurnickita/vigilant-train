package config

const (
	StoreTypeVar    string = ""
	StoreTypeFile   string = "1"
	StoreTypeDB     string = "2"
	DefaultFilename string = "shortener.json"
)

type Config struct {
	StoreType string
	Filename  string
	DB_DSN    string
}
