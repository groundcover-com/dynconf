package listener

import (
	metrics_factory "github.com/groundcover-com/metrics/pkg/factory"
	metrics_types "github.com/groundcover-com/metrics/pkg/types"
)

const (
	networkListenerMetricPrefix = "dynconf_listener_network_"
	errorMetricName             = networkListenerMetricPrefix + "error"
	errorMetricKey              = "error"
	idMetricKey                 = "id"
)

type NetworkListenerMetrics struct {
	errorFetchingConfiguration      *metrics_types.LazyCounter
	errorWritingConfigurationToFile *metrics_types.LazyCounter
}

func NewNetworkListenerMetrics(id string) *NetworkListenerMetrics {
	return &NetworkListenerMetrics{
		errorFetchingConfiguration: metrics_factory.CreateErrorCounter(
			errorMetricName,
			map[string]string{errorMetricKey: "error_fetching_configuration", idMetricKey: id},
		),
		errorWritingConfigurationToFile: metrics_factory.CreateErrorCounter(
			errorMetricName,
			map[string]string{errorMetricKey: "error_wring_configuration_to_file", idMetricKey: id},
		),
	}
}
