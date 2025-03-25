package getter

import "fmt"

type DynamicConfigurationGettable interface {
	Register(path []string, callback any) error
	Get(path []string, out any) error
}

type MockDynamicConfigurationGettable struct {
	register func(path []string, callback any) error
	get      func(path []string, out any) error
}

func NewMockDynamicConfigurationGettable(
	register func(path []string, callback any) error,
	get func(path []string, out any) error,
) *MockDynamicConfigurationGettable {
	return &MockDynamicConfigurationGettable{
		register: register,
		get:      get,
	}
}

func (gettable *MockDynamicConfigurationGettable) Register(path []string, callback any) error {
	return gettable.register(path, callback)
}

func (gettable *MockDynamicConfigurationGettable) Get(path []string, out any) error {
	return gettable.get(path, out)
}

type MockDynamicConfigurationGettableWithType[T any] struct {
	get      func(path []string, out *T) error
	register func(path []string, callback func(T) error) error
}

func NewMockDynamicConfigurationGettableWithType[T any](
	get func(path []string, out *T) error,
	register func(path []string, callback func(T) error) error,
) *MockDynamicConfigurationGettableWithType[T] {
	return &MockDynamicConfigurationGettableWithType[T]{
		get:      get,
		register: register,
	}
}

func (gettable *MockDynamicConfigurationGettableWithType[T]) Register(path []string, callback any) error {
	callbackFunc, ok := callback.(func(T) error)
	if !ok {
		return fmt.Errorf("invalid callback type: expected 'func(T) error', got %T", callback)
	}

	return gettable.register(path, callbackFunc)
}

func (gettable *MockDynamicConfigurationGettableWithType[T]) Get(path []string, out any) error {
	typedOut, ok := out.(*T)
	if !ok {
		return fmt.Errorf("invalid out type: expected '*T', got %T", out)
	}

	return gettable.get(path, typedOut)
}
