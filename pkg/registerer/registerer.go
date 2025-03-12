package registerer

type DynamicConfigurationRegisterable interface {
	Register(path []string, callback any) error
}

type DynamicConfigurationRegisterer struct {
	registerable DynamicConfigurationRegisterable
	prefix       []string
}

func NewDynamicConfigurationRegisterer(registerable DynamicConfigurationRegisterable) *DynamicConfigurationRegisterer {
	return &DynamicConfigurationRegisterer{
		registerable: registerable,
		prefix:       make([]string, 0),
	}
}

func (registerer *DynamicConfigurationRegisterer) Register(callback any) error {
	return registerer.registerable.Register(registerer.prefix, callback)
}

func (registerer *DynamicConfigurationRegisterer) Under(under string) *DynamicConfigurationRegisterer {
	p := make([]string, 0, len(registerer.prefix)+1)
	for _, v := range registerer.prefix {
		p = append(p, v)
	}
	p = append(p, under)

	return &DynamicConfigurationRegisterer{
		registerable: registerer.registerable,
		prefix:       p,
	}
}
