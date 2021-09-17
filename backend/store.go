package main

/*
	This file is designed to encasulate the interation of the server and the database.
	No other module should connect to the database IOT
		- Properly manage SQL connections
		- Maintain DB integrity
		- Discourage stateful systems
*/

import (
	"fmt"
	"os"

	"github.com/inflowml/logger"
	"github.com/inflowml/structql"
)

// Default database configuration for non-production builds
const (
	DB_NAME   = "dbtest"
	DB_USER   = "tester"
	DB_PASS   = "testpass"
	DB_HOST   = "localhost"
	DB_PORT   = "5432"
	DB_DRIVER = structql.Postgres
)

func InitSQL() error {
	logger.Info("Attempting to initialize database")

	// Connect to database
	conn, err := connectSQL()
	if err != nil {
		return fmt.Errorf("failed to connect to database: %v", err)
	}
	defer conn.Close()

	// Create image_meta table if it doesn't already exist
	err = conn.CreateTableFromObject("image_meta", Image{})
	if err != nil {
		return fmt.Errorf("failed to create image_meta table: %v", err)
	}

	// Create user_meta table if it doesn't already exist
	err = conn.CreateTableFromObject("user_meta", User{})
	if err != nil {
		return fmt.Errorf("failed to create user_meta table: %v", err)
	}

	logger.Info("Database successfully initialized")

	return nil
}

func connectSQL() (*structql.Connection, error) {
	dbConfig, err := generateDBConfig()
	if err != nil {
		return nil, fmt.Errorf("unable to generate db config: %v", err)
	}

	conn, err := structql.Connect(dbConfig)
	if err != nil {
		return nil, fmt.Errorf("undable to connect to sql db: %v", err)
	}

	return conn, nil
}

// GenerateDBConfig assigns appropriate environment variables
// when environment variables don't exist the defaults for testing are applied
func generateDBConfig() (structql.ConnectionConfig, error) {

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
		dbPass = DB_PASS
	}

	// DBHOST Env Variable -> Address of server
	dbHost := os.Getenv("DB_HOST")
	if len(dbHost) == 0 {
		dbHost = DB_HOST
	}

	// DBHOST Env Variable -> Port of server
	dbPort := os.Getenv("DB_PORT")
	if len(dbPort) == 0 {
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

	logger.Info("%v", dbConfig)

	return dbConfig, nil

}
