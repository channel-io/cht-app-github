package config

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

func Init() {
	fillDefaultValues()

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
}

func Load() (*Config, error) {
	stage, err := readStage()
	if err != nil {
		return nil, errors.Wrapf(err, "viper failed to load config")
	}

	viper.SetConfigName(stage)
	viper.SetConfigType("yaml")

	configDir, err := configDir()
	if err != nil {
		return nil, errors.Wrapf(err, "viper failed to load config")
	}

	viper.AddConfigPath(configDir)

	config = &Config{}

	if err := viper.ReadInConfig(); err != nil {
		if !errors.As(err, &viper.ConfigFileNotFoundError{}) {
			return nil, errors.Wrapf(err, "viper failed to load config")
		}
	}

	if err := bindEnvs(config); err != nil {
		return nil, errors.Wrapf(err, "viper failed to load config")
	}

	if err := viper.Unmarshal(config); err != nil {
		return nil, errors.Wrapf(err, "viper failed to load config")
	}

	return config, nil
}

func fillDefaultValues() {
	viper.SetDefault("stage", string(StageDevelopment))
	viper.SetDefault("api.public.http.port", "4000")
	viper.SetDefault("api.metric.http.port", "9090")
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.enableConsole", false)
	viper.SetDefault("log.enableSentry", false)
}

func readStage() (Stage, error) {
	stage := viper.GetString("stage")
	switch stage {
	case string(StageDevelopment):
		return StageDevelopment, nil
	case string(StageExp):
		return StageExp, nil
	case string(StageProduction):
		return StageProduction, nil
	case string(StageTest):
		return StageTest, nil
	default:
		return "", errors.Errorf("invalid stage: %s", stage)
	}
}

func configDir() (string, error) {
	configAbsPath, err := filepath.Abs(".")
	if err != nil {
		return "", errors.Wrapf(err, "fail to read config path")
	}
	configAbsPath += "/config"
	return configAbsPath, nil
}

func bindEnvs(env *Config) error {
	return bindEnvToKey("", reflect.TypeOf(env))
}

func bindEnvToKey(prefix string, dataType reflect.Type) error {
	if dataType.Kind() == reflect.Ptr {
		dataType = dataType.Elem()
	}

	if dataType.Kind() == reflect.Struct {
		for i := 0; i < dataType.NumField(); i++ {
			field := dataType.Field(i)
			nextPrefix := ""
			if len(prefix) > 0 {
				nextPrefix = fmt.Sprintf("%s.%s", prefix, field.Name)
			} else {
				nextPrefix = field.Name
			}
			if err := bindEnvToKey(nextPrefix, field.Type); err != nil {
				return err
			}
		}
	} else {
		err := viper.BindEnv(prefix)
		if err != nil {
			return err
		}
	}

	return nil
}
