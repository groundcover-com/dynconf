package file

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

var (
	ErrInvalidBaseConfigurationType = errors.New("invalid default configuration type")
)

type BaseConfigurationType uint32

const (
	BaseConfigurationTypeString BaseConfigurationType = iota
	BaseConfigurationTypeFile
)

type Options struct {
	Viper ViperOptions
	// The base configuration is the configuration that you start with. Its options define things like where it
	// originates from (etc. a file or a string).
	BaseConfiguration BaseConfigurationOptions
	Callbacks         Callbacks
}

type BaseConfigurationOptions struct {
	Type   BaseConfigurationType
	String string
	File   string
}

func (options *BaseConfigurationOptions) Init(vpr *viper.Viper) error {
	if options.Type == BaseConfigurationTypeString {
		if err := vpr.ReadConfig(strings.NewReader(options.String)); err != nil {
			return fmt.Errorf("base configuration can't be loaded from string: %w", err)
		}
		return nil
	}

	if options.Type == BaseConfigurationTypeFile {
		file, err := os.Open(options.File)
		if err != nil {
			return fmt.Errorf("error opening file: %w", err)
		}
		defer file.Close()

		if err := vpr.ReadConfig(file); err != nil {
			return fmt.Errorf("base configuration can't be loaded from file: %w", err)
		}
		return nil
	}

	return ErrInvalidBaseConfigurationType
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

type Callbacks struct {
	OnConfigurationUpdateFailure func(error)
	OnConfigurationUpdateSuccess func()
}
