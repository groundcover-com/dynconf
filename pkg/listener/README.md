# Dynamic Configuration Updater

The `Dynamic Configuration Updater` listens on updates to a configuration file, merges them onto a default configuration, and notifies that the configuration has been updated.

A simple example, using a [Dynamic Configuration Manager](/pkg/manager) as the object to be notified on configuration update:

```go
type Config struct { /* configuration here */ }

//go:embed default_config.yaml
var defaultConfig string

listener, err := NewDynamicConfigurationListener[Config](
    vpr,
    defaultConfig,
    "config.yaml",
    DynamicConfigurationManager,
    nil,
)
```
