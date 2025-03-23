package e2e

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/groundcover-com/dynconf/internal/testutils"
	"github.com/groundcover-com/dynconf/pkg/getter"
	fileListener "github.com/groundcover-com/dynconf/pkg/listener/file"
	"github.com/groundcover-com/dynconf/pkg/listener/network"
	networkListener "github.com/groundcover-com/dynconf/pkg/listener/network"
	"github.com/groundcover-com/dynconf/pkg/manager"

	"gopkg.in/yaml.v3"
)

const MockConfigurationWithTwoDepthLevelsYAML = `
First:
  A:
    Value: "some_value_A"
  B:
    Value: true
Second:
  A:
    Value: "some_value_A2"
  B:
    Value: false
`

func TestE2EWithTwoDepthLevels(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "config_*.yaml")
	fileName := tmpFile.Name()
	os.Remove(fileName)

	mgr, err := manager.NewDynamicConfigurationManager[testutils.MockConfigurationWithTwoDepthLevels](
		"testE2ETwoDepthLevels",
	)
	if err != nil {
		t.Fatalf("failed to initiate configuration manager: %v", err)
	}

	topLevelGetter := getter.NewDynamicConfigurationGetter(mgr)
	firstGetter := topLevelGetter.Select("First")
	secondGetter := topLevelGetter.Select("Second")
	firstAGetter := firstGetter.Select("A")
	firstBGetter := firstGetter.Select("B")

	copyConfiguration := testutils.MockConfigurationWithTwoDepthLevels{}
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
		t.Fatalf("failed to register callback on Second configuration: %v", err)
	}
	if err := firstAGetter.Register(callbackFirstA); err != nil {
		t.Fatalf("failed to register callback on FirstA configuration: %v", err)
	}
	if err := firstBGetter.Register(callbackFirstB); err != nil {
		t.Fatalf("failed to register callback on FirstB configuration: %v", err)
	}

	mockConfiguration := testutils.RandomMockConfigurationWithTwoDepthLevels()
	mockConfigurationYaml, err := yaml.Marshal(mockConfiguration)
	if err != nil {
		t.Fatalf("failed to marshal mock configuration: %v", err)
	}

	port := rand.Intn(65535-64535) + 64535
	serverErrCh := make(chan error)
	defer func() {
		serverErr := <-serverErrCh
		if serverErr != nil && !errors.Is(serverErr, http.ErrServerClosed) {
			t.Fatalf("%v", serverErr)
		}
	}()
	server := &http.Server{Addr: fmt.Sprintf(":%d", port)}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-yaml")
		w.Write([]byte(mockConfigurationYaml))
	})
	go func() {
		if err := server.ListenAndServe(); err != nil {
			serverErrCh <- fmt.Errorf("error starting server: %w", err)
			return
		}
		serverErrCh <- nil
	}()
	defer func() {
		if err := server.Close(); err != nil {
			t.Fatalf("failed to close server: %v", err)
		}
	}()

	// wait for http server to get up
	time.Sleep(time.Millisecond * 5)

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
	_, err = fileListener.NewConfigurationFileListener(
		"id",
		fileName,
		mgr,
		fileListener.Options{
			Viper: fileListener.ViperOptions{
				ConfigType: "yaml",
			},
			BaseConfiguration: fileListener.BaseConfigurationOptions{
				Type:   fileListener.BaseConfigurationTypeString,
				String: MockConfigurationWithTwoDepthLevelsYAML,
			},
			Callbacks: fileListener.Callbacks{
				OnConfigurationUpdateSuccess: onConfigurationUpdateSuccess,
			},
		},
	)
	if err != nil {
		t.Fatalf("failed to initiate configuration file listener: %v", err)
	}

	_, err = networkListener.Listen(
		"id",
		context.Background(),
		network.Options{
			Request: network.RequestOptions{
				URL:      fmt.Sprintf("http://localhost:%d", port),
				Sections: []string{"First", "Second"},
			},
			Output: network.OutputOptions{
				Mode: network.OutputModeFile,
				File: fileName,
			},
			Interval: network.IntervalOptions{
				RequestIntervalEnabled: false,
				MaximumInitialJitter:   0,
			},
		},
	)

	select {
	case <-configUpdatedChannel:
	case <-time.After(time.Millisecond * 100):
		t.Fatalf("timeout when awaiting configuration updated callback")
	}

	if !reflect.DeepEqual(copyConfiguration, mockConfiguration) {
		t.Fatalf(
			"after updating configuration, expected %#v but got %#v",
			mockConfiguration,
			copyConfiguration,
		)
	}
}
