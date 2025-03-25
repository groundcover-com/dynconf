package getter_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/groundcover-com/dynconf/internal/testutils"
	"github.com/groundcover-com/dynconf/pkg/getter"
	"github.com/groundcover-com/dynconf/pkg/manager"
)

func TestGetterWithManager(t *testing.T) {
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

	if err := firstBGetter.Register(callbackFirstA); err == nil ||
		!errors.Is(err, manager.ErrBadCallback) {
		t.Fatalf("wrong error when registering bad callback: %v", err)
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

func TestGetterOnTopLevel(t *testing.T) {
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

	topLevelGetter := getter.NewDynamicConfigurationGetter(mgr)

	copyConfiguration := testutils.MockConfigurationWithTwoDepthLevels{}
	callback := func(cfg testutils.MockConfigurationWithTwoDepthLevels) error {
		copyConfiguration = cfg
		return nil
	}

	if err := topLevelGetter.Register(callback); err != nil {
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

func TestGetterMockWithType(t *testing.T) {
	cfgA1 := testutils.RandomMockConfigurationA()
	cfgA2 := testutils.RandomMockConfigurationA()
	cfgA2_2 := testutils.RandomMockConfigurationA()

	var gotCallback func(a testutils.MockConfigurationA) error = nil
	getter := getter.NewDynamicConfigurationGetter(
		getter.NewMockDynamicConfigurationGettableWithType(
			func(path []string, out *testutils.MockConfigurationA) error {
				*out = cfgA1
				return nil
			},
			func(path []string, callback func(a testutils.MockConfigurationA) error) error {
				gotCallback = callback
				return callback(cfgA2)
			},
		),
	)

	var outA1 testutils.MockConfigurationA
	var outA2 testutils.MockConfigurationA
	callbackA2 := func(a testutils.MockConfigurationA) error {
		outA2 = a
		return nil
	}

	if err := getter.Get(&outA1); err != nil {
		t.Fatalf("failed to get A1: %v", err)
	}
	if !reflect.DeepEqual(outA1, cfgA1) {
		t.Fatalf("after updating configuration using get, expected %#v but got %#v", cfgA1, outA1)
	}

	if err := getter.Register(callbackA2); err != nil {
		t.Fatalf("failed to register A2: %v", err)
	}
	if !reflect.DeepEqual(outA2, cfgA2) {
		t.Fatalf("after updating configuration using callback, expected %#v but got %#v", cfgA2, outA2)
	}

	if gotCallback == nil {
		t.Fatalf("didn't get callback")
	}
	if err := gotCallback(cfgA2_2); err != nil {
		t.Fatalf("failed to explicitly call callback: %v", err)
	}
	if !reflect.DeepEqual(outA2, cfgA2_2) {
		t.Fatalf("after updating configuration using callback 2nd time, expected %#v but got %#v", cfgA2_2, outA2)
	}
}
