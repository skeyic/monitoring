package config

import (
	"github.com/jinzhu/configor"
)

var Config = struct {
	DebugMode bool `default:"true"`
	Port      uint `default:"8000"`

	LookBack bool `default:"false" env:"LOOK_BACK"`
}{}

func init() {
	if err := configor.Load(&Config); err != nil {
		panic(err)
	}
}
