package httpserver

type Config struct {
	Mode string `yaml:"mode" json:"mode"`
	Port string `yaml:"port" json:"port"`
}
