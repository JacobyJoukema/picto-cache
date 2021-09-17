package main

import (
	"fmt"
	"os"

	"github.com/inflowml/logger"
	"github.com/inflowml/structql"
)

// Default database configuration for non-production builds
const (
	DB_NAME   = "metadb"
	DB_USER   = "dbadmin"
	DB_PASS   = "A45189C09"
	DB_HOST   = "localhost"
	DB_PORT   = "5432"
	DB_DRIVER = structql.Postgres
)

func ConnectSQL() (*structql.Connection, error) {
	logger.Info("Attempting to Connect to SQL Server")

	dbConfig, err := GenerateDBConfig()
	if err != nil {
		return nil, fmt.Errorf("unable to generate db config: %v", err)
	}

	conn, err := structql.Connect(dbConfig)
	defer conn.Close()
	if err != nil {
		return nil, fmt.Errorf("undable to connect to sql db: %v", err)
	}

	logger.Info("Successfully connected to db.")

	return conn, nil
}

// GenerateDBConfig assigns appropriate environment variables
// when environment variables don't exist the defaults for testing are applied
func GenerateDBConfig() (structql.ConnectionConfig, error) {

	// DBNAME Env Variable -> Name of database
	dbName := os.Getenv("DB_NAME")
	if len(dbName) == 0 {
		dbName = DB_NAME
	}

	// DBUSER Env Variable -> User for this service
	dbUser := os.Getenv("DB_USER")
	if len(dbUser) == 0 {
		dbUser = DB_USER
	}

	// DBPASS Env Variable -> Pass for this service's user
	dbPass := os.Getenv("DB_PASS")
	if len(dbPass) == 0 {
		dbPass = DB_HOST
	}

	// DBHOST Env Variable -> Address of server
	dbHost := os.Getenv("DB_HOST")
	if len(dbPass) == 0 {
		dbHost = DB_HOST
	}

	// DBHOST Env Variable -> Port of server
	dbPort := os.Getenv("DB_PORT")
	if len(dbHost) == 0 {
		dbPort = DB_PORT
	}

	// Configuration for test db
	// NOTE: PRODUCTION DEPLOYMENTS MUST USE SECURED PASSWORDS
	dbConfig := structql.ConnectionConfig{
		Database: dbName,
		User:     dbUser,
		Password: dbPass,
		Host:     dbHost,
		Port:     dbPort,
		Driver:   structql.Postgres,
	}

	return dbConfig, nil

}
