package e2e_test

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/groundcover-com/dynconf/internal/testutils"
	"github.com/groundcover-com/dynconf/pkg/getter"
	networkListener "github.com/groundcover-com/dynconf/pkg/listener/network"
	"github.com/groundcover-com/dynconf/pkg/manager"

	"gopkg.in/yaml.v3"
)

type cnf testutils.MockConfigurationWithTwoDepthLevels

func createManagerWithRegisteredConfigurables(
	id string,
) (*manager.DynamicConfigurationManager[cnf], *cnf, error) {
	mgr, err := manager.NewDynamicConfigurationManager[cnf](id)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initiate configuration manager: %w", err)
	}

	topLevelGetter := getter.NewDynamicConfigurationGetter(mgr)
	firstGetter := topLevelGetter.Select("First")
	secondGetter := topLevelGetter.Select("Second")
	firstAGetter := firstGetter.Select("A")
	firstBGetter := firstGetter.Select("B")

	copyConfiguration := cnf{}
	callbackSecond := func(cfg testutils.MockConfigurationWithOneDepthLevel) error {
		copyConfiguration.Second = cfg
		return nil
	}
	callbackFirstA := func(cfg testutils.MockConfigurationA) error {
		copyConfiguration.First.A = cfg
		return nil
	}
	callbackFirstB := func(cfg testutils.MockConfigurationB) error {
		copyConfiguration.First.B = cfg
		return nil
	}

	if err := secondGetter.Register(callbackSecond); err != nil {
		return nil, nil, fmt.Errorf("failed to register callback on Second configuration: %w", err)
	}
	if err := firstAGetter.Register(callbackFirstA); err != nil {
		return nil, nil, fmt.Errorf("failed to register callback on FirstA configuration: %w", err)
	}
	if err := firstBGetter.Register(callbackFirstB); err != nil {
		return nil, nil, fmt.Errorf("failed to register callback on FirstB configuration: %w", err)
	}

	return mgr, &copyConfiguration, nil
}

func startHttpServer(port int, cnf cnf) (*http.Server, error) {
	cnfYaml, err := yaml.Marshal(cnf)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal mock configuration: %w", err)
	}

	serverErrCh := make(chan error)
	serverErrorChannelMutex := sync.Mutex{}
	serverErrorChannelOpen := true
	defer func() {
		serverErrorChannelMutex.Lock()
		defer serverErrorChannelMutex.Unlock()
		serverErrorChannelOpen = false
		close(serverErrCh)
	}()

	server := &http.Server{Addr: fmt.Sprintf(":%d", port)}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-yaml")
		w.Write([]byte(cnfYaml))
	})

	go func() {
		if err := server.ListenAndServe(); err != nil {
			serverErrorChannelMutex.Lock()
			defer serverErrorChannelMutex.Unlock()
			if serverErrorChannelOpen {
				serverErrCh <- fmt.Errorf("error starting server: %w", err)
			}
		}
	}()

	addr := fmt.Sprintf("localhost:%d", port)
	timeout := 50 * time.Millisecond
	deadline := time.Now().Add(timeout)
	ready := false

	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, timeout)
		if err == nil {
			ready = true
			conn.Close()
			break
		}
	}

	if !ready {
		server.Close()
		return nil, fmt.Errorf("server not ready in time")
	}

	select {
	case serverErr := <-serverErrCh:
		server.Close()
		return nil, fmt.Errorf("server error %w", serverErr)
	default:
	}

	return server, nil
}

func TestE2EWithTwoDepthLevels(t *testing.T) {
	mgr, copyConfiguration, err := createManagerWithRegisteredConfigurables("testE2E")
	if err != nil {
		t.Fatalf("failed to create manager with registered configurables: %v", err)
	}

	mockConfiguration := cnf(testutils.RandomMockConfigurationWithTwoDepthLevels())
	port := rand.Intn(65535-64535) + 64535

	server, err := startHttpServer(port, mockConfiguration)
	if err != nil {
		t.Fatalf("failed to start HTTP server: %v", err)
	}
	defer func() {
		if err := server.Close(); err != nil {
			t.Fatalf("failed to close server: %v", err)
		}
	}()

	onConfigurationUpdateSuccessLock := sync.Mutex{}
	configUpdatedChannelOpen := true
	configUpdatedChannel := make(chan struct{})
	defer func() {
		onConfigurationUpdateSuccessLock.Lock()
		defer onConfigurationUpdateSuccessLock.Unlock()
		close(configUpdatedChannel)
		configUpdatedChannelOpen = false
	}()
	onConfigurationUpdateSuccess := func() {
		onConfigurationUpdateSuccessLock.Lock()
		defer onConfigurationUpdateSuccessLock.Unlock()
		if !configUpdatedChannelOpen {
			return
		}
		select {
		case configUpdatedChannel <- struct{}{}:
		default:
		}
	}

	_, err = networkListener.NewConfigurationNetworkListener(
		"id",
		context.Background(),
		mgr,
		networkListener.Options{
			Request: networkListener.RequestOptions{
				URL:      fmt.Sprintf("http://localhost:%d", port),
				Sections: []string{"First", "Second"},
			},
			Interval: networkListener.IntervalOptions{
				RequestIntervalEnabled: false,
				MaximumInitialJitter:   0,
			},
			Callback: networkListener.CallbackOptions{
				OnConfigurationUpdateSuccess: onConfigurationUpdateSuccess,
			},
		},
	)
	if err != nil {
		t.Fatalf("failed to initiate configuration network listener: %v", err)
	}

	select {
	case <-configUpdatedChannel:
	case <-time.After(time.Millisecond * 100):
		t.Fatalf("timeout when awaiting configuration updated callback")
	}

	if !reflect.DeepEqual(*copyConfiguration, mockConfiguration) {
		t.Fatalf(
			"after updating configuration, expected %#v but got %#v",
			mockConfiguration,
			*copyConfiguration,
		)
	}
}
