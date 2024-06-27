package httpserver

type Config struct {
	Enabled bool   `yaml:"enabled" json:"enabled"`
	Mode    string `yaml:"mode" json:"mode"`
	Port    string `yaml:"port" json:"port"`
}
