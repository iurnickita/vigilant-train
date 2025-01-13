package config

const (
	StoreTypeVar  string = ""
	StoreTypeFile string = "1"
	StoreTypeDB   string = "2"
)

type Config struct {
	StoreType string
	Filename  string
	DBDsn     string
}
