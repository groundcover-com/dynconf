package manager

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"

	agent_metrics "github.com/groundcover-com/flora/pkg/metrics"
	"github.com/groundcover-com/metrics"
)

const (
	pathSeparator = "."

	managerMetricPrefix    = agent_metrics.PromethuesMetricsPrefix + "config_dynamicmanager_"
	managerErrorMetricName = managerMetricPrefix + "error"
	managerErrorMetricKey  = "error"
	managerIDMetricKey     = "id"
)

var (
	// Registering to listen on updates to a type that doesn't exist within the configuration will return this error.
	ErrNoMatchingTypeFound = errors.New("no matching type found")

	// The configuration manager allows only specific types of configurations to be used. This error indicates that a
	// wrong configuration type is used, and it can only be returned on the first configuration update.
	ErrWrongConfigurationType = errors.New("wrong configuration type")

	// If the configuration has more than one instance of a struct type inside it, and a user listens on updates to this
	// type, the behaviour is undefined as to which instance they listen to. Therefore, we disallow dynamic
	// configuration with two instances of the same type.
	//
	// Working around that is easy - simply define a new type for the same struct.
	// If needed, the user can cast between the different types when their callback is triggered.
	ErrConfigurationHasDuplicates = errors.New("configuration has duplicates")

	// When registering, the callback is very unspecific. However, in fact it has to have a very specific definition.
	// If a wrong callback is given, this error is returned.
	ErrBadCallback = errors.New("bad callback")

	errInvalidTypeToIterate = errors.New("invalid type to iterate")
)

type DynamicConfigurationManagerMetrics struct {
	failedToRestore                    *metrics.LazyCounter
	newPathConfigurationDoesNotExist   *metrics.LazyCounter
	oldPathConfigurationDoesNotExist   *metrics.LazyCounter
	moduleDoesNotAllowNewConfiguration *metrics.LazyCounter
	invalidNewConfigurationType        *metrics.LazyCounter
}

func NewDynamicConfigurationManagerMetrics(id string) *DynamicConfigurationManagerMetrics {
	return &DynamicConfigurationManagerMetrics{
		failedToRestore: agent_metrics.CreateErrorCounter(
			managerErrorMetricName,
			map[string]string{managerErrorMetricKey: "failed_to_restore", managerIDMetricKey: id},
		),
		newPathConfigurationDoesNotExist: agent_metrics.CreateErrorCounter(
			managerErrorMetricName,
			map[string]string{managerErrorMetricKey: "new_path_configuration_does_not_exist", managerIDMetricKey: id},
		),
		oldPathConfigurationDoesNotExist: agent_metrics.CreateErrorCounter(
			managerErrorMetricName,
			map[string]string{managerErrorMetricKey: "old_path_configuration_does_not_exist", managerIDMetricKey: id},
		),
		moduleDoesNotAllowNewConfiguration: agent_metrics.CreateErrorCounter(
			managerErrorMetricName,
			map[string]string{managerErrorMetricKey: "module_does_not_allow_new_configuration", managerIDMetricKey: id},
		),
		invalidNewConfigurationType: agent_metrics.CreateErrorCounter(
			managerErrorMetricName,
			map[string]string{managerErrorMetricKey: "invalid_new_configuration_type", managerIDMetricKey: id},
		),
	}
}

type DynamicConfigurationManager[Configuration any] struct {
	id        string
	cfg       Configuration
	initiated bool

	registrationLock sync.Mutex
	registered       map[string][]registeredConfigurable

	metrics *DynamicConfigurationManagerMetrics
}

func NewDynamicConfigurationManager[Configuration any](id string) *DynamicConfigurationManager[Configuration] {
	return &DynamicConfigurationManager[Configuration]{
		registered: make(map[string][]registeredConfigurable),
		id:         id,
		metrics:    NewDynamicConfigurationManagerMetrics(id),
	}
}

// Update the configuration manager that the configuration has been updated.
func (mgr *DynamicConfigurationManager[Configuration]) OnConfigurationUpdate(
	newConfiguration Configuration,
) (finalError error) {
	mgr.registrationLock.Lock()
	defer mgr.registrationLock.Unlock()

	if !mgr.initiated {
		if err := mgr.validateConfigurationType(newConfiguration); err != nil {
			mgr.metrics.invalidNewConfigurationType.Inc()
			return err
		}
		if err := mgr.validateConfigurationDoesNotHaveDuplicateTypes(newConfiguration); err != nil {
			return err
		}
	}

	configurationsToRestore := make([]any, 0)
	modulesToRestore := make([]registeredConfigurable, 0)
	defer func() {
		if finalError == nil {
			mgr.initiated = true
			return
		}

		for i := 0; i < len(modulesToRestore); i++ {
			if err := modulesToRestore[i].call(configurationsToRestore[i]); err != nil {
				mgr.metrics.failedToRestore.Inc()
			}
		}
	}()

	for pathStr, registeredConfigurables := range mgr.registered {
		path := stringToPath(pathStr)

		newPathConfiguration, err := mgr.getStructByPath(newConfiguration, path)
		if err != nil {
			mgr.metrics.newPathConfigurationDoesNotExist.Inc()
			return fmt.Errorf("failed to find new configuration of path %s: %w", path, err)
		}

		oldPathConfiguration, err := mgr.getStructByPath(mgr.cfg, path)
		if err != nil {
			mgr.metrics.oldPathConfigurationDoesNotExist.Inc()
			return fmt.Errorf("failed to find old configuration of path %s: %w", path, err)
		}

		// Only trigger callbacks if the relevant configuration has changed
		if reflect.DeepEqual(oldPathConfiguration, newPathConfiguration) {
			continue
		}

		for _, configurable := range registeredConfigurables {
			if err := configurable.call(newPathConfiguration); err != nil {
				mgr.metrics.moduleDoesNotAllowNewConfiguration.Inc()
				return fmt.Errorf("registered module doesn't allow new configuration for path %s: %w", path, err)
			}

			configurationsToRestore = append(configurationsToRestore, oldPathConfiguration)
			modulesToRestore = append(modulesToRestore, configurable)
		}
	}

	mgr.cfg = newConfiguration

	return nil
}

// Register a callback to be called upon dynamic configuration change.
//
// The first argument is an instance of the configuration struct that the callback will receive. It shouldn't be a
// pointer to the instance, but the instance itself.
//
// The second argument is the callback. It must be a function that receives a single argument, which is of the correct
// type of the configuration, and returns a single return value, an error.
// The callback should return an error if the given configuration is invalid. It is possible due to the source of the
// configuration. For example, a user may provide an invalid string configuration.
// If one of the registered callbacks returns an error for a configuration update, the update is deemed invalid, and
// configuration restoration will occur: all of the callbacks that already finished successfully will be called again,
// with the previous configuration.
//
// Upon successful registration, the callback is instantly called, from the calling thread and before this function
// returns, with the most up-to-date configuration available.
func (mgr *DynamicConfigurationManager[Configuration]) Register(cfg any, callback any) error {
	expectedType := reflect.TypeOf(cfg)

	path, err := mgr.findPathToType(cfg)
	if err != nil {
		return fmt.Errorf("failed to find path: %w", err)
	}
	pathString := pathToString(path)

	mgr.registrationLock.Lock()
	defer mgr.registrationLock.Unlock()

	// Get the most up-to-date configuration after acquiring the lock, so that if further changes follow, the registerer
	// will always get the updates.
	pathConfiguration, err := mgr.getStructByPath(mgr.cfg, path)
	if err != nil {
		return fmt.Errorf("failed to perform initial query of path %s: %w", path, err)
	}

	if _, exists := mgr.registered[pathString]; !exists {
		mgr.registered[pathString] = make([]registeredConfigurable, 0)
	}

	callbackType := reflect.TypeOf(callback)
	if callbackType.Kind() != reflect.Func {
		return fmt.Errorf("%w: can't register non-function", ErrBadCallback)
	}

	if callbackType.NumIn() != 1 {
		return fmt.Errorf(
			"%w: can't register type whose callback does not receive exactly one argument",
			ErrBadCallback,
		)
	}

	if callbackType.In(0) != expectedType {
		return fmt.Errorf("%w: can't register type whose callback argument is the wrong type", ErrBadCallback)
	}

	if callbackType.NumOut() != 1 {
		return fmt.Errorf("%w: can't register type whose callback does not return exactly one argument", ErrBadCallback)
	}

	if callbackType.Out(0) != reflect.TypeOf((*error)(nil)).Elem() {
		return fmt.Errorf("%w: can't register type whose callback does not return an error", ErrBadCallback)
	}

	callbackMethod := reflect.ValueOf(callback)

	registeredConfigurable := registeredConfigurable{
		configurable: callback,
		expectedType: expectedType,
		callback:     callbackMethod,
	}

	mgr.registered[pathString] = append(mgr.registered[pathString], registeredConfigurable)

	return registeredConfigurable.call(pathConfiguration)
}

// Finds the path to a struct inside the configuration that matches the type of the target.
// It returns a slice of field names needed to reach the struct, or an error if no match is found.
func (mgr *DynamicConfigurationManager[Configuration]) findPathToType(target any) ([]string, error) {
	srcVal := reflect.ValueOf(mgr.cfg)
	targetType := reflect.TypeOf(target)

	if targetType.Kind() == reflect.Ptr {
		targetType = targetType.Elem()
	}

	var foundPath []string
	callback := func(v reflect.Value, path []string) bool {
		if v.Type() != targetType {
			return false
		}
		foundPath = make([]string, len(path))
		copy(foundPath, path)
		return true
	}

	done, err := mgr.iterateStructsInConfiguration(srcVal, []string{}, callback)
	if err != nil {
		return nil, err
	}
	if !done {
		return nil, ErrNoMatchingTypeFound
	}
	return foundPath, nil
}

func (mgr *DynamicConfigurationManager[Configuration]) iterateStructsInConfiguration(
	v reflect.Value,
	path []string,
	callback func(v reflect.Value, path []string) bool,
) (bool, error) {
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return false, errInvalidTypeToIterate
		}
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return false, errInvalidTypeToIterate
	}

	if callback(v, path) {
		return true, nil
	}

	for i := 0; i < v.NumField(); i++ {
		field := v.Type().Field(i)
		fieldVal := v.Field(i)

		innerPath := make([]string, len(path))
		copy(innerPath, path)
		innerPath = append(innerPath, field.Name)

		isDone, err := mgr.iterateStructsInConfiguration(fieldVal, innerPath, callback)
		if err != nil {
			if errors.Is(err, errInvalidTypeToIterate) {
				continue
			}
			return false, fmt.Errorf("error iterating path %s: %w", pathToString(innerPath), err)
		}

		if isDone {
			return isDone, nil
		}
	}

	return false, nil
}

// Traverses the configuration using the given path of field names and returns the found struct.
func (mgr *DynamicConfigurationManager[Configuration]) getStructByPath(cfg any, path []string) (any, error) {
	srcVal := reflect.ValueOf(cfg)

	for _, field := range path {
		if srcVal.Kind() == reflect.Ptr {
			if srcVal.IsNil() {
				return nil, fmt.Errorf("field %s of path %s is nil", field, pathToString(path))
			}
			srcVal = srcVal.Elem()
		}

		srcVal = srcVal.FieldByName(field)

		if srcVal.Kind() == reflect.Ptr {
			srcVal = srcVal.Elem()
			if !srcVal.IsValid() {
				return nil, fmt.Errorf("nil pointer encountered at field %s of path %s", field, pathToString(path))
			}
		}
	}

	return srcVal.Interface(), nil
}

func (mgr *DynamicConfigurationManager[Configuration]) validateConfigurationType(newConfiguration Configuration) error {
	newVal := reflect.ValueOf(newConfiguration)

	if newVal.Kind() != reflect.Struct {
		return fmt.Errorf("%w: configuration must be a struct", ErrWrongConfigurationType)
	}

	return nil
}

func (mgr *DynamicConfigurationManager[Configuration]) validateConfigurationDoesNotHaveDuplicateTypes(
	cfg Configuration,
) error {
	srcVal := reflect.ValueOf(cfg)

	foundTypes := make(map[string]string)
	foundDuplicatePath := ""

	callback := func(v reflect.Value, path []string) bool {
		pathStr := pathToString(path)
		t := v.Type()
		typeStr := t.String()
		if t.PkgPath() != "" {
			typeStr = t.PkgPath() + "." + typeStr
		}
		if _, exists := foundTypes[typeStr]; exists {
			foundDuplicatePath = pathStr
			return true
		}
		foundTypes[typeStr] = pathStr
		return false
	}

	done, err := mgr.iterateStructsInConfiguration(srcVal, []string{}, callback)
	if err != nil {
		return fmt.Errorf("failed to iterate configuration: %w", err)
	}
	if done {
		return fmt.Errorf("%w: %s", ErrConfigurationHasDuplicates, foundDuplicatePath)
	}
	return nil
}

func pathToString(path []string) string {
	return strings.Join(path, pathSeparator)
}

func stringToPath(str string) []string {
	return strings.Split(str, pathSeparator)
}
