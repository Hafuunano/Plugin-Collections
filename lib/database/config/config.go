// Package config provides CRUD for plugin config files under data/config/PLUGIN_NAME/config.yaml.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	// ConfigDirName is the subdir under data root: data/config
	ConfigDirName = "config"
	// ConfigFileName is the default config file name per plugin
	ConfigFileName = "config.yaml"
)

// Path returns the absolute path for a plugin's config file.
// dataDir is the root data directory (e.g. "data"); pluginName is the plugin identifier.
// Result: dataDir/config/pluginName/config.yaml
func Path(dataDir, pluginName string) string {
	return filepath.Join(dataDir, ConfigDirName, pluginName, ConfigFileName)
}

// Dir returns the config directory for a plugin: dataDir/config/pluginName
func Dir(dataDir, pluginName string) string {
	return filepath.Join(dataDir, ConfigDirName, pluginName)
}

// Read reads the plugin config from data/config/pluginName/config.yaml and unmarshals into dest.
// dest should be a pointer to a struct or map. If the file does not exist, no error is returned and dest is unchanged.
func Read(dataDir, pluginName string, dest interface{}) error {
	p := Path(dataDir, pluginName)
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("config read: %w", err)
	}
	if len(data) == 0 {
		return nil
	}
	if err := yaml.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("config unmarshal: %w", err)
	}
	return nil
}

// Save creates or updates the plugin config file (Create/Update). Creates parent dirs if needed.
func Save(dataDir, pluginName string, v interface{}) error {
	dir := Dir(dataDir, pluginName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("config mkdir: %w", err)
	}
	data, err := yaml.Marshal(v)
	if err != nil {
		return fmt.Errorf("config marshal: %w", err)
	}
	p := Path(dataDir, pluginName)
	if err := os.WriteFile(p, data, 0644); err != nil {
		return fmt.Errorf("config write: %w", err)
	}
	return nil
}

// Delete removes the plugin config file. If the plugin config dir is empty, it is removed.
func Delete(dataDir, pluginName string) error {
	p := Path(dataDir, pluginName)
	if err := os.Remove(p); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("config delete: %w", err)
	}
	dir := Dir(dataDir, pluginName)
	_ = os.Remove(dir)
	return nil
}

// Exists reports whether the plugin config file exists.
func Exists(dataDir, pluginName string) bool {
	_, err := os.Stat(Path(dataDir, pluginName))
	return err == nil
}
