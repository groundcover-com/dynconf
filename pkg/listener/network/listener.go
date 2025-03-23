package listener

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"
)

type NetworkListener struct {
	ctx     context.Context
	options Options

	triggerChannel       chan struct{}
	triggerChannelLock   sync.Mutex
	triggerChannelIsOpen bool

	url string
}

func Listen(
	ctx context.Context,
	options Options,
) (*NetworkListener, error) {
	if err := options.Validate(); err != nil {
		return nil, fmt.Errorf("bad options: %w", err)
	}

	listener := NetworkListener{
		ctx:                  ctx,
		options:              options,
		triggerChannel:       make(chan struct{}, 1),
		triggerChannelIsOpen: true,
		url:                  options.Request.Build(),
	}

	go listener.start()

	return &listener, nil
}

// Trigger an immediate request.
// This resets the ticker so that even if a request was due momentarily, it won't be sent.
func (nl *NetworkListener) Trigger() {
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

func (nl *NetworkListener) closeTriggerChannel() {
	nl.triggerChannelLock.Lock()
	defer nl.triggerChannelLock.Unlock()

	nl.triggerChannelIsOpen = false
	close(nl.triggerChannel)
}

func (nl *NetworkListener) randomInitialJitter() time.Duration {
	return time.Duration(rand.Int63n(int64(nl.options.Interval.MaximumInitialJitter)))
}

func (nl *NetworkListener) fetchConfig() ([]byte, error) {
	resp, err := http.Get(nl.url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch configuration: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch configuration with status \"%s\"", resp.Status)
	}

	return io.ReadAll(resp.Body)
}

func (nl *NetworkListener) writeConfigToFile(data []byte) error {
	return os.WriteFile(nl.options.OutputFile, data, 0644)
}

func (nl *NetworkListener) fetchConfigCycle() error {
	data, err := nl.fetchConfig()
	if err != nil {
		return fmt.Errorf("error fetching config: %w", err)
	}

	if err := nl.writeConfigToFile(data); err != nil {
		return fmt.Errorf("error writing config to file: %w", err)
	}

	return nil
}

func (nl *NetworkListener) initialJitter() bool {
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

func (nl *NetworkListener) start() {
	defer nl.closeTriggerChannel()

	if nl.initialJitter() {
		return
	}

	nl.fetchConfigCycle()

	ticker := time.NewTicker(nl.options.Interval.RequestInterval)
	defer ticker.Stop()

	for {
		select {
		case <-nl.ctx.Done():
			return
		case <-ticker.C:
			nl.fetchConfigCycle()
		case <-nl.triggerChannel:
			nl.fetchConfigCycle()
			ticker.Reset(nl.options.Interval.RequestInterval)
		}
	}
}
