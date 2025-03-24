package network_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/groundcover-com/dynconf/internal/testutils"
	"github.com/groundcover-com/dynconf/pkg/listener/network"
)

const (
	requestTimeout = time.Millisecond * 10

	fileContents = `
First:
  A:
    Value: "some_value_A"
  B:
    Value: true
`

	base = `
First:
  A:
    Value: "baseFirst"
  B:
    Value: true
Second:
  A:
    Value: "baseSecond"
  B:
    Value: true
`
)

type mockServer struct {
	timesCalled int
}

func (server *mockServer) mockServerResponse(w http.ResponseWriter, r *http.Request) {
	server.timesCalled++
	fmt.Fprintln(w, fileContents)
}

type dynamicConfigurable[T any] struct {
	callback func(T) error
}

func (dc dynamicConfigurable[T]) OnConfigurationUpdate(t T) error {
	return dc.callback(t)
}

func initiateListener[T any](id string) (server *mockServer, channel <-chan T, cleanup func(), err error) {
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

	ms := mockServer{}
	httpServer = httptest.NewServer(http.HandlerFunc(ms.mockServerResponse))

	ch := make(chan T, 10)
	myConfigurable := dynamicConfigurable[testutils.MockConfigurationWithTwoDepthLevels]{
		callback: func(cfg testutils.MockConfigurationWithTwoDepthLevels) error {
			ch <- cfg
			return nil
		},
	}
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
			String: base,
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	ctxCancel = cancel

	_, err = network.NewConfigurationNetworkListenerWithClient[testutils.MockConfigurationWithTwoDepthLevels](
		id,
		ctx,
		myConfigurable,
		options,
		httpServer.Client(),
	)
	if err != nil {
		return nil, nil, cleanup, fmt.Errorf("failed to create listener %s: %w", id, err)
	}

	return &ms, ch, cleanup, nil
}

func TestFetchConfig(t *testing.T) {
	_, channel, cleanup, err := initiateListener("testFetchConfig")
	defer cleanup()
	if err != nil {
		t.Fatalf("failed to initiate listener: %v", err)
	}

	data := <-channel
	if string(data) != fileContents+"\n" { // httptest adds newline
		t.Errorf("expected %q, but got %q", fileContents, string(data))
	}
}
