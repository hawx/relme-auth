package config

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"

	"github.com/BurntSushi/toml"
)

type Config struct {
	GitHub  *Strategy `toml:"github"`
	Twitter *Strategy `toml:"twitter"`
}

type Strategy struct {
	Id     string `toml:"id"`
	Secret string `toml:"secret"`
}

func Read(path string) (Config, error) {
	var conf Config
	_, err := toml.DecodeFile(path, &conf)
	return conf, err
}

func ReadPrivateKey(path string) (*rsa.PrivateKey, error) {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	encoded, _ := pem.Decode(bytes)
	return x509.ParsePKCS1PrivateKey(encoded.Bytes)
}
