# Dynamic Configuration Updater

The `Dynamic Configuration Updater` listens on updates to a configuration file, merges them onto a default configuration, and notifies that the configuration has been updated.

This works using [viper](github.com/spf13/viper)'s abilities to listen to file updates and to merge configurations.
Using viper also allows further abilities. For example, if the `viper` object used is configured to have environment variables overrides, they will also override any dynamic configuration.

A simple example, using a [Dynamic Configuration Manager](/pkg/manager) as the object to be notified on configuration update:

```go
type Config struct { /* configuration here */ }

//go:embed default_config.yaml
var defaultConfig string

vpr := viper.New()
vpr.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
vpr.SetEnvPrefix("ENV")
vpr.AutomaticEnv()
vpr.SetConfigType("yaml")

// This will be called on errors, for example if illegal configuration is written to the file.
onUpdateErrorCallback := func(error){}

listener, err := NewDynamicConfigurationListener[Config](
    vpr,
    defaultConfig,
    "config.yaml",
    DynamicConfigurationManager,
    onUpdateErrorCallback,
)

// assuming no error occurred, from this moment, DynamicConfigurationManager receives updates when "config.yaml" is edited.
```
