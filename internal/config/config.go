package config

import "github.com/BurntSushi/toml"

// Config has the options required for running relme-auth.
type Config struct {
	Flickr *Strategy `toml:"flickr"`
	GitHub *Strategy `toml:"github"`
}

// Strategy has configuration required for an OAuth/OAuth 2.0 service.
type Strategy struct {
	ID     string `toml:"id"`
	Secret string `toml:"secret"`
}

// Read a TOML formatted configuration file listing the 3rd party authentication
// that can be delegated to.
func Read(path string) (Config, error) {
	var conf Config
	_, err := toml.DecodeFile(path, &conf)
	return conf, err
}
