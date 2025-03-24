package listener

type DynamicConfigurable[Configuration any] interface {
	OnConfigurationUpdate(newConfiguration Configuration) error
}
