package getter

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
