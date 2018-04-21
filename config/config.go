package config

import "github.com/BurntSushi/toml"

type Config struct {
	Flickr  *Strategy `toml:"flickr"`
	GitHub  *Strategy `toml:"github"`
	Twitter *Strategy `toml:"twitter"`
}

type Strategy struct {
	Id     string `toml:"id"`
	Secret string `toml:"secret"`
}

// Read a TOML formatted configuration file listing the 3rd party authentication
// that can be delegated to.
func Read(path string) (Config, error) {
	var conf Config
	_, err := toml.DecodeFile(path, &conf)
	return conf, err
}
