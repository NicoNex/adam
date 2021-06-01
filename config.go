package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Port    string `toml:port`
	BaseDir string `toml:base_dir`
}

func parseConfig(path string) Config {
	var c Config
	var port int
	var basedir string

	if _, err := toml.Decode(path, &c); err != nil {
		log.Println("parseConfig", err)
	}

	flag.IntVar(&port, "p", 0, "The port Adam will listen to.")
	flag.StringVar(&basedir, "d", "", "The directory Adam will use as root directory.")
	flag.Parse()

	if port != 0 {
		c.Port = fmt.Sprintf(":%d", port)
	}
	if basedir != "" {
		c.BaseDir = basedir
	}
	if !strings.HasPrefix(c.Port, ":") {
		c.Port = fmt.Sprintf(":%s", c.Port)
	}
	return c
}
