package manager

import (
	"fmt"
	"reflect"
)

type registeredConfigurable struct {
	configurable any
	expectedType reflect.Type
	callback     reflect.Value
}

func (configurable *registeredConfigurable) call(configuration any) error {
	castedCfg := reflect.New(configurable.expectedType).Interface()

	castedCfgValue := reflect.ValueOf(castedCfg).Elem()
	newCfgValue := reflect.ValueOf(configuration)
	newCfgValueType := newCfgValue.Type()

	if !newCfgValueType.AssignableTo(configurable.expectedType) {
		return fmt.Errorf(
			"configuration value of type %s isn't assignable to expected type %s",
			newCfgValueType.String(),
			configurable.expectedType.String(),
		)
	}

	castedCfgValue.Set(newCfgValue)
	returnValue := configurable.callback.Call([]reflect.Value{castedCfgValue})[0]
	if returnValue.IsNil() {
		return nil
	}

	err := returnValue.Interface().(error)
	return err
}
