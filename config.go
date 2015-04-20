package main

import (
	"github.com/juju/errgo"
	"github.com/spf13/viper"
)

func configure() error {
	viper.SetConfigName("config")
	viper.AddConfigPath("$HOME/.wikigo")
	viper.AddConfigPath("/etc/wikigo/")

	err := viper.ReadInConfig()
	if err != nil {
		return errgo.Notef(err, "can not read in config file")
	}

	viper.SetDefault("PagesFolder", "pages")
	viper.SetDefault("Binding", ":12522")

	return nil
}
