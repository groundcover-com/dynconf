# dynconf

With this package you can configure your Go modules dynamically.

The [Dynamic Configuration Listener](pkg/listener) listens on updates to a configuration file, merges them onto a default configuration, and notifies that the configuration has been updated.

The [Dynamic Configuration Manager](pkg/manager) allows modules to register to a specific part of the configuration, and distributes the relevant parts of the updated configuration to the registered modules.
