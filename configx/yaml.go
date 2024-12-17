package configx

import (
	"github.com/niudaii/goutil/constants"
	"github.com/spf13/viper"
)

func ParseYaml(filename string, val interface{}) (err error) {
	v := viper.New()
	v.SetConfigFile(filename)
	v.SetConfigType(constants.YamlFormat)
	err = v.ReadInConfig()
	if err != nil {
		return
	}
	err = v.Unmarshal(&val)
	return
}
