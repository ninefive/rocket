package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/astrocorp42/astroflow-go/log"
)

// DefaultConfigurationFileName is the default configuration file name, without extension
const DefaultConfigurationFileName = ".rocket.toml"

var PredefinedEnv = []string{
	"ROCKET_COMMIT_HASH",
	"ROCKET_LAST_TAG",
	"ROCKET_GIT_REPO",
}

type Config struct {
	Description string            `json:"description" toml:"description"`
	Env         map[string]string `json:"env" toml:"env"`

	// providers
	Script         ScriptConfig          `json:"script,omitempty" toml:"script,omitempty"`
	Heroku         *HerokuConfig         `json:"heroku,omitempty" toml:"heroku,omitempty"`
	GitHubReleases *GitHubReleasesConfig `json:"github_releases,omitempty" toml:"github_releases,omitempty"`
	Docker         *DockerConfig         `json:"docker" toml:"docker"`
	AWSS3          *AWSS3Config          `json:"aws_s3" toml:"aws_s3"`
	ZeitNow        *ZeitNowConfig        `json:"zeit_now" toml:"zeit_now"`
	AWSEB          *AWSEBConfig          `json:"aws_eb" toml:"aws_eb"`
}

// ScriptConfig is the configration for the script provider
type ScriptConfig []string

// HerokuConfig is the configuration for the `heroku` provider
type HerokuConfig struct {
	APIKey    *string `json:"api_key" toml:"api_key"`
	App       *string `json:"app" toml:"app"`
	Directory *string `json:"directory" toml:"directory"`
	Version   *string `json:"version" toml:"version"`
}

// GitHubReleasesConfig is the configuration for the `github_releases` provider
type GitHubReleasesConfig struct {
	Name       *string  `json:"name" toml:"name"`
	Body       *string  `json:"body" toml:"body"`
	Prerelease *bool    `json:"prerelease" toml:"prerelease"`
	Repo       *string  `json:"repo" toml:"repo"`
	APIKey     *string  `json:"api_key" toml:"api_key"`
	Assets     []string `json:"assets" toml:"assets"`
	Tag        *string  `json:"tag" toml:"tag"`
	BaseURL    *string  `json:"base_url" toml:"base_url"`
	UploadURL  *string  `json:"upload_url" toml:"upload_url"`
}

// DockerConfig is the configration for the docker provider
type DockerConfig struct {
	Username *string  `json:"username" toml:"username"`
	Password *string  `josn:"password" toml:"password"`
	Login    *bool    `json:"login" toml:"login"`
	Images   []string `json:"images" toml:"images"`
}

// AWSS3Config is the configration for the aws_s3 provider
type AWSS3Config struct {
	AccessKeyID     *string `json:"access_key_id" toml:"access_key_id"`
	SecretAccessKey *string `json:"secret_access_key" toml:"secret_access_key"`
	Region          *string `json:"region" toml:"region"`
	Bucket          *string `json:"bucket" toml:"bucket"`
	LocalDirectory  *string `json:"local_directory" toml:"local_directory"`
	RemoteDirectory *string `json:"remote_directory" toml:"remote_directory"`
}

// ZeitNowConfig is the configration for the `zeit_now` provider
type ZeitNowConfig struct {
	Token           *string           `json:"token" toml:"token"`
	Directory       *string           `json:"directory" toml:"directory"`
	Env             map[string]string `json:"env" toml:"env"`
	Public          *bool             `json:"public" toml:"public"`
	DeploymentType  *string           `json:"deployment_type" toml:"deployment_type"`
	Name            *string           `json:"name" toml:"name"`
	ForceNew        *bool             `json:"force_new" toml:"force_new"`
	Engines         map[string]string `json:"engines" toml:"engines"`
	SessionAffinity *string           `json:"session_affinity" toml:"session_affinity"`
}

// AWSEBConfig is the configration for the `aws_eb` provider
type AWSEBConfig struct {
	AccessKeyID     *string `json:"access_key_id" toml:"access_key_id"`
	SecretAccessKey *string `json:"secret_access_key" toml:"secret_access_key"`
	Region          *string `json:"region" toml:"region"`
	Application     *string `json:"application" toml:"application"`
	Environment     *string `json:"environment" toml:"environment"`
	S3Bucket        *string `json:"s3_bucket" toml:"s3_bucket"`
	Version         *string `json:"version" toml:"version"`
	Directory       *string `json:"directory" toml:"directory"`
	S3Key           *string `json:"s3_key" toml:"s3_key"`
}

// ExpandEnv 'fix' os.ExpandEnv by allowing to use $$ to escape a dollar e.g: $$HOME -> $HOME
func ExpandEnv(s string) string {
	os.Setenv("ROCKET_DOLLAR", "$")
	return os.ExpandEnv(strings.Replace(s, "$$", "${ROCKET_DOLLAR}", -1))
}

func parseConfig(configFilePath string) (Config, error) {
	var ret Config
	var err error

	file, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return ret, err
	}

	_, err = toml.Decode(string(file), &ret)

	return ret, err
}

func fileExists(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}

// Default return a Config struct filled with default configuration
func Default() Config {
	var ret Config

	ret.Description = "This is a configuration file for rocket: automated software delivery as fast and easy as possible. " +
		"See https://github.com/astrocorp42/rocket"

	return ret
}

// FindConfigFile return the path of the first configuration file found
// it returns an emtpy string if none is found
func FindConfigFile(file string) string {
	if file != "" {
		if fileExists(file) {
			return file
		}
		return ""
	}

	if fileExists(DefaultConfigurationFileName) {
		return DefaultConfigurationFileName
	}

	return ""
}

// Get return the parsed found configuration file or an error
func Get(file string) (Config, error) {
	var err error
	var config Config

	configFilePath := FindConfigFile(file)

	if configFilePath == "" {
		if file == "" {
			return config, fmt.Errorf("%s configuration file not found. Please run \"rocket init\"", DefaultConfigurationFileName)
		}
		return config, fmt.Errorf("%s file not found.", file)
	}

	config, err = parseConfig(configFilePath)
	if err != nil {
		return config, err
	}

	err = setPredefinedEnv()
	if err != nil {
		return config, err
	}

	err = parseEnv(config)
	if err != nil {
		return config, err
	}

	return config, err
}

// set the default env variables
// it does not overwrite the already existing
func setPredefinedEnv() error {
	if os.Getenv("ROCKET_COMMIT_HASH") == "" {
		v := ""
		out, err := exec.Command("git", "rev-parse", "HEAD").Output()
		if err == nil {
			v = strings.TrimSpace(string(out))
		} else {
			log.With("err", err, "var", "ROCKET_COMMIT_HASH").Debug("error setting env var")
		}
		err = os.Setenv("ROCKET_COMMIT_HASH", v)
		if err != nil {
			return err
		}
	}

	if os.Getenv("ROCKET_LAST_TAG") == "" {
		v := ""
		out, err := exec.Command("git", "describe", "--tags", "--abbrev=0").Output()
		if err == nil {
			v = strings.TrimSpace(string(out))
		} else {
			log.With("err", err, "var", "ROCKET_LAST_TAG").Debug("error setting env var")
		}
		err = os.Setenv("ROCKET_LAST_TAG", v)
		if err != nil {
			return err
		}
	}

	if os.Getenv("ROCKET_GIT_REPO") == "" {
		v := ""
		out, err := exec.Command("git", "config", "--get", "remote.origin.url").Output()
		if err == nil {
			parts := strings.Split(strings.TrimSpace(string(out)), ":")
			parts = strings.Split(parts[len(parts)-1], "/")
			repo := parts[len(parts)-2] + "/" + parts[len(parts)-1]
			v = strings.Replace(repo, ".git", "", -1)
		} else {
			log.With("err", err, "var", "ROCKET_GIT_REPO").Debug("error setting env var")
		}
		err = os.Setenv("ROCKET_GIT_REPO", v)
		if err != nil {
			return err
		}
	}

	return nil
}

func isPredefined(key string) bool {
	for _, v := range PredefinedEnv {
		if v == key {
			return true
		}
	}

	return false
}

// parseVariables parse the 'variables' field of the configuration, expand them and set them as env
func parseEnv(conf Config) error {
	if conf.Env != nil {
		for key, value := range conf.Env {
			var err error
			key = strings.ToUpper(key)
			if os.Getenv(key) == "" || isPredefined(key) {
				err = os.Setenv(key, ExpandEnv(value))
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}
