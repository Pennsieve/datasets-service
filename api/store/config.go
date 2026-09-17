package store

import (
	"database/sql"
	"fmt"
	"os"
)

// Logging configuration used to live here in an init() that duplicated the one
// in the handler package (and a third in api/logging). Go's init() ordering
// made "whichever ran last" decide the effective level. The Lambda entrypoint
// now calls logging.SetDefaultFromEnv once instead.

type PostgresConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

func (c *PostgresConfig) String() string {
	port := c.Port
	if port == "" {
		port = "5432"
	}
	noSSLConfig := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s",
		c.Host, port, c.User, c.Password, c.DBName)
	if c.SSLMode == "" {
		return noSSLConfig
	}
	return fmt.Sprintf("%s sslmode=%s", noSSLConfig, c.SSLMode)
}

func (c *PostgresConfig) LogString() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=**** dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.DBName, c.SSLMode)
}

func (c *PostgresConfig) Open() (*sql.DB, error) {
	return sql.Open("postgres", c.String())
}

func (c *PostgresConfig) OpenAtSchema(schema string) (*sql.DB, error) {
	// Setting search_path in the connection string is a lib/pq driver extension.
	// Might not be available with other drivers.
	connStr := fmt.Sprintf("%s search_path=%s", c, schema)
	return sql.Open("postgres", connStr)
}

func PostgresConfigFromEnv() *PostgresConfig {
	return &PostgresConfig{
		Host:     os.Getenv("POSTGRES_HOST"),
		Port:     os.Getenv("POSTGRES_PORT"),
		User:     os.Getenv("POSTGRES_USER"),
		Password: os.Getenv("POSTGRES_PASSWORD"),
		DBName:   os.Getenv("PENNSIEVE_DB"),
		SSLMode:  os.Getenv("POSTGRES_SSL_MODE"),
	}
}
