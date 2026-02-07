package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config описывает все настройки приложения
type Config struct {
	Printer struct {
		Hostname   string `yaml:"hostname"`
		Password   string `yaml:"password"`
		EncodeWait int    `yaml:"encode_wait"`
		Serial     string `yaml:"serial"`
	} `yaml:"printer"`

	Web struct {
		BindAddress string `yaml:"bind_address"`
		Port        int    `yaml:"port"`
		Username    string `yaml:"username"`
		Password    string `yaml:"password"`
	} `yaml:"web"`

	Timelapse struct {
		Enabled  bool   `yaml:"enabled"`
		Interval int    `yaml:"interval_seconds"`
		SavePath string `yaml:"save_path"`
		Fps      int    `yaml:"fps"`
	} `yaml:"timelapse"`
}

// DefaultConfig возвращает настройки по умолчанию
func DefaultConfig() *Config {
	cfg := &Config{}
	cfg.Printer.EncodeWait = 500
	cfg.Web.BindAddress = "0.0.0.0"
	cfg.Web.Port = 8080
	cfg.Timelapse.Enabled = false
	cfg.Timelapse.Interval = 60
	cfg.Timelapse.SavePath = "timelapse"
	cfg.Timelapse.Fps = 20
	return cfg
}

// Load загружает конфиг из файла. Если файла нет — создает дефолтный.
func Load() (*Config, error) {
	cfg := DefaultConfig()
	dir := filepath.Dir(os.Args[0])
	filename := filepath.Join(dir, "config.yaml")

	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			err = cfg.Save()
			return cfg, err
		}
		return nil, err
	}

	err = yaml.Unmarshal(data, cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

// Save записывает текущие настройки в файл
func (cfg *Config) Save() error {
	dir := filepath.Dir(os.Args[0])
	filename := filepath.Join(dir, "config.yaml")

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}
