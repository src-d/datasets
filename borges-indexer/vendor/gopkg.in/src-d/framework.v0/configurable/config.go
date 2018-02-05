package configurable

var (
	// Initialized BasicConfiguration
	Config *BasicConfiguration = &BasicConfiguration{}
)

// BasicConfiguration is the default configuration
type BasicConfiguration struct {
}

// Init initializes BasicConfiguration
func (c *BasicConfiguration) Init() {
}

func init() {
	InitConfig(Config)
}
