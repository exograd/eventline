package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/exograd/eventline/pkg/utils"
)

type Config struct {
	Interface InterfaceConfig `json:"interface,omitempty"`
	API       APIConfig       `json:"api,omitempty"`
	Misc      MiscConfig      `json:"misc,omitempty"`
}

type InterfaceConfig struct {
	Color bool `json:"color"`
}

type APIConfig struct {
	Endpoint string `json:"endpoint,omitempty"`
	Key      string `json:"key,omitempty"`
}

type MiscConfig struct {
	DisableUpdateCheck bool `json:"disable_update_check,omitempty"`
}

func LoadConfig() (*Config, error) {
	config := DefaultConfig()

	filePath := ConfigPath()

	if err := config.CheckPermissions(filePath); err != nil {
		return nil, err
	}

	p.Debug(1, "loading configuration from %s", filePath)

	err := config.LoadFile(filePath)
	if errors.Is(err, os.ErrNotExist) {
		if err := config.Write(); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, fmt.Errorf("cannot load %q: %w", filePath, err)
	}

	return config, nil
}

func (c *Config) Write() error {
	filePath := ConfigPath()

	p.Debug(1, "writing configuration to %s", filePath)

	dirPath := filepath.Dir(filePath)
	if err := os.MkdirAll(dirPath, 0700); err != nil {
		return fmt.Errorf("cannot create directory %s: %w", dirPath, err)
	}

	return c.WriteFile(filePath)
}

func ConfigPath() string {
	if path := os.Getenv("EVCLI_CONFIG_PATH"); path != "" {
		return path
	}

	homePath, err := os.UserHomeDir()
	if err != nil {
		p.Fatal("cannot locate user home directory: %v", err)
	}

	return path.Join(homePath, ".config", "evcli", "config.json")
}

func DefaultConfig() *Config {
	return &Config{
		Interface: InterfaceConfig{
			Color: true,
		},

		API: APIConfig{
			Endpoint: "http://localhost:8085",
		},
	}
}

func (c *Config) CheckPermissions(filePath string) error {
	info, err := os.Stat(filePath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return fmt.Errorf("cannot stat %q: %w", filePath, err)
	}

	mode := uint32(info.Mode().Perm())

	//   8   7   6   5   4   3   2   1   0
	// +---+---+---+---+---+---+---+---+---+
	// | UR| UW| UX| GR| GW| GX| OR| OW| OX|
	// +---+---+---+---+---+---+---+---+---+

	if (mode & (1 << 2)) != 0 {
		return fmt.Errorf("%q must not be world readable", filePath)
	}

	if (mode & (1 << 5)) != 0 {
		return fmt.Errorf("%q must not be group readable", filePath)
	}

	return nil
}

func (c *Config) LoadFile(filePath string) error {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("cannot read file: %w", err)
	}

	return c.LoadData(data)
}

func (c *Config) LoadData(data []byte) error {
	if err := json.Unmarshal(data, c); err != nil {
		return fmt.Errorf("cannot parse json data: %w", err)
	}

	return nil
}

func (c *Config) WriteFile(filePath string) error {
	data, err := utils.JSONEncode(c)
	if err != nil {
		return fmt.Errorf("cannot encode configuration: %w", err)
	}

	if err := ioutil.WriteFile(filePath, data, 0600); err != nil {
		return fmt.Errorf("cannot write file: %w", err)
	}

	return nil
}

func (c *Config) GetEntry(name string) (string, error) {
	e, found := ConfigEntries[name]
	if !found {
		return "", fmt.Errorf("unknown configuration entry %q", name)
	}

	return e.Get(c), nil
}

func (c *Config) SetEntry(name, value string) error {
	e, found := ConfigEntries[name]
	if !found {
		return fmt.Errorf("unknown configuration entry %q", name)
	}

	if err := e.Set(c, value); err != nil {
		return fmt.Errorf("invalid value: %v", err)
	}

	return nil
}
