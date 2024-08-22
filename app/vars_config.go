package app

import (
	"os"

	"gopkg.in/yaml.v2"
)

type VarsConfig struct {
	DefaultVars map[string]string            `yaml:"default_vars"`
	ProductVars map[string]map[string]string `yaml:"product_vars"`
}

func LoadVarsConfigYaml(filename string) (*VarsConfig, error) {
	fd, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer fd.Close()

	cfg := &VarsConfig{}
	if err := yaml.NewDecoder(fd).Decode(&cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
