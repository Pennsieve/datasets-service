package store

import (
	"context"
	"database/sql"
	"fmt"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/rds/auth"
	log "github.com/sirupsen/logrus"
	"os"
)

func init() {
	log.SetFormatter(&log.JSONFormatter{})
	ll, err := log.ParseLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		log.SetLevel(log.InfoLevel)
	} else {
		log.SetLevel(ll)
	}
}

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

func PostgresConfigForRDS() (*PostgresConfig, error) {

	cfg, err := awsConfig.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, err
	}

	region := os.Getenv("REGION")
	dbHost := os.Getenv("RDS_PROXY_ENDPOINT")
	dbPort := "5432"
	dbEndpoint := fmt.Sprintf("%s:%s", dbHost, dbPort)
	dbUser := fmt.Sprintf("%s_rds_proxy_user", os.Getenv("ENV"))

	authenticationToken, err := auth.BuildAuthToken(
		context.TODO(), dbEndpoint, region, dbUser, cfg.Credentials)
	if err != nil {
		return nil, err
	}

	config := PostgresConfig{
		Host:     dbHost,
		Port:     dbPort,
		User:     dbUser,
		Password: authenticationToken,
		DBName:   "pennsieve_postgres",
	}

	return &config, nil
}
