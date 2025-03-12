# Dynamic Configuration Registerer

With this package you can limit different modules in your code to register on specific parts of your dynamic configuration.

This should be used in conjunction with the [dynamic configuration manager](/pkg/manager), in order to avoid passing full paths within the configuration struct by each module.

When a dynamic configurable module is initiated, it can be passed a `Dynamic Configuration Registerer`, hereby only allowing it access to fields within the registerer's scope.

## Initiation

After having set up your manager, initiate a registerer:

```go
topLevelRegisterer := registerer.NewDynamicConfigurationRegisterer(mgr)
```

Now you can traverse the configuration tree, one field at a time:

```go
nextLevelRegisterer := topLevelRegisterer.Under("fieldName")
```

When you've reached the destination field, you can register a callback to be triggered whenever this field changes:

```go
callback := func(cfg ModuleConfiguration) error {
	return nil
}
err := nextLevelRegisterer.Register(callback)
```

Like so, the module above only needs access to `nextLevelRegisterer`.
If the path to its configuration alters, it doesn't need to be aware: only the module that initiates it needs be.
