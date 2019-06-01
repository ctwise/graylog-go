// Package config is a wrapper for an INI configuration file.
// The package is domain-specific, not general purpose.
package config

import (
	"fmt"
	"gopkg.in/ini.v1"
	"os"
	"path/filepath"
)

const formatsSection string = "formats"
const serverSection string = "server"

// IniFile is a wrapper around the INI file reader
type IniFile struct {
	ini *ini.File
}

// FormatDefinition stores a single format line.
type FormatDefinition struct {
	Name   string
	Format string
}

// New creates a new INI file reader and wraps it.
func New(configPath string) (*IniFile, error) {
	f, err := readConfig(configPath)
	if err == nil {
		var config = IniFile{ini: f}
		return &config, nil
	} else {
		return nil, err
	}
}

// Uri gets the uri from the config file.
func (c *IniFile) Uri() string {
	server := c.ini.Section(serverSection)
	return server.Key("uri").String()
}

// Username gets the username from the config file. Defaults to an empty string.
func (c *IniFile) Username() string {
	server := c.ini.Section(serverSection)
	return server.Key("username").MustString("")
}

// Password gets the password from the config file. Defaults to an empty string.
func (c *IniFile) Password() string {
	server := c.ini.Section(serverSection)
	return server.Key("password").MustString("")
}

// IgnoreCert gets the ignoreCert value from the config file. Defaults to false.
func (c *IniFile) IgnoreCert() bool {
	server := c.ini.Section(serverSection)
	return server.Key("ignoreCert").MustBool(false)
}

// Formats gets the log messages formats from the config file. Adds a final default format case so the user knows that
// no formats were applied successfully.
func (c *IniFile) Formats() (formats []FormatDefinition) {
	for _, f := range c.ini.Section(formatsSection).Keys() {
		formats = append(formats, FormatDefinition{Name: f.Name(), Format: f.Value()})
	}
	formats = append(formats, FormatDefinition{Name: "_default", Format: "No Formats Defined>> {{._message_text}}"})

	return formats
}

// Reads the configuration file. The configuration is stored in a INI style file.
func readConfig(configPath string) (cfg *ini.File, err error) {
	configPath, err = filepath.Abs(configPath)
	if err != nil {
		return nil, fmt.Errorf("configuration file not found at %s", configPath)
	}

	if _, err2 := os.Stat(configPath); err2 != nil {
		return nil, fmt.Errorf("configuration file not found or not readable at %s", configPath)
	}

	cfg, err = ini.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("configuration file cannot be parsed at %s", configPath)
	}

	return cfg, nil
}
