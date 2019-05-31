// Wrapper for an INI configuration file.
package config

import (
	"fmt"
	"gopkg.in/ini.v1"
	"os"
	"path/filepath"
)

const formatsSection string = "formats"
const serverSection string = "server"

// Wrapper around the INI file reader
type ConfigFile struct {
	ini *ini.File
}

type FormatDefinition struct {
	Name   string
	Format string
}

// Create a new INI file reader and wrap it.
func New(configPath string) (*ConfigFile, error) {
	f, err := readConfig(configPath)
	if err == nil {
		var config = ConfigFile{ini: f}
		return &config, nil
	} else {
		return nil, err
	}
}

// Get the uri from the config file
func (c *ConfigFile) Uri() string {
	serverSection := c.ini.Section(serverSection)
	return serverSection.Key("uri").String()
}

// Get the username from the config file. Default to an empty string.
func (c *ConfigFile) Username() string {
	serverSection := c.ini.Section(serverSection)
	return serverSection.Key("username").MustString("")
}

// Get the password from the config file. Default to an empty string.
func (c *ConfigFile) Password() string {
	serverSection := c.ini.Section(serverSection)
	return serverSection.Key("password").MustString("")
}

// Get the ignoreCert value from the config file. Default to false.
func (c *ConfigFile) IgnoreCert() bool {
	serverSection := c.ini.Section(serverSection)
	return serverSection.Key("ignoreCert").MustBool(false)
}

// Get the formats from the config file. Adds a final default format just in case.
func (c *ConfigFile) Formats() []FormatDefinition {
	var formats = []FormatDefinition{}

	for _, f := range c.ini.Section(formatsSection).Keys() {
		formats = append(formats, FormatDefinition{Name: f.Name(), Format: f.Value()})
	}
	formats = append(formats, FormatDefinition{Name: "_default", Format: "No Formats Defined>> {{._message_text}}"})

	return formats
}

// Read the configuration file. It's stored in a INI style.
func readConfig(configPath string) (*ini.File, error) {
	configPath, err := filepath.Abs(configPath)
	if err != nil {
		return nil, fmt.Errorf("configuration file not found at %s", configPath)
	}

	if _, err := os.Stat(configPath); err != nil {
		return nil, fmt.Errorf("configuration file not found or not readable at %s", configPath)
	}

	cfg, err := ini.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("configuration file cannot be parsed at %s", configPath)
	}

	return cfg, nil
}