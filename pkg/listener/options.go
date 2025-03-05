package listener

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

var (
	ErrInvalidDefaultConfigurationType = errors.New("invalid default configuration type")
)

type DefaultConfigurationType uint32

const (
	DefaultConfigurationTypeString DefaultConfigurationType = iota
	DefaultConfigurationTypeFile
)

type Options struct {
	Viper                ViperOptions
	DefaultConfiguration DefaultConfigurationOptions
}

type DefaultConfigurationOptions struct {
	Type   DefaultConfigurationType
	String string
	File   string
}

func (options *DefaultConfigurationOptions) Init(vpr *viper.Viper) error {
	if options.Type == DefaultConfigurationTypeString {
		if err := vpr.ReadConfig(strings.NewReader(options.String)); err != nil {
			return fmt.Errorf("default configuration can't be loaded from string: %w", err)
		}
		return nil
	}

	if options.Type == DefaultConfigurationTypeFile {
		file, err := os.Open(options.File)
		if err != nil {
			return fmt.Errorf("error opening file: %w", err)
		}
		defer file.Close()

		if err := vpr.ReadConfig(file); err != nil {
			return fmt.Errorf("default configuration can't be loaded from file: %w", err)
		}
		return nil
	}

	return ErrInvalidDefaultConfigurationType
}

type ViperOptions struct {
	EnvKeyReplacer *strings.Replacer
	EnvPrefix      string
	AutomaticEnv   bool
	ConfigType     string
}

func (options *ViperOptions) New() *viper.Viper {
	vpr := viper.New()

	if options.EnvKeyReplacer != nil {
		vpr.SetEnvKeyReplacer(options.EnvKeyReplacer)
	}

	vpr.SetEnvPrefix(options.EnvPrefix)

	if options.AutomaticEnv {
		vpr.AutomaticEnv()
	}

	vpr.SetConfigType(options.ConfigType)

	return vpr
}
