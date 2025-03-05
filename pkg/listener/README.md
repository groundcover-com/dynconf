# Dynamic Configuration Updater

The `Dynamic Configuration Updater` listens on updates to a configuration file, merges them onto a default configuration, and notifies that the configuration has been updated.

This works using [viper](github.com/spf13/viper)'s abilities to listen to file updates and to merge configurations.
Using viper also allows further abilities. For example, if the `viper` object used is configured to have environment variables overrides, they will also override any dynamic configuration.

Below is a simple example, using a [Dynamic Configuration Manager](/pkg/manager) as the object to be notified on configuration update.
Assuming no error occurred, DynamicConfigurationManager receives an the initial configuration when the initiation occurs, and from that moment, further updates whenever the configuration file is updated.

```go
type Config struct { /* configuration struct here */ }

//go:embed default_config.yaml
var defaultConfig string

listener, err := NewDynamicConfigurationListener[Config](
    "id",
    "config.yaml",
    DynamicConfigurationManager,
    Options{
        Viper: ViperOptions{
            EnvKeyReplacer: strings.NewReplacer(".", "_"),
            EnvPrefix: "ENV",
            AutomaticEnv: true,
            ConfigType: "yaml",
        },
        DefaultConfiguration: DefaultConfigurationOptions{
            Type: DefaultConfigurationTypeString,
            String: defaultConfig,
        }
    },
)
```
