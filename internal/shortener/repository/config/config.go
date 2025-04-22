package config

// Возможные типы хранилища
const (
	StoreTypeVar  string = ""
	StoreTypeFile string = "1"
	StoreTypeDB   string = "2"
)

// Кофигурация store
type Config struct {
	StoreType string
	Filename  string
	DBDsn     string
}
