# Dynamic Configuration Getter

With this package you can limit different modules in your code to get and register on specific parts of your dynamic configuration.

This should be used in conjunction with the [dynamic configuration manager](/pkg/manager), in order to avoid passing full paths within the configuration struct by each module.

When a dynamic configurable module is initiated, it can be passed a `Dynamic Configuration Registerer`, hereby only allowing it access to fields within the registerer's scope.

## Initiation

After having set up your manager, initiate a getter:

```go
topLevelGetter := registerer.NewDynamicConfigurationGetter(mgr)
```

Now you can traverse the configuration tree, one field at a time:

```go
nextLevelGetter := topLevelGetter.Select("fieldName")
```

When you've reached the destination field, you can register a callback to be triggered whenever this field changes:

```go
callback := func(cfg ModuleConfiguration) error {
	return nil
}
err := nextLevelGetter.Register(callback)
```

Like so, the module above only needs access to `nextLevelGetter`.
If the path to its configuration alters, it doesn't need to be aware: only the module that initiates it needs be.

## Testing

When testing a dynamically-configurable module that uses a getter, mocks can be used.
The following example sets a getter that when passed to a module that uses it, handles `Get` or `Registers` as requested inline.
An example module that uses `Register` can have its configuration reload logic tested by triggering the registered callback directly.

```go
getter.NewDynamicConfigurationGetter(
	getter.NewMockDynamicConfigurationGettableWithType(
		func(path []string, out *ConfigurationType) error {
			*out = myConfiguration
			return nil
		},
		func(path []string, callback func(a ConfigurationType) error) error {
			myCallback = callback // store the callback to push new configuration later if needed
			return callback(myConfiguration)
		},
	),
)
```
