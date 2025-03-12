# Dynamic Configuration Manager

With this package you can distribute your dynamic configuration to different modules.

## Initiation

First define your configuration struct, which should be a nested struct, such as:

```go
type ModuleA struct {
	Value string
}
type ModuleB struct {
	Value bool
}
type ConfigurationExample struct {
	A ModuleA
	B ModuleB
}
```

Then, initiate a dynamic configuration manager:

```go
DynamicConfigurationManager = manager.NewDynamicConfigurationManager[ConfigurationExample]("example")
```

Now, the manager is ready to be used, but it still hasn't been given its first configuration.

## Configuration Update

To initiate the configuration, as well as whenever the configuration changes, use `OnConfigurationUpdate` to notify the manager.
The manager finds the relevant parts of the configuration which have changed since the last call, and notifies all registered users.
If the manager fails to perform the update, or if one of the registered users don't agree to receive the new configuration, an error is returned.

```go
cnf := ConfigurationExample{}
err := DynamicConfigurationManager.OnConfigurationUpdate(cnf)
```

## User Registration

To register a user, provide a callback function. This function will be called by the manager whenever the relevant part of the configuration changes.
In addition, this function is called immediately, with the most up-to-date configuration at the time of registration (the `Register` call internally calls the callback).

If the provided configuration is illegal, the callback may return an error. This indicates to the manager that the configuration should not be accepted.
Registered users who have not yet been given the new configuration will not be given it. Those who have already accepted it will have their callback called again, with the configuration before the change. This is referred to as "restoration".

```go
callback := func(cfg ModuleA) error {
	return nil
}
err := DynamicConfigurationManager.Register("A", callback)
```
