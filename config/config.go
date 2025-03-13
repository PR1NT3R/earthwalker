// Package config handles the config.toml file and the environment variables
package config

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"

	"github.com/BurntSushi/toml"
	"gitlab.com/glatteis/earthwalker/domain"
)

// Read a Config from environment variables and TOML file, and return it
func Read() (domain.Config, error) {
	// defaults
	appPath := AppPath()
	conf := domain.Config{
		ConfigPath:           getEnv("EARTHWALKER_CONFIG_PATH", appPath+"/config.toml"),
		StaticPath:           appPath,
		DBPath:               appPath + "/badger",
		Port:                 "8080",
		TileServerURL:        "https://mt.google.com/vt/lyrs=m&hl=en&x={x}&y={y}&z={z}",
		NoLabelTileServerURL: "https://mt.google.com/vt/lyrs=s&hl=en&x={x}&y={y}&z={z}",
		AllowRemoteMapDeletion: "False",
		AllowRemoteMapCreation: "False",
		IsBehindProxy: "True",
		AllowedIPs: []string{"localhost", "127.0.0.1", "192.168.0.127"},
	}

	// TOML
	tomlData, err := ioutil.ReadFile(conf.ConfigPath)
	if err != nil {
		log.Printf("No config file at '%s', using default configuration.\n", conf.ConfigPath)
	}
	if err := toml.Unmarshal(tomlData, &conf); err != nil {
		return conf, fmt.Errorf("error parsing TOML config file: %v", err)
	}

	// env vars
	conf.Port = getEnv("EARTHWALKER_PORT", conf.Port)
	conf.DBPath = getEnv("EARTHWALKER_DB_PATH", conf.DBPath)
	conf.StaticPath = getEnv("EARTHWALKER_STATIC_PATH", conf.StaticPath)

	return conf, nil
}

func getEnv(key string, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && len(v) > 0 {
		return v
	}
	return fallback
}

// AppPath gets the executable's path.
func AppPath() string {
	appPath, err := os.Executable()
	if err != nil {
		log.Fatal("App path not accessible!")
	}
	return path.Dir(appPath)
}
