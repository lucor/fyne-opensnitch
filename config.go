package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/evilsocket/opensnitch/daemon/rule"
)

var defaultConfig = &uiConfig{
	DefaultTimeout:  15,
	DefaultAction:   rule.Allow,
	DefaultDuration: rule.Restart,
	DefaultOperator: rule.OpProcessPath,
}

type uiConfig struct {
	DefaultTimeout  uint          `json:"default_timeout"`
	DefaultAction   rule.Action   `json:"default_action"`
	DefaultDuration rule.Duration `json:"default_duration"`
	DefaultOperator rule.Operand  `json:"default_operand"`
}

func loadConfigFromFile(configFile string) (*uiConfig, error) {
	if strings.HasPrefix(configFile, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("Could not get the current user's home directory: %v", err)
		}
		configFile = strings.Replace(configFile, "~", home, 1)
	}
	_, err := os.Stat(configFile)
	if os.IsNotExist(err) {
		return createDefaultConfigFile(configFile)
	}

	cfg := &uiConfig{}
	f, err := os.Open(configFile)
	if err != nil {
		return nil, fmt.Errorf("Could not open the config file %q. Error: %v", configFile, err)
	}
	defer f.Close()

	err = json.NewDecoder(f).Decode(cfg)
	if err != nil {
		return nil, fmt.Errorf("Could not read the config file %q. Error: %v", configFile, err)
	}

	return cfg, nil
}

func createDefaultConfigFile(configFile string) (*uiConfig, error) {
	dir := path.Dir(configFile)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return nil, fmt.Errorf("Could not create the config folder %q. Error: %v", dir, err)
	}

	f, err := os.Create(configFile)
	if err != nil {
		return nil, fmt.Errorf("Could not create the config file %q. Error: %v", configFile, err)
	}
	defer f.Close()

	err = json.NewEncoder(f).Encode(defaultConfig)
	if err != nil {
		return nil, fmt.Errorf("Could not write the config file %q. Error: %v", configFile, err)
	}

	return defaultConfig, nil
}
