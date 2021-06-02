package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Port    string `toml:"port"`
	BaseDir string `toml:"base_dir"`
}

func parseConfig(path string) Config {
	var c Config

	if _, err := toml.Decode(path, &c); err != nil {
		log.Println("parseConfig", err)
	}
	if !strings.HasPrefix(c.Port, ":") {
		c.Port = fmt.Sprintf(":%s", c.Port)
	}
	return c
}
