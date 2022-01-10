/*
 *     Copyright 2020 The Dragonfly Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package nginx

import (
	"fmt"
	"path/filepath"
)

// Config defines nginx configuration.
type Config struct {
	Binary string `yaml:"binary"`

	// DownloadPort is the port for download files from cdn.
	// default: 8001
	DownloadPort int `yaml:"downloadPort" mapstructure:"downloadPort"`

	StorageLocation string `yaml:"storageLocation" mapstructure:"storageLocation"`

	Root bool `yaml:"root"`

	// Name defines the default nginx template for each component.
	Name string `yaml:"name"`

	// TemplatePath takes precedence over Name, overwrites default template.
	TemplatePath string `yaml:"template_path"`

	CacheDir string `yaml:"cache_dir"`

	LogDir string `yaml:"log_dir"`

	// Optional log path overrides.
	StdoutLogPath string `yaml:"stdout_log_path"`
	AccessLogPath string `yaml:"access_log_path"`
	ErrorLogPath  string `yaml:"error_log_path"`
}

// applyDefaults 设置nginx的默认配置
func (c *Config) applyDefaults() error {
	if c.Binary == "" {
		c.Binary = "/usr/sbin/nginx"
	}

	if c.DownloadPort == 0 {
		c.DownloadPort = DefaultDownloadPort
	}

	if c.StdoutLogPath == "" {
		if c.LogDir == "" {
			return errors.New("one of log_dir or stdout_log_path must be set")
		}
		c.StdoutLogPath = filepath.Join(c.LogDir, "nginx-stdout.log")
	}
	if c.AccessLogPath == "" {
		if c.LogDir == "" {
			return errors.New("one of log_dir or access_log_path must be set")
		}
		c.AccessLogPath = filepath.Join(c.LogDir, "nginx-access.log")
	}
	if c.ErrorLogPath == "" {
		if c.LogDir == "" {
			return errors.New("one of log_dir or error_log_path must be set")
		}
		c.ErrorLogPath = filepath.Join(c.LogDir, "nginx-error.log")
	}
	return nil
}

func (c Config) Validate() []error {
	var errors []error
	if c.DownloadPort > 65535 || c.DownloadPort < 1024 {
		errors = append(errors, fmt.Errorf("rpc server DownloadPort must be between 0 and 65535, inclusive. but is: %d", c.DownloadPort))
	}
	return errors
}

const (
	// DefaultDownloadPort is the default port for download files from cdn.
	DefaultDownloadPort = 8001
)
