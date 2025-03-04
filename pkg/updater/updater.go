package updater

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	metrics_factory "github.com/groundcover-com/metrics/pkg/factory"
	metrics_types "github.com/groundcover-com/metrics/pkg/types"
	"github.com/spf13/viper"
)

const (
	updaterMetricPrefix    = "dynconf_updater_"
	updaterErrorMetricName = updaterMetricPrefix + "error"
	updaterErrorMetricKey  = "error"
	idMetricName           = "id"
)

type DynamicConfigurable[Configuration any] interface {
	OnConfigurationUpdate(newConfiguration Configuration) error
}

type DynamicConfigurationUpdater[Configuration any] struct {
	defaultConfigurationString   string
	configurationFile            string
	dynamicConfigurable          DynamicConfigurable[Configuration]
	onConfigurationUpdateFailure func(error)

	configuration Configuration
	lock          sync.Mutex

	metricFailedToUpdateDynamicConfiguration *metrics_types.LazyCounter
}

func NewDynamicConfigurationUpdater[Configuration any](
	id string,
	vpr *viper.Viper,
	defaultConfiguration string,
	file string,
	dynamicConfigurable DynamicConfigurable[Configuration],
	onConfigurationUpdateFailure func(error),
) (*DynamicConfigurationUpdater[Configuration], error) {
	metricFailedToUpdateDynamicConfiguration := metrics_factory.CreateErrorCounter(
		updaterErrorMetricName,
		map[string]string{
			updaterErrorMetricKey: "failed_to_update_dynamic_configuration",
			"filepath":            file,
			idMetricName:          id,
		},
	)

	updater := &DynamicConfigurationUpdater[Configuration]{
		defaultConfigurationString:               defaultConfiguration,
		configurationFile:                        file,
		dynamicConfigurable:                      dynamicConfigurable,
		metricFailedToUpdateDynamicConfiguration: metricFailedToUpdateDynamicConfiguration,
		onConfigurationUpdateFailure:             onConfigurationUpdateFailure,
	}

	if _, err := os.Stat(file); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("error checking file %s existence: %w", file, err)
		}
		if err := os.WriteFile(file, []byte(""), 0644); err != nil {
			return nil, fmt.Errorf("error writing to file: %w", err)
		}
	}

	if err := updater.update(vpr); err != nil {
		return nil, fmt.Errorf("failed to update initial dynamic configuration: %w", err)
	}

	vpr.WatchConfig()
	vpr.OnConfigChange(func(e fsnotify.Event) {
		if err := updater.update(vpr); err != nil {
			metricFailedToUpdateDynamicConfiguration.Inc()
			onConfigurationUpdateFailure(err)
		}
	})

	return updater, nil
}

func (updater *DynamicConfigurationUpdater[Configuration]) GetConfiguration() Configuration {
	return updater.configuration
}

func (updater *DynamicConfigurationUpdater[Configuration]) update(vpr *viper.Viper) error {
	updater.lock.Lock()
	defer updater.lock.Unlock()

	var defaultConfig Configuration
	updater.initConfigFromStringWithViper(vpr, updater.defaultConfigurationString, &defaultConfig)

	vpr.SetConfigFile(updater.configurationFile)
	if err := vpr.MergeInConfig(); err != nil {
		return fmt.Errorf("error performing configuration merge: %w", err)
	}

	var mergedConfig Configuration
	if err := vpr.Unmarshal(&mergedConfig); err != nil {
		return fmt.Errorf("failed to unmarshal merged configuration: %w", err)
	}

	if err := updater.dynamicConfigurable.OnConfigurationUpdate(mergedConfig); err != nil {
		return fmt.Errorf("failed to update configuration: %w", err)
	}

	updater.configuration = mergedConfig
	return nil
}

func (updater *DynamicConfigurationUpdater[Configuration]) initConfigFromStringWithViper(
	vpr *viper.Viper,
	source string,
	out any,
) error {
	if err := vpr.ReadConfig(strings.NewReader(source)); err != nil {
		return fmt.Errorf("configuration can't be loaded: %w", err)
	}

	if err := vpr.Unmarshal(out); err != nil {
		return fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	return nil
}
