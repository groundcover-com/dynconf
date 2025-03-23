package network_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/groundcover-com/dynconf/pkg/listener/network"
)

const (
	fileContents   = `{"config": "mocked_data"}`
	requestTimeout = time.Millisecond * 10
)

type mockServer struct {
	timesCalled int
}

func (server *mockServer) mockServerResponse(w http.ResponseWriter, r *http.Request) {
	server.timesCalled++
	fmt.Fprintln(w, fileContents)
}

func initiateListener(id string) (server *mockServer, channel <-chan []byte, cleanup func(), err error) {
	var httpServer *httptest.Server
	var dataChan *chan []byte
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

	ch := make(chan []byte, 10)
	dataChan = &ch
	options := network.Options{
		Request: network.RequestOptions{
			URL:      httpServer.URL,
			Sections: []string{"section1"},
		},
		Output: network.OutputOptions{
			Mode: network.OutputModeCallback,
			Callback: func(b []byte) error {
				ch <- b
				return nil
			},
		},
		Interval: network.IntervalOptions{
			RequestIntervalEnabled: false,
			MaximumInitialJitter:   0,
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	ctxCancel = cancel

	_, err = network.ListenWithClient(id, ctx, options, httpServer.Client())
	if err != nil {
		return nil, nil, cleanup, fmt.Errorf("Failed to create listener %s: %w", id, err)
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
