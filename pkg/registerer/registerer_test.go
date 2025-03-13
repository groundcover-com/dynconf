package registerer_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/groundcover-com/dynconf/internal/testutils"
	"github.com/groundcover-com/dynconf/pkg/manager"
	"github.com/groundcover-com/dynconf/pkg/registerer"
)

func TestRegistererWithManager(t *testing.T) {
	mgr, err := manager.NewDynamicConfigurationManager[testutils.MockConfigurationWithTwoDepthLevels](
		"testTwoDepthLevels",
	)
	if err != nil {
		t.Fatalf("failed to initiate configuration manager: %v", err)
	}

	mockConfiguration := testutils.RandomMockConfigurationWithTwoDepthLevels()

	if err := mgr.OnConfigurationUpdate(mockConfiguration); err != nil {
		t.Fatalf("failed to initiate configuration that has two depth levels: %v", err)
	}

	topLevelRegisterer := registerer.NewDynamicConfigurationRegisterer(mgr)

	firstRegisterer := topLevelRegisterer.Under("First")
	secondRegisterer := topLevelRegisterer.Under("Second")
	firstARegisterer := firstRegisterer.Under("A")
	firstBRegisterer := firstRegisterer.Under("B")

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

	if err := firstBRegisterer.Register(callbackFirstA); err == nil ||
		!errors.Is(err, manager.ErrBadCallback) {
		t.Fatalf("wrong error when registering bad callback: %v", err)
	}

	if err := secondRegisterer.Register(callbackSecond); err != nil {
		t.Fatalf("failed to register callback on Second configuration: %v", err)
	}
	if err := firstARegisterer.Register(callbackFirstA); err != nil {
		t.Fatalf("failed to register callback on FirstA configuration: %v", err)
	}
	if err := firstBRegisterer.Register(callbackFirstB); err != nil {
		t.Fatalf("failed to register callback on FirstB configuration: %v", err)
	}

	if err := mgr.OnConfigurationUpdate(mockConfiguration); err != nil {
		t.Fatalf("failed to update configuration: %#v", err)
	}

	if !reflect.DeepEqual(copyConfiguration, mockConfiguration) {
		t.Fatalf(
			"after updating configuration, expected %#v but got %#v",
			mockConfiguration,
			copyConfiguration,
		)
	}
}

func TestRegistererOnTopLevel(t *testing.T) {
	mgr, err := manager.NewDynamicConfigurationManager[testutils.MockConfigurationWithTwoDepthLevels](
		"testRegisterOnTopLevel",
	)
	if err != nil {
		t.Fatalf("failed to initiate configuration manager: %v", err)
	}
	mockConfiguration := testutils.RandomMockConfigurationWithTwoDepthLevels()

	if err := mgr.OnConfigurationUpdate(mockConfiguration); err != nil {
		t.Fatalf("failed to initiate configuration that has two depth levels: %v", err)
	}

	topLevelRegisterer := registerer.NewDynamicConfigurationRegisterer(mgr)

	copyConfiguration := testutils.MockConfigurationWithTwoDepthLevels{}
	callback := func(cfg testutils.MockConfigurationWithTwoDepthLevels) error {
		copyConfiguration = cfg
		return nil
	}

	if err := topLevelRegisterer.Register(callback); err != nil {
		t.Fatalf("failed to register callback: %v", err)
	}

	mockConfiguration.First.A.Value += "asda"

	if err := mgr.OnConfigurationUpdate(mockConfiguration); err != nil {
		t.Fatalf("failed to update configuration: %#v", err)
	}

	if !reflect.DeepEqual(copyConfiguration, mockConfiguration) {
		t.Fatalf(
			"after updating configuration, expected %#v but got %#v",
			mockConfiguration,
			copyConfiguration,
		)
	}
}
