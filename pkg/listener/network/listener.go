package network

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/groundcover-com/dynconf/pkg/listener"
	"gopkg.in/yaml.v3"
)

type ConfigurationNetworkListener[Configuration any] struct {
	ctx                 context.Context
	dynamicConfigurable listener.DynamicConfigurable[Configuration]
	options             Options

	cfg Configuration

	metrics *NetworkListenerMetrics

	triggerChannel       chan struct{}
	triggerChannelLock   sync.Mutex
	triggerChannelIsOpen bool

	url        string
	httpClient *http.Client
}

func NewConfigurationNetworkListener[Configuration any](
	id string,
	ctx context.Context,
	dynamicConfigurable listener.DynamicConfigurable[Configuration],
	options Options,
) (*ConfigurationNetworkListener[Configuration], error) {
	return NewConfigurationNetworkListenerWithClient(id, ctx, dynamicConfigurable, options, &http.Client{})
}

func NewConfigurationNetworkListenerWithClient[Configuration any](
	id string,
	ctx context.Context,
	dynamicConfigurable listener.DynamicConfigurable[Configuration],
	options Options,
	httpClient *http.Client,
) (*ConfigurationNetworkListener[Configuration], error) {
	if err := options.Validate(); err != nil {
		return nil, fmt.Errorf("invalid options: %w", err)
	}

	listener := ConfigurationNetworkListener[Configuration]{
		ctx:                  ctx,
		dynamicConfigurable:  dynamicConfigurable,
		options:              options,
		metrics:              NewNetworkListenerMetrics(id),
		triggerChannel:       make(chan struct{}),
		triggerChannelIsOpen: true,
		url:                  options.Request.Build(),
		httpClient:           httpClient,
	}

	go listener.start()

	return &listener, nil
}

// Trigger an immediate request.
// This resets the ticker so that even if a request was due momentarily, it won't be sent.
func (nl *ConfigurationNetworkListener[Configuration]) Trigger() {
	nl.triggerChannelLock.Lock()
	defer nl.triggerChannelLock.Unlock()

	if !nl.triggerChannelIsOpen {
		return
	}

	select {
	case nl.triggerChannel <- struct{}{}:
	default:
	}
}

func (listener *ConfigurationNetworkListener[Configuration]) GetConfiguration() Configuration {
	return listener.cfg
}

func (nl *ConfigurationNetworkListener[Configuration]) closeTriggerChannel() {
	nl.triggerChannelLock.Lock()
	defer nl.triggerChannelLock.Unlock()

	nl.triggerChannelIsOpen = false
	close(nl.triggerChannel)
}

func (nl *ConfigurationNetworkListener[Configuration]) randomInitialJitter() time.Duration {
	return time.Duration(rand.Int63n(int64(nl.options.Interval.MaximumInitialJitter)))
}

func (nl *ConfigurationNetworkListener[Configuration]) fetchConfig() ([]byte, error) {
	req, err := http.NewRequestWithContext(nl.ctx, "GET", nl.url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := nl.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch configuration: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch configuration with status \"%s\"", resp.Status)
	}

	return io.ReadAll(resp.Body)
}

func (nl *ConfigurationNetworkListener[Configuration]) outputConfig(data []byte) error {
	baseConfig, err := nl.options.BaseConfiguration.Unmarshal()
	if err != nil {
		nl.metrics.errorUnmarshalingBaseConfiguration.Inc()
		return fmt.Errorf("failed to unmarshal base configuration: %w", err)
	}

	overrides := make(map[string]any)
	if err := yaml.Unmarshal(data, &overrides); err != nil {
		return fmt.Errorf("failed to unmarshal received configuration: %w", err)
	}

	mergedMap := mergeMaps(baseConfig, overrides)
	mergedYamlConfigurationBytes, err := yaml.Marshal(mergedMap)
	if err != nil {
		return fmt.Errorf("failed to marshal merged configuration into yaml: %w", err)
	}

	var config Configuration
	if err := yaml.Unmarshal(mergedYamlConfigurationBytes, &config); err != nil {
		return fmt.Errorf("failed to unmarshal merged configuration: %w", err)
	}

	if err := nl.dynamicConfigurable.OnConfigurationUpdate(config); err != nil {
		return fmt.Errorf("failed to update configuration: %w", err)
	}

	nl.cfg = config
	return nil
}

func (nl *ConfigurationNetworkListener[Configuration]) fetchConfigCycle() {
	startTime := time.Now()

	data, err := nl.fetchConfig()
	if err != nil {
		nl.metrics.errorFetchingConfiguration.Inc()
		if nl.options.Callback.OnFetchError != nil {
			nl.options.Callback.OnFetchError(fmt.Errorf("error fetching configuration: %w", err))
		}
		return
	}

	if err := nl.outputConfig(data); err != nil {
		if nl.options.Callback.OnFetchError != nil {
			nl.options.Callback.OnFetchError(fmt.Errorf("error outputting configuration: %w", err))
		}
		return
	}

	nl.metrics.requestDuration.UpdateDuration(startTime)
}

func (nl *ConfigurationNetworkListener[Configuration]) initialJitter() bool {
	if nl.options.Interval.MaximumInitialJitter == 0 {
		return false
	}

	ticker := time.NewTicker(nl.randomInitialJitter())
	defer ticker.Stop()

	select {
	case <-nl.ctx.Done():
		return true
	case <-ticker.C:
	case <-nl.triggerChannel:
	}

	return false
}

func (nl *ConfigurationNetworkListener[Configuration]) start() {
	defer nl.closeTriggerChannel()

	if nl.initialJitter() {
		return
	}

	nl.fetchConfigCycle()

	var tickerChannel <-chan time.Time
	tickerResetFunc := func() {}
	if nl.options.Interval.RequestIntervalEnabled {
		ticker := time.NewTicker(nl.options.Interval.RequestInterval)
		defer ticker.Stop()
		tickerChannel = ticker.C
		tickerResetFunc = func() { ticker.Reset(nl.options.Interval.RequestInterval) }
	} else {
		tickerChannelNew := make(chan time.Time)
		defer close(tickerChannelNew)
		tickerChannel = tickerChannelNew
	}

	for {
		select {
		case <-nl.ctx.Done():
			return
		case <-tickerChannel:
			nl.fetchConfigCycle()
		case <-nl.triggerChannel:
			nl.fetchConfigCycle()
			tickerResetFunc()
		}
	}
}

func mergeMaps(base map[string]any, override map[string]any) map[string]interface{} {
	merged := make(map[string]any)
	for k, v := range base {
		merged[k] = v
	}

	for overrideKey, overrideValue := range override {
		baseValueAsMap, baseValueIsMap := base[overrideKey].(map[string]any)
		if !baseValueIsMap {
			merged[overrideKey] = overrideValue
			continue
		}

		overrideValueAsMap, overrideValueIsMap := overrideValue.(map[string]any)
		if !overrideValueIsMap {
			continue
		}
		merged[overrideKey] = mergeMaps(baseValueAsMap, overrideValueAsMap)
	}

	return merged
}
