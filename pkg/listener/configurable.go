package listener

type DynamicConfigurable[Configuration any] interface {
	OnConfigurationUpdate(newConfiguration Configuration) error
}

type DynamicConfigurableWithCallback[Configuration any] struct {
	callback func(Configuration) error
}

func NewDynamicConfigurableWithCallback[Configuration any](
	callback func(cfg Configuration) error,
) *DynamicConfigurableWithCallback[Configuration] {
	return &DynamicConfigurableWithCallback[Configuration]{callback: callback}
}

func (configurable *DynamicConfigurableWithCallback[Configuration]) OnConfigurationUpdate(cfg Configuration) error {
	return configurable.callback(cfg)
}
