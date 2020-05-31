package main

import (
	"errors"
	"log"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

var config *Config

type Config struct {
	Main MainConfig

	// Параметры подключения к БД
	Database DatabaseConfig
}

type MainConfig struct {
	// Путь к каталогу с данными сайта
	SiteDir string

	// Адрес прокси-сервера
	ProxyAddr string

	// Кол-во потоков при обновлении базы данных
	UpdateThreadCount int
}

type DatabaseConfig struct {
	User     string
	Password string
	Host     string
	Port     uint16
	DbName   string
}

func initConfig(configFilePath string) {
	absConfigFilePath, err := filepath.Abs(configFilePath)
	if err != nil {
		absConfigFilePath = configFilePath
	}

	log.Printf("Reading config file: %s...", absConfigFilePath)

	_, err = toml.DecodeFile(defaultConfigPath, &config)
	if err != nil {
		log.Fatalln("read config error:", err)
	}

	err = config.Validate()
	if err != nil {
		log.Fatalln("validate config error:", err)
	}
}

func (config *Config) Validate() error {
	log.Println("Validating config...")

	if config.Main.SiteDir == "" {
		config.Main.SiteDir = defaultSitePath
	}

	if config.Main.UpdateThreadCount <= 0 {
		config.Main.UpdateThreadCount = 1
	}

	if config.Database.User == "" {
		return errors.New("empty database username, check config.Database.User field")
	}

	if config.Database.Host == "" {
		config.Database.Host = "localhost"
	}

	if config.Database.Port == 0 {
		config.Database.Port = 5432
	}

	if config.Database.DbName == "" {
		return errors.New("empty database name, check config.Database.DbName field")
	}

	return nil
}
