package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Database struct {
	Driver string `yaml:"driver"`
	DSN    string `yaml:"dsn"`
}

type Storage struct {
	ContentRoot string `yaml:"content_root"`
}

type Schedule struct {
	Cron string `yaml:"cron"`
}

type Portal struct {
	Addr string `yaml:"addr"`
}

type FetcherEntry struct {
	Name    string                 `yaml:"name"`
	Enabled bool                   `yaml:"enabled"`
	Queries []string               `yaml:"queries"`
	Config  map[string]interface{} `yaml:"config"`
}

type Config struct {
	Database Database       `yaml:"database"`
	Storage  Storage        `yaml:"storage"`
	Schedule Schedule       `yaml:"schedule"`
	Portal   Portal         `yaml:"portal"`
	Fetchers []FetcherEntry `yaml:"fetchers"`
}

func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("parse yaml: %w", err)
	}
	c.applyDefaults()
	if err := c.validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

func (c *Config) applyDefaults() {
	if c.Database.Driver == "" {
		c.Database.Driver = "sqlite"
	}
	if c.Database.DSN == "" {
		c.Database.DSN = "data/aggregator.db"
	}
	if c.Storage.ContentRoot == "" {
		c.Storage.ContentRoot = "data/articles"
	}
	if c.Schedule.Cron == "" {
		c.Schedule.Cron = "0 2 * * *"
	}
	if c.Portal.Addr == "" {
		c.Portal.Addr = "127.0.0.1:8080"
	}
}

func (c *Config) validate() error {
	for i, f := range c.Fetchers {
		if f.Name == "" {
			return fmt.Errorf("fetchers[%d].name is required", i)
		}
	}
	return nil
}
