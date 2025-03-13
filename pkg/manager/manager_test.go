package manager_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/groundcover-com/dynconf/internal/testutils"
	"github.com/groundcover-com/dynconf/pkg/manager"
)

func newInitiatedConfigurationManagerWithOneDepthLevel(id string) (
	*manager.DynamicConfigurationManager[testutils.MockConfigurationWithOneDepthLevel],
	testutils.MockConfigurationWithOneDepthLevel,
	error,
) {
	mgr := manager.NewDynamicConfigurationManager[testutils.MockConfigurationWithOneDepthLevel](id)
	mockConfiguration := testutils.RandomMockConfigurationWithOneDepthLevel()
	if err := mgr.OnConfigurationUpdate(mockConfiguration); err != nil {
		return nil, mockConfiguration, err
	}

	return mgr, mockConfiguration, nil
}

func TestConfigurationWithDuplicates(t *testing.T) {
	mgr := manager.NewDynamicConfigurationManager[testutils.MockConfigurationWithDuplicates]("testDuplicates")
	mockConfiguration := testutils.RandomMockConfigurationWithDuplicates()

	err := mgr.OnConfigurationUpdate(mockConfiguration)
	if err != nil {
		t.Fatalf("failed to initiate configuration that has duplicates: %v", err)
	}

	copyConfiguration := testutils.MockConfigurationWithDuplicates{}
	callbackA := func(cfg testutils.MockConfigurationA) error {
		copyConfiguration.A = cfg
		return nil
	}
	callbackA2 := func(cfg testutils.MockConfigurationA) error {
		copyConfiguration.A2 = cfg
		return nil
	}
	callbackB := func(cfg testutils.MockConfigurationB) error {
		copyConfiguration.B = cfg
		return nil
	}

	if err := mgr.Register([]string{"A"}, callbackA); err != nil {
		t.Fatalf("failed to register mock configuration A: %#v", err)
	}
	if err := mgr.Register([]string{"A2"}, callbackA2); err != nil {
		t.Fatalf("failed to register mock configuration A2: %#v", err)
	}
	if err := mgr.Register([]string{"B"}, callbackB); err != nil {
		t.Fatalf("failed to register mock configuration B: %#v", err)
	}

	mockConfiguration.A.Value += "bla4"
	mockConfiguration.A2.Value += "bla3"
	mockConfiguration.B.Value = !mockConfiguration.B.Value

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

func TestConfigurationWithTypedef(t *testing.T) {
	mgr := manager.NewDynamicConfigurationManager[testutils.MockConfigurationWithTypedef]("testTypedef")
	mockConfiguration := testutils.RandomMockConfigurationWithTypedef()

	err := mgr.OnConfigurationUpdate(mockConfiguration)
	if err != nil {
		t.Fatalf("failed to initiate configuration that has typedef: %v", err)
	}

	copyConfiguration := testutils.MockConfigurationWithTypedef{}
	callbackA4 := func(cfg testutils.MockConfigurationA4) error {
		copyConfiguration.A4 = cfg
		return nil
	}
	callbackA3 := func(cfg testutils.MockConfigurationA3) error {
		copyConfiguration.A3 = cfg
		return nil
	}
	callbackA2 := func(cfg testutils.MockConfigurationA2) error {
		copyConfiguration.A2 = cfg
		return nil
	}

	if err := mgr.Register([]string{"A4"}, callbackA4); err != nil {
		t.Fatalf("failed to register mock configuration A4: %#v", err)
	}
	if err := mgr.Register([]string{"A3"}, callbackA3); err != nil {
		t.Fatalf("failed to register mock configuration A3: %#v", err)
	}
	if err := mgr.Register([]string{"A2"}, callbackA2); err != nil {
		t.Fatalf("failed to register mock configuration A2: %#v", err)
	}

	mockConfiguration.A4.Value += "bla4"
	mockConfiguration.A3.Value += "bla3"
	mockConfiguration.A2.Value += "bla2"

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

func TestRegisterOnPathThatIsNotInTheConfiguration(t *testing.T) {
	mgr, _, err := newInitiatedConfigurationManagerWithOneDepthLevel("registerItemNotInConf")
	if err != nil {
		t.Fatalf("failed to initiate configuration: %#v", err)
	}

	callbackA4 := func(cfg testutils.MockConfigurationA4) error {
		return nil
	}

	err = mgr.Register([]string{"A4"}, callbackA4)
	if err == nil {
		t.Fatalf("succeeded registering on path that is not in the configuration")
	}
	if !errors.Is(err, manager.ErrNoMatchingFieldFound) {
		t.Fatalf("wrong error when registering on item that is not in the configuration: %#v", err)
	}
}

func TestChangeConfigurationAfterInitiatingButBeforeRegistering(t *testing.T) {
	mgr, mockConfiguration, err := newInitiatedConfigurationManagerWithOneDepthLevel("")
	if err != nil {
		t.Fatalf("failed to initiate configuration: %#v", err)
	}

	copyConfiguration := testutils.MockConfigurationWithOneDepthLevel{}
	callbackB := func(cfg testutils.MockConfigurationB) error {
		copyConfiguration.B = cfg
		return nil
	}

	mockConfiguration.B.Value = !mockConfiguration.B.Value

	if err := mgr.OnConfigurationUpdate(mockConfiguration); err != nil {
		t.Fatalf("failed to update configuration: %#v", err)
	}

	if err := mgr.Register([]string{"B"}, callbackB); err != nil {
		t.Fatalf("failed to register mock configuration: %#v", err)
	}

	if !reflect.DeepEqual(copyConfiguration.B, mockConfiguration.B) {
		t.Fatalf(
			"after registering mock configuration, expected %#v but got %#v",
			mockConfiguration.B,
			copyConfiguration.B,
		)
	}
}

func TestRegisterCallbackThatReceivesWrongArgumentType(t *testing.T) {
	mgr, _, err := newInitiatedConfigurationManagerWithOneDepthLevel("testCallbackWrongArg")
	if err != nil {
		t.Fatalf("failed to initiate configuration: %#v", err)
	}

	callbackA := func(cfg testutils.MockConfigurationA) error {
		return nil
	}

	err = mgr.Register([]string{"B"}, callbackA)
	if err == nil {
		t.Fatalf("Success error when registering bad callback")
	}

	if !errors.Is(err, manager.ErrBadCallback) {
		t.Fatalf("Wrong error when registering bad callback: %#v", err)
	}
}

func TestRestoration(t *testing.T) {
	mgr, mockConfiguration, err := newInitiatedConfigurationManagerWithOneDepthLevel("testRestoration")
	if err != nil {
		t.Fatalf("failed to initiate configuration: %#v", err)
	}

	origConfiguration := mockConfiguration
	copyConfiguration := testutils.MockConfigurationWithOneDepthLevel{}

	shouldFail := false
	initialA := true
	initialB := true
	totalUpdates := 0
	callbackA := func(cfg testutils.MockConfigurationA) error {
		if initialA {
			initialA = false
			copyConfiguration.A = cfg
			return nil
		}
		if shouldFail {
			shouldFail = false
			return errors.ErrUnsupported
		}
		totalUpdates++
		copyConfiguration.A = cfg
		shouldFail = true
		return nil
	}
	callbackB := func(cfg testutils.MockConfigurationB) error {
		if initialB {
			initialB = false
			copyConfiguration.B = cfg
			return nil
		}
		if shouldFail {
			shouldFail = false
			return errors.ErrUnsupported
		}
		totalUpdates++
		copyConfiguration.B = cfg
		shouldFail = true
		return nil
	}

	if err := mgr.Register([]string{"A"}, callbackA); err != nil {
		t.Fatalf("failed to register mock configuration A: %#v", err)
	}

	if err := mgr.Register([]string{"B"}, callbackB); err != nil {
		t.Fatalf("failed to register mock configuration B: %#v", err)
	}

	if !reflect.DeepEqual(copyConfiguration.A, mockConfiguration.A) {
		t.Fatalf(
			"after registering mock configuration A, expected %#v but got %#v",
			mockConfiguration.A,
			copyConfiguration.A,
		)
	}

	if !reflect.DeepEqual(copyConfiguration.B, mockConfiguration.B) {
		t.Fatalf(
			"after registering mock configuration B, expected %#v but got %#v",
			mockConfiguration.B,
			copyConfiguration.B,
		)
	}

	mockConfiguration.B.Value = !mockConfiguration.B.Value
	mockConfiguration.A.Value += "bla"

	err = mgr.OnConfigurationUpdate(mockConfiguration)
	if err == nil {
		t.Fatalf("Succeeded updating to illegal configuration")
	}
	if !errors.Is(err, errors.ErrUnsupported) {
		t.Fatalf("Wrong error when updating to illegal configuration: %#v", err)
	}

	if totalUpdates != 2 {
		t.Fatalf("after failing to update illegal configuration, total updates %d (expected %d)", totalUpdates, 2)
	}

	if !reflect.DeepEqual(copyConfiguration, origConfiguration) {
		t.Fatalf(
			"after failing to update to illegal configuration, expected restoration to original %#v but got %#v",
			origConfiguration.A,
			copyConfiguration.A,
		)
	}
}

func TestChangeConfigurationOfTwoTypes(t *testing.T) {
	mgr, mockConfiguration, err := newInitiatedConfigurationManagerWithOneDepthLevel("testTwoTypes")
	if err != nil {
		t.Fatalf("failed to initiate configuration: %#v", err)
	}

	copyConfiguration := testutils.MockConfigurationWithOneDepthLevel{}
	callbackA := func(cfg testutils.MockConfigurationA) error {
		copyConfiguration.A = cfg
		return nil
	}
	callbackB := func(cfg testutils.MockConfigurationB) error {
		copyConfiguration.B = cfg
		return nil
	}

	if err := mgr.Register([]string{"A"}, callbackA); err != nil {
		t.Fatalf("failed to register mock configuration A: %#v", err)
	}

	if !reflect.DeepEqual(copyConfiguration.A, mockConfiguration.A) {
		t.Fatalf(
			"after registering mock configuration A, expected %#v but got %#v",
			mockConfiguration.A,
			copyConfiguration.A,
		)
	}

	mockConfiguration.A.Value += "bla"
	if err := mgr.OnConfigurationUpdate(mockConfiguration); err != nil {
		t.Fatalf("failed to update configuration after A change: %#v", err)
	}

	if !reflect.DeepEqual(copyConfiguration.A, mockConfiguration.A) {
		t.Fatalf(
			"after updating mock configuration A, expected %#v but got %#v",
			mockConfiguration.A,
			copyConfiguration.A,
		)
	}

	if err := mgr.Register([]string{"B"}, callbackB); err != nil {
		t.Fatalf("failed to register mock configuration B: %#v", err)
	}

	if !reflect.DeepEqual(copyConfiguration.B, mockConfiguration.B) {
		t.Fatalf(
			"after registering mock configuration B, expected %#v but got %#v",
			mockConfiguration.B,
			copyConfiguration.B,
		)
	}

	mockConfiguration.B.Value = !mockConfiguration.B.Value
	mockConfiguration.A.Value += "bla"

	if err := mgr.OnConfigurationUpdate(mockConfiguration); err != nil {
		t.Fatalf("failed to update configuration after A and B changes: %#v", err)
	}

	if !reflect.DeepEqual(copyConfiguration, mockConfiguration) {
		t.Fatalf(
			"after updating mock configuration, expected %#v but got %#v",
			mockConfiguration,
			copyConfiguration,
		)
	}
}

func TestChangeConfigurationOnlyTriggersAlteredCallbacks(t *testing.T) {
	mgr, mockConfiguration, err := newInitiatedConfigurationManagerWithOneDepthLevel("testOnlyTriggerAlteredCallbacks")
	if err != nil {
		t.Fatalf("failed to initiate configuration: %#v", err)
	}

	copyConfiguration := testutils.MockConfigurationWithOneDepthLevel{}
	timesA := 0
	callbackA := func(cfg testutils.MockConfigurationA) error {
		timesA++
		copyConfiguration.A = cfg
		return nil
	}
	timesB := 0
	callbackB := func(cfg testutils.MockConfigurationB) error {
		timesB++
		copyConfiguration.B = cfg
		return nil
	}

	if err := mgr.Register([]string{"A"}, callbackA); err != nil {
		t.Fatalf("failed to register mock configuration A: %#v", err)
	}

	if err := mgr.Register([]string{"B"}, callbackB); err != nil {
		t.Fatalf("failed to register mock configuration B: %#v", err)
	}

	if !reflect.DeepEqual(copyConfiguration, mockConfiguration) {
		t.Fatalf(
			"after registering mock configuration elements, expected %#v but got %#v",
			mockConfiguration,
			copyConfiguration,
		)
	}

	if err := mgr.OnConfigurationUpdate(mockConfiguration); err != nil {
		t.Fatalf("failed to update configuration without any changes: %#v", err)
	}

	if timesA != 1 || timesB != 1 {
		t.Fatalf("callbacks called without changes")
	}

	mockConfiguration.B.Value = !mockConfiguration.B.Value

	if err := mgr.OnConfigurationUpdate(mockConfiguration); err != nil {
		t.Fatalf("failed to update configuration after B change: %#v", err)
	}

	if timesA != 1 {
		t.Fatalf("callback A called when only B changed")
	}

	if timesB != 2 {
		t.Fatalf("callback B called wrong number of times %d", timesB)
	}
}

func TestConfigurationWithTwoDepthLevels(t *testing.T) {
	mgr := manager.NewDynamicConfigurationManager[testutils.MockConfigurationWithTwoDepthLevels]("testTwoDepthLevels")
	mockConfiguration := testutils.RandomMockConfigurationWithTwoDepthLevels()

	if err := mgr.OnConfigurationUpdate(mockConfiguration); err != nil {
		t.Fatalf("failed to initiate configuration that has two depth levels: %v", err)
	}

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

	if err := mgr.Register([]string{"First", "B"}, callbackFirstA); err == nil ||
		!errors.Is(err, manager.ErrBadCallback) {
		t.Fatalf("wrong error when registering bad callback: %v", err)
	}

	if err := mgr.Register([]string{"Second"}, callbackSecond); err != nil {
		t.Fatalf("failed to register callback on Second configuration: %v", err)
	}
	if err := mgr.Register([]string{"First", "A"}, callbackFirstA); err != nil {
		t.Fatalf("failed to register callback on FirstA configuration: %v", err)
	}
	if err := mgr.Register([]string{"First", "B"}, callbackFirstB); err != nil {
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

func TestConfigurationWithPointers(t *testing.T) {
	mgr := manager.NewDynamicConfigurationManager[testutils.MockConfigurationWithPointer]("testPointers")
	mockConfiguration := testutils.RandomMockConfigurationWithPointer()

	if err := mgr.OnConfigurationUpdate(mockConfiguration); err != nil {
		t.Fatalf("failed to initiate configuration that has two depth levels: %v", err)
	}

	copyConfiguration := testutils.RandomMockConfigurationWithPointer()
	badCallback := func(cfg *testutils.MockConfigurationWithA) error {
		copyConfiguration.PtrWithA = cfg
		return nil
	}
	callbackOnPointer := func(cfg testutils.MockConfigurationWithA) error {
		copyConfiguration.PtrWithA = &cfg
		return nil
	}
	callbackWithPointerOnThePath := func(cfg testutils.MockConfigurationA) error {
		copyConfiguration.PtrWithA2.A = cfg
		return nil
	}

	if err := mgr.Register([]string{"PtrWithA"}, badCallback); err == nil || !errors.Is(err, manager.ErrBadCallback) {
		t.Fatalf("wrong error when registering bad callback: %v", err)
	}

	if err := mgr.Register([]string{"PtrWithA"}, callbackOnPointer); err != nil {
		t.Fatalf("failed to register callback on pointer configuration: %v", err)
	}

	if err := mgr.Register([]string{"PtrWithA2", "A"}, callbackWithPointerOnThePath); err != nil {
		t.Fatalf("failed to register callback on configuration that has pointer on the path: %v", err)
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
