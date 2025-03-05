package listener

import (
	"fmt"
	"os"
	"sync"

	"github.com/fsnotify/fsnotify"
	metrics_factory "github.com/groundcover-com/metrics/pkg/factory"
	"github.com/spf13/viper"
)

const (
	listenerMetricPrefix = "dynconf_listener_"
	listenerMetricName   = listenerMetricPrefix + "error"
	listenerMetricKey    = "error"
	idMetricKey          = "id"
	filepathMetricKey    = "filepath"
)

type DynamicConfigurable[Configuration any] interface {
	OnConfigurationUpdate(newConfiguration Configuration) error
}

type DynamicConfigurationListener[Configuration any] struct {
	options                      *Options
	dynamicConfigurable          DynamicConfigurable[Configuration]
	onConfigurationUpdateFailure func(error)

	configuration Configuration
	lock          sync.Mutex
}

func NewDynamicConfigurationListener[Configuration any](
	id string,
	options *Options,
	file string,
	dynamicConfigurable DynamicConfigurable[Configuration],
	onConfigurationUpdateFailure func(error),
) (*DynamicConfigurationListener[Configuration], error) {
	metricFailedToUpdateDynamicConfiguration := metrics_factory.CreateErrorCounter(
		listenerMetricName,
		map[string]string{
			listenerMetricKey: "failed_to_update_dynamic_configuration",
			filepathMetricKey: file,
			idMetricKey:       id,
		},
	)

	listener := &DynamicConfigurationListener[Configuration]{
		options:                      options,
		dynamicConfigurable:          dynamicConfigurable,
		onConfigurationUpdateFailure: onConfigurationUpdateFailure,
	}

	// To watch a configuration file with viper it has to exist when setting the watcher.
	if _, err := os.Stat(file); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("error checking file %s existence: %w", file, err)
		}
		if err := os.WriteFile(file, []byte(""), 0644); err != nil {
			return nil, fmt.Errorf("error writing to file: %w", err)
		}
	}

	vpr := options.Viper.New()
	vpr.SetConfigFile(file)

	if err := listener.update(vpr); err != nil {
		return nil, fmt.Errorf("failed to update initial dynamic configuration: %w", err)
	}

	vpr.WatchConfig()
	vpr.OnConfigChange(func(e fsnotify.Event) {
		if err := listener.update(vpr); err != nil {
			metricFailedToUpdateDynamicConfiguration.Inc()
			if onConfigurationUpdateFailure != nil {
				onConfigurationUpdateFailure(err)
			}
		}
	})

	return listener, nil
}

func (listener *DynamicConfigurationListener[Configuration]) GetConfiguration() Configuration {
	return listener.configuration
}

func (listener *DynamicConfigurationListener[Configuration]) update(vpr *viper.Viper) error {
	listener.lock.Lock()
	defer listener.lock.Unlock()

	if err := listener.options.DefaultConfiguration.Init(vpr); err != nil {
		return fmt.Errorf("failed to initiate default configuration: %w", err)
	}

	if err := vpr.MergeInConfig(); err != nil {
		return fmt.Errorf("error performing configuration merge: %w", err)
	}

	var mergedConfig Configuration
	if err := vpr.Unmarshal(&mergedConfig); err != nil {
		return fmt.Errorf("failed to unmarshal merged configuration: %w", err)
	}

	if err := listener.dynamicConfigurable.OnConfigurationUpdate(mergedConfig); err != nil {
		return fmt.Errorf("failed to update configuration: %w", err)
	}

	listener.configuration = mergedConfig
	return nil
}
