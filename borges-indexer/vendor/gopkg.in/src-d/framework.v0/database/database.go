package database

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"gopkg.in/src-d/framework.v0/configurable"
)

// DatabaseConfig describes the configuration of the database parameters used
// to establish a connection with it.
type DatabaseConfig struct {
	configurable.BasicConfiguration

	// Username of the database user.
	Username string `envconfig:"dbuser" default:"testing"`
	// Password of the database user.
	Password string `envconfig:"dbpass" default:"testing"`
	// Port used to establish connection with the database.
	Port int `envconfig:"dbport" default:"5432"`
	// Host to establish connection with the database.
	Host string `envconfig:"dbhost" default:"0.0.0.0"`
	// Name of the database to connect to.
	Name string `envconfig:"dbname" default:"testing"`
	// SSLMode used to specify the way of using SSL in the connection.
	SSLMode SSLMode `envconfig:"dbsslmode" default:"disable"`
	// AppName is the name of the app using the connection.
	AppName string `envconfig:"dbappname"`
	// Timeout is the number time to consider a connection timed out.
	Timeout time.Duration `envconfig:"dbtimeout" default:"30s"`
}

// DataSourceName returns the DSN string to connect to the database.
func (c *DatabaseConfig) DataSourceName() (string, error) {
	if c.Name == "" {
		return "", fmt.Errorf("database: database name cannot be empty")
	}

	if c.Port <= 0 {
		return "", fmt.Errorf("database: port is not valid")
	}

	if c.Host == "" {
		return "", fmt.Errorf("database: host cannot be empty")
	}

	if c.Username == "" {
		return "", fmt.Errorf("database: username cannot be empty")
	}

	if string(c.SSLMode) == "" {
		c.SSLMode = Disable
	}

	ds := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.Username,
		c.Password,
		c.Host,
		c.Port,
		c.Name,
		c.SSLMode,
	)

	if c.AppName != "" {
		ds += fmt.Sprintf("&application_name=%s", c.AppName)
	}

	if c.Timeout >= 1*time.Second {
		ds += fmt.Sprintf("&connect_timeout=%d", int(c.Timeout.Seconds()))
	}

	return ds, nil
}

// SSLMode provides different levels of protection against attacks.
// Check https://www.postgresql.org/docs/9.1/static/libpq-ssl.html
type SSLMode string

const (
	// Disable disables the SSL.
	Disable SSLMode = "disable"
	// Allow enables the SSL only if the server insists on it.
	Allow SSLMode = "allow"
	// Prefer enables the SSL but only if the server supports it.
	Prefer SSLMode = "prefer"
	// Require ensures SSL is used but does not check the network is secure.
	Require SSLMode = "require"
	// VerifyCA ensures SSL is used and the connection is established with a
	// trusted server.
	VerifyCA SSLMode = "verify-ca"
	// VerifyFull ensures SSL is used and the connection is established with
	// the specified server.
	VerifyFull SSLMode = "verify-full"
)

// DefaultConfig is the default database configuration, whose values come from
// environment variables with default values if the environment variables are
// not provided to the value used in a testing setup.
// - CONFIG_DBUSER: database username
// - CONFIG_DBPASS: database user password
// - CONFIG_DBHOST: database host
// - CONFIG_DBPORT: database port
// - CONFIG_DBNAME: database name
// - CONFIG_DBSSLMODE: ssl mode to use
// - CONFIG_DBAPPNAME: application name
// - CONFIG_DBTIMEOUT: connection timeout
var DefaultConfig = new(DatabaseConfig)

// ErrNoConfig is returned when there is an attempt of getting a databsse
// connection with no configuration.
var ErrNoConfig = errors.New("database: can't get database with no configuration")

// Get returns a database connection with the configuration resultant of
// applying the given configfuncs to the config.
// Passing a nil configuration will result in an error.
func Get(config *DatabaseConfig, configurators ...ConfigFunc) (*sql.DB, error) {
	if config == nil {
		return nil, ErrNoConfig
	}

	for _, c := range configurators {
		config = c(config)
	}

	ds, err := config.DataSourceName()
	if err != nil {
		return nil, err
	}

	return sql.Open("postgres", ds)
}

// Default returns a database connection established using the default
// configuration.
func Default(configurators ...ConfigFunc) (*sql.DB, error) {
	return Get(DefaultConfig, configurators...)
}

// Must will panic if the given error is not nil and, otherwise, will return
// the database connection.
func Must(db *sql.DB, err error) *sql.DB {
	if err != nil {
		panic(err)
	}
	return db
}

// ConfigFunc is a function that will receive a database configuration and
// return a new one with some parameters changed.
type ConfigFunc func(*DatabaseConfig) *DatabaseConfig

// WithName returns a ConfigFunc that will change the database name of the
// received database config to the one given.
func WithName(name string) ConfigFunc {
	return func(c *DatabaseConfig) *DatabaseConfig {
		cfg := *c
		cfg.Name = name
		return &cfg
	}
}

func init() {
	configurable.InitConfig(DefaultConfig)
}
