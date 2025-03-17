package manager

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"

	metrics_factory "github.com/groundcover-com/metrics/pkg/factory"
	metrics_types "github.com/groundcover-com/metrics/pkg/types"
)

const (
	pathSeparator = "."

	managerMetricPrefix = "dynconf_manager_"
	errorMetricName     = managerMetricPrefix + "error"
	errorMetricKey      = "error"
	idMetricKey         = "id"
)

var (
	// Registering to listen on updates to a path that doesn't exist within the configuration will return this error.
	ErrNoMatchingFieldFound = errors.New("no matching field found")

	// The configuration manager allows only specific types of configurations to be used. This error indicates that a
	// wrong configuration type is used, and it can only be returned on the first configuration update.
	ErrWrongConfigurationType = errors.New("wrong configuration type")

	// When registering, the callback is very unspecific. However, in fact it has to have a very specific definition.
	// If a wrong callback is given, this error is returned.
	ErrBadCallback = errors.New("bad callback")

	// When getting configuration, the out parameter is very unspecific. However, in fact it has to have a specific
	// type. If a wrong type is given, this error is returned.
	ErrBadType = errors.New("bad type")

	// When registering, a valid path must be provided.
	ErrInvalidPath = errors.New("invalid path")
)

type DynamicConfigurationManagerMetrics struct {
	failedToRestore                    *metrics_types.LazyCounter
	newPathConfigurationDoesNotExist   *metrics_types.LazyCounter
	oldPathConfigurationDoesNotExist   *metrics_types.LazyCounter
	moduleDoesNotAllowNewConfiguration *metrics_types.LazyCounter
}

func NewDynamicConfigurationManagerMetrics(id string) *DynamicConfigurationManagerMetrics {
	return &DynamicConfigurationManagerMetrics{
		failedToRestore: metrics_factory.CreateErrorCounter(
			errorMetricName,
			map[string]string{errorMetricKey: "failed_to_restore", idMetricKey: id},
		),
		newPathConfigurationDoesNotExist: metrics_factory.CreateErrorCounter(
			errorMetricName,
			map[string]string{errorMetricKey: "new_path_configuration_does_not_exist", idMetricKey: id},
		),
		oldPathConfigurationDoesNotExist: metrics_factory.CreateErrorCounter(
			errorMetricName,
			map[string]string{errorMetricKey: "old_path_configuration_does_not_exist", idMetricKey: id},
		),
		moduleDoesNotAllowNewConfiguration: metrics_factory.CreateErrorCounter(
			errorMetricName,
			map[string]string{errorMetricKey: "module_does_not_allow_new_configuration", idMetricKey: id},
		),
	}
}

type DynamicConfigurationManager[Configuration any] struct {
	id  string
	cfg Configuration

	configUpdateLock sync.Mutex
	registered       map[string][]registeredConfigurable

	metrics *DynamicConfigurationManagerMetrics
}

func NewDynamicConfigurationManager[Configuration any](id string) (*DynamicConfigurationManager[Configuration], error) {
	if err := validateConfigurationType[Configuration](); err != nil {
		return nil, err
	}

	return &DynamicConfigurationManager[Configuration]{
		registered: make(map[string][]registeredConfigurable),
		id:         id,
		metrics:    NewDynamicConfigurationManagerMetrics(id),
	}, nil
}

// Pass updated configuration to the configuration manager.
// Before calling that, the configuration is the zero configuration, so it's good practice to call this for the first
// time right after initiating the manager.
func (mgr *DynamicConfigurationManager[Configuration]) OnConfigurationUpdate(
	newConfiguration Configuration,
) (finalError error) {
	mgr.configUpdateLock.Lock()
	defer mgr.configUpdateLock.Unlock()

	configurationsToRestore := make([]any, 0)
	modulesToRestore := make([]registeredConfigurable, 0)
	defer func() {
		if finalError == nil {
			return
		}

		for i := range modulesToRestore {
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

// Get the current value of a part of the configuration.
// To get the current value and also be notified on updates, register instead.
//
// The second argument is an out parameter, where the current configuration will be set.
// The configuration under this path is a struct, and this has to be a pointer a struct of the same type.
func (mgr *DynamicConfigurationManager[Configuration]) Get(path []string, out any) error {
	if out == nil {
		return fmt.Errorf("%w: out parameter can not be nil", ErrBadType)
	}

	outValue := reflect.ValueOf(out)
	if outValue.Kind() != reflect.Pointer {
		return fmt.Errorf("%w: out parameter must be a pointer", ErrBadType)
	}

	outValue = outValue.Elem()
	if !outValue.CanSet() { // shouldn't return false, because the out parameter is obviously addressable.
		return fmt.Errorf("%w: out parameter can not be set", ErrBadType)
	}

	if err := validatePath(path); err != nil {
		return err
	}
	pathString := pathToString(path)

	mgr.configUpdateLock.Lock()
	defer mgr.configUpdateLock.Unlock()

	pathConfiguration, err := mgr.getStructByPath(mgr.cfg, path)
	if err != nil {
		return fmt.Errorf("failed to perform query of path %s: %w", pathString, err)
	}

	if !reflect.TypeOf(pathConfiguration).AssignableTo(outValue.Type()) {
		return fmt.Errorf("%w: can't get configuration into out parameter of the wrong type", ErrBadType)
	}

	outValue.Set(reflect.ValueOf(pathConfiguration))
	return nil
}

// Register a callback to be called upon dynamic configuration change.
//
// The first argument is the path to the struct requested within the configuration struct.
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
// If no configuration was passed to the manager yet, the most up-to-date configuration is the zero configuration.
func (mgr *DynamicConfigurationManager[Configuration]) Register(path []string, callback any) error {
	if err := validatePath(path); err != nil {
		return err
	}
	pathString := pathToString(path)

	mgr.configUpdateLock.Lock()
	defer mgr.configUpdateLock.Unlock()

	// Get the most up-to-date configuration after acquiring the lock, so that if further changes follow, the registerer
	// will always get the updates.
	pathConfiguration, err := mgr.getStructByPath(mgr.cfg, path)
	if err != nil {
		return fmt.Errorf("failed to perform initial query of path %s: %w", pathString, err)
	}
	expectedType := reflect.TypeOf(pathConfiguration)

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

	if !reflect.TypeOf(pathConfiguration).AssignableTo(callbackType.In(0)) {
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

// Traverses the configuration using the given path of field names and returns the found struct.
func (mgr *DynamicConfigurationManager[Configuration]) getStructByPath(cfg any, path []string) (any, error) {
	srcVal := reflect.ValueOf(cfg)

	for _, field := range path {
		if srcVal.Kind() == reflect.Ptr {
			if srcVal.IsNil() {
				return nil, fmt.Errorf(
					"field %s of path %s is nil: %w",
					field,
					pathToString(path),
					ErrNoMatchingFieldFound,
				)
			}
			srcVal = srcVal.Elem()
		}

		if srcVal.Kind() != reflect.Struct {
			return nil, fmt.Errorf(
				"can't access field %s of non-struct type %s in path %s: %w",
				field,
				srcVal.Kind(),
				pathToString(path),
				ErrNoMatchingFieldFound,
			)
		}

		structType := srcVal.Type()
		_, found := structType.FieldByName(field)
		if !found {
			return nil, fmt.Errorf(
				"field %s does not exist in struct type %s with path %s: %w",
				field,
				structType,
				pathToString(path),
				ErrNoMatchingFieldFound,
			)
		}

		srcVal = srcVal.FieldByName(field)

		if srcVal.Kind() == reflect.Ptr {
			srcVal = srcVal.Elem()
			if !srcVal.IsValid() {
				return nil, fmt.Errorf(
					"nil pointer encountered at field %s of path %s: %w",
					field,
					pathToString(path),
					ErrNoMatchingFieldFound,
				)
			}
		}
	}

	return srcVal.Interface(), nil
}

func validateConfigurationType[Configuration any]() error {
	var zeroConfig Configuration
	newVal := reflect.ValueOf(zeroConfig)

	if newVal.Kind() != reflect.Struct {
		return fmt.Errorf("%w: configuration must be a struct", ErrWrongConfigurationType)
	}

	return nil
}

func validatePath(path []string) error {
	for _, p := range path {
		if strings.Contains(p, pathSeparator) {
			return ErrInvalidPath
		}
	}

	return nil
}

func pathToString(path []string) string {
	return strings.Join(path, pathSeparator)
}

func stringToPath(str string) []string {
	if str == "" {
		return make([]string, 0)
	}

	return strings.Split(str, pathSeparator)
}
