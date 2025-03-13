package registerer

import "slices"

type DynamicConfigurationRegisterable interface {
	Register(path []string, callback any) error
}

type DynamicConfigurationRegisterer struct {
	registerable DynamicConfigurationRegisterable
	prefix       []string
}

func NewDynamicConfigurationRegisterer(registerable DynamicConfigurationRegisterable) *DynamicConfigurationRegisterer {
	return NewDynamicConfigurationRegistererWithPrefix(registerable, make([]string, 0))
}

func NewDynamicConfigurationRegistererWithPrefix(
	registerable DynamicConfigurationRegisterable,
	prefix []string,
) *DynamicConfigurationRegisterer {
	return &DynamicConfigurationRegisterer{
		registerable: registerable,
		prefix:       slices.Clone(prefix),
	}
}

func (registerer *DynamicConfigurationRegisterer) Register(callback any) error {
	return registerer.registerable.Register(registerer.prefix, callback)
}

func (registerer *DynamicConfigurationRegisterer) Under(under string) *DynamicConfigurationRegisterer {
	return &DynamicConfigurationRegisterer{
		registerable: registerer.registerable,
		prefix:       append(slices.Clone(registerer.prefix), under),
	}
}
