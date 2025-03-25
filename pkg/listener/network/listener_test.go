package network_test

import (
	"context"
	"fmt"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/groundcover-com/dynconf/internal/testutils"
	"github.com/groundcover-com/dynconf/pkg/listener"
	"github.com/groundcover-com/dynconf/pkg/listener/network"
	"gopkg.in/yaml.v3"
)

const (
	requestTimeout = time.Millisecond * 10

	mockConfigurationWithTwoDepthLevelsHTTPResponse = `
first:
  a:
    value: "some_value_A"
  b:
    value: true
`

	mockConfigurationWithTwoDepthLevelsBaseConfiguration = `
first:
  a:
    value: "baseFirst"
  b:
    value: true
second:
  a:
    value: "baseSecond"
  b:
    value: true
`

	mockConfigurationWithTwoDepthLevelsExpectedOutcome = `
first:
  a:
    value: "some_value_A"
  b:
    value: true
second:
  a:
    value: "baseSecond"
  b:
    value: true
`
)

func initiateListener[T any](
	id string,
	baseConfiguration string,
	server *testutils.HTTPServerMock,
) (channel <-chan T, cleanup func(), err error) {
	var httpServer *httptest.Server
	var dataChan *chan T
	var ctxCancel context.CancelFunc
	cleanup = func() {
		if server != nil {
			httpServer.Close()
		}
		if dataChan != nil {
			close(*dataChan)
		}
		if ctxCancel != nil {
			ctxCancel()
		}
	}

	httpServer = httptest.NewServer(server)
	ch := make(chan T, 10)
	myConfigurable := listener.NewDynamicConfigurableWithCallback(
		func(cfg T) error {
			ch <- cfg
			return nil
		},
	)
	dataChan = &ch

	options := network.Options{
		Request: network.RequestOptions{
			URL:      httpServer.URL,
			Sections: []string{"section1"},
		},
		Interval: network.IntervalOptions{
			RequestIntervalEnabled: false,
			MaximumInitialJitter:   0,
		},
		BaseConfiguration: network.BaseConfigurationOptions{
			Type:   network.BaseConfigurationTypeString,
			String: baseConfiguration,
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	ctxCancel = cancel

	_, err = network.NewConfigurationNetworkListenerWithClient(
		id,
		ctx,
		myConfigurable,
		options,
		httpServer.Client(),
	)
	if err != nil {
		return nil, cleanup, fmt.Errorf("failed to create listener %s: %w", id, err)
	}

	return ch, cleanup, nil
}

func TestFetchMockConfigurationWithTwoDepthLevels(t *testing.T) {
	server := testutils.NewHTTPServerMock()
	server.SetResponse(mockConfigurationWithTwoDepthLevelsHTTPResponse)
	channel, cleanup, err := initiateListener[testutils.MockConfigurationWithTwoDepthLevels](
		"testFetchConfigurationWithTwoDepthLevels",
		mockConfigurationWithTwoDepthLevelsBaseConfiguration,
		server,
	)
	defer cleanup()
	if err != nil {
		t.Fatalf("failed to initiate listener: %v", err)
	}

	var data testutils.MockConfigurationWithTwoDepthLevels
	select {
	case data = <-channel:
	case <-time.After(time.Millisecond * 100):
		t.Fatalf("data channel timeout")
	}

	var expected testutils.MockConfigurationWithTwoDepthLevels
	if err := yaml.Unmarshal([]byte(mockConfigurationWithTwoDepthLevelsExpectedOutcome), &expected); err != nil {
		t.Fatalf("failed to prepare expected outcome: %v", err)
	}

	if !reflect.DeepEqual(data, expected) {
		t.Fatalf("after updating configuration, expected %#v but got %#v", expected, data)
	}
}
