package configurable

import (
	"github.com/kelseyhightower/envconfig"
)

// Configurable allows InitConfig to properly configure the config struct using
// environment variables and default values.
type Configurable interface {
	Init()
}

// Initializes the configuration
func InitConfig(config Configurable) {
	envconfig.MustProcess("config", config)
	config.Init()
}
