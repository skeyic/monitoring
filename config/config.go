package config

import (
	"github.com/jinzhu/configor"
)

var Config = struct {
	NeuronServer struct {
		URL  string `default:"http://www.xiaxuanli.com:7474" env:"NEURON_SERVER_URL"`
		User string `default:"2db982e4-9492-4202-a4c9-e615e01883f9" env:"NEURON_SERVER_USER"`
	}
}{}

func init() {
	if err := configor.Load(&Config); err != nil {
		panic(err)
	}
}
