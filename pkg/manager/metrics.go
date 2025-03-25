package manager

import (
	metrics_factory "github.com/groundcover-com/metrics/pkg/factory"
	metrics_types "github.com/groundcover-com/metrics/pkg/types"
)

const (
	managerMetricPrefix = "dynconf_manager_"
	errorMetricName     = managerMetricPrefix + "error"
	errorMetricKey      = "error"
	idMetricKey         = "id"
)

type DynamicConfigurationManagerMetrics struct {
	failedToRestore                    *metrics_types.LazyCounter
	newPathConfigurationDoesNotExist   *metrics_types.LazyCounter
	oldPathConfigurationDoesNotExist   *metrics_types.LazyCounter
	moduleDoesNotAllowNewConfiguration *metrics_types.LazyCounter
}

func NewDynamicConfigurationManagerMetrics(id string) *DynamicConfigurationManagerMetrics {
	return &DynamicConfigurationManagerMetrics{
		failedToRestore: metrics_factory.CreateErrorCounter(
			errorMetricName,
			map[string]string{errorMetricKey: "failed_to_restore", idMetricKey: id},
		),
		newPathConfigurationDoesNotExist: metrics_factory.CreateErrorCounter(
			errorMetricName,
			map[string]string{errorMetricKey: "new_path_configuration_does_not_exist", idMetricKey: id},
		),
		oldPathConfigurationDoesNotExist: metrics_factory.CreateErrorCounter(
			errorMetricName,
			map[string]string{errorMetricKey: "old_path_configuration_does_not_exist", idMetricKey: id},
		),
		moduleDoesNotAllowNewConfiguration: metrics_factory.CreateErrorCounter(
			errorMetricName,
			map[string]string{errorMetricKey: "module_does_not_allow_new_configuration", idMetricKey: id},
		),
	}
}
