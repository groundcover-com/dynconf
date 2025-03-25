package getter

import "slices"

type DynamicConfigurationGetter struct {
	gettable DynamicConfigurationGettable
	prefix   []string
}

func NewDynamicConfigurationGetter(gettable DynamicConfigurationGettable) *DynamicConfigurationGetter {
	return NewDynamicConfigurationGetterWithPrefix(gettable, make([]string, 0))
}

func NewDynamicConfigurationGetterWithPrefix(
	gettable DynamicConfigurationGettable,
	prefix []string,
) *DynamicConfigurationGetter {
	return &DynamicConfigurationGetter{
		gettable: gettable,
		prefix:   slices.Clone(prefix),
	}
}

func (getter *DynamicConfigurationGetter) Register(callback any) error {
	return getter.gettable.Register(getter.prefix, callback)
}

func (getter *DynamicConfigurationGetter) Get(out any) error {
	return getter.gettable.Get(getter.prefix, out)
}

func (getter *DynamicConfigurationGetter) Select(selection string) *DynamicConfigurationGetter {
	return &DynamicConfigurationGetter{
		gettable: getter.gettable,
		prefix:   append(slices.Clone(getter.prefix), selection),
	}
}

type MockDynamicConfigurationGetter[T any] struct {
	gettable DynamicConfigurationGettable
	prefix   []string
}

func (getter *MockDynamicConfigurationGetter[T]) Register(callback func(T) error) error {
	return getter.gettable.Register([]string{}, callback)
}

func (getter *MockDynamicConfigurationGetter[T]) Get(out *T) error {
	return getter.gettable.Get([]string{}, out)
}
