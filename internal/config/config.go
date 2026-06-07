package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	configFilePerm = 0o644
	configDirPerm  = 0o755
)

type Config struct {
	Commands  CommandsConfig  `yaml:"commands"`
	Privacy   PrivacyConfig   `yaml:"privacy"`
	Recording RecordingConfig `yaml:"recording"`
}

type CommandsConfig struct {
	PromptPatterns []string `yaml:"promptPatterns"`
}

type PrivacyConfig struct {
	Redact         bool     `yaml:"redact"`
	RedactPatterns []string `yaml:"redactPatterns"`
}

type RecordingConfig struct {
	StopKey string `yaml:"stopKey"`
}

func Default() Config {
	return Config{}
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Default(), nil
		}
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config %s: %w", path, err)
	}
	return cfg, nil
}

func Save(path string, cfg Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), configDirPerm); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	if err := os.WriteFile(path, data, configFilePerm); err != nil {
		return fmt.Errorf("write config %s: %w", path, err)
	}
	return nil
}
