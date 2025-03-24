package network

import (
	"strings"

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
	requestDuration                    *metrics_types.Summary
	errorUnmarshalingBaseConfiguration *metrics_types.LazyCounter
	errorFetchingConfiguration         *metrics_types.LazyCounter
}

func NewNetworkListenerMetrics(id string) *NetworkListenerMetrics {
	return &NetworkListenerMetrics{
		requestDuration: metrics_factory.CreateInfoSummary(
			strings.Join([]string{networkListenerMetricPrefix, "request_duration"}, "_"),
			map[string]string{idMetricKey: id},
		),
		errorUnmarshalingBaseConfiguration: metrics_factory.CreateErrorCounter(
			errorMetricName,
			map[string]string{errorMetricKey: "error_unmarshaling_base_configuration", idMetricKey: id},
		),
		errorFetchingConfiguration: metrics_factory.CreateErrorCounter(
			errorMetricName,
			map[string]string{errorMetricKey: "error_fetching_configuration", idMetricKey: id},
		),
	}
}
