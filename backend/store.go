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

// Default database configuration for non-production deployments
const (
	// Table Names
	IMAGE_TABLE = "image_meta"
	USER_TABLE  = "user_meta"
	PASS_TABLE  = "user_pass"

	// Default DB Configuration
	DB_NAME   = "dbtest"
	DB_USER   = "tester"
	DB_PASS   = "testpass"
	DB_HOST   = "localhost"
	DB_PORT   = "5432"
	DB_DRIVER = structql.Postgres
)

// InitSQL attempts to connect to the database and generates necessary tables if required
func InitSQL() error {
	logger.Info("Attempting to initialize database")

	// Connect to database
	conn, err := connectSQL()
	if err != nil {
		return fmt.Errorf("failed to connect to database: %v", err)
	}
	defer conn.Close()

	// Create image_meta table if it doesn't already exist
	err = conn.CreateTableFromObject(IMAGE_TABLE, Image{})
	if err != nil {
		return fmt.Errorf("failed to create image_meta table: %v", err)
	}

	// Create user_meta table if it doesn't already exist
	err = conn.CreateTableFromObject(USER_TABLE, User{})
	if err != nil {
		return fmt.Errorf("failed to create user_meta table: %v", err)
	}

	// Create user_pass table if it doesn't already exist
	err = conn.CreateTableFromObject(PASS_TABLE, UserPassword{})
	if err != nil {
		return fmt.Errorf("failed to create user_meta table: %v", err)
	}

	logger.Info("Database successfully initialized")

	return nil
}

// AddImageMeta inserts a row into the image_meta table and returns the assigned id
func AddImageData(imgData Image) (int32, error) {

	conn, err := connectSQL()
	if err != nil {
		return 0, fmt.Errorf("unable to add image meta to db due to connection error: %v", err)
	}
	defer conn.Close()

	id, err := conn.InsertObject(IMAGE_TABLE, imgData)
	if err != nil {
		return 0, fmt.Errorf("unable to add image meta due to insertion error: %v", err)
	}

	return int32(id), nil
}

// UpdateImageData accepts an imgData objects and updates the corresponding row to match the parameter
func UpdateImageData(imgData Image) error {
	conn, err := connectSQL()
	if err != nil {
		return fmt.Errorf("unable to update image meta to db due to connection error: %v", err)
	}
	defer conn.Close()

	err = conn.UpdateObject(IMAGE_TABLE, imgData)
	if err != nil {
		return fmt.Errorf("unable to update image meta: %v", err)
	}

	return nil
}

// DeleteImageData deletes the row corresponding to the imageData provided in the func parameter
func DeleteImageData(imageData Image) error {
	conn, err := connectSQL()
	if err != nil {
		return fmt.Errorf("unable to delete image meta to db due to connection error: %v", err)
	}
	defer conn.Close()

	err = conn.DeleteObject(IMAGE_TABLE, imageData)
	if err != nil {
		return fmt.Errorf("unable to delete image meta: %v", err)
	}

	return nil
}

// AddUserMeta inserts a row into the image_meta table and returns the assigned id
func AddUserData(userData User) (int32, error) {

	conn, err := connectSQL()
	if err != nil {
		return 0, fmt.Errorf("unable to add user meta to db due to connection error: %v", err)
	}
	defer conn.Close()

	id, err := conn.InsertObject(USER_TABLE, userData)
	if err != nil {
		return 0, fmt.Errorf("unable to add user meta due to insertion error: %v", err)
	}

	return int32(id), nil
}

// UpdateUserMeta updates the corresponding row into the user_meta table according to the provided parameter
func UpdateUserData(userData User) error {

	conn, err := connectSQL()
	if err != nil {
		return fmt.Errorf("unable to update user meta to db due to connection error: %v", err)
	}
	defer conn.Close()

	err = conn.UpdateObject(USER_TABLE, userData)
	if err != nil {
		return fmt.Errorf("unable to update user meta: %v", err)
	}

	return nil
}

// DeleteUserMeta deletes the corresponding row from the user_meta tables
func DeleteUserData(userData User) error {

	conn, err := connectSQL()
	if err != nil {
		return fmt.Errorf("unable to delete user meta to db due to connection error: %v", err)
	}
	defer conn.Close()

	err = conn.DeleteObject(USER_TABLE, userData)
	if err != nil {
		return fmt.Errorf("unable to delete user meta: %v", err)
	}

	return nil
}

// AddUserMeta inserts a row into the image_meta table and returns the assigned id
func AddUserPass(pass UserPassword) (int32, error) {

	conn, err := connectSQL()
	if err != nil {
		return 0, fmt.Errorf("unable to add user pass to db due to connection error: %v", err)
	}
	defer conn.Close()

	id, err := conn.InsertObject(PASS_TABLE, pass)
	if err != nil {
		return 0, fmt.Errorf("unable to add user pass due to insertion error: %v", err)
	}

	return int32(id), nil
}

// UpdateUserMeta updates the corresponding row into the user_meta table according to the provided parameter
func UpdateUserPass(pass UserPassword) error {

	conn, err := connectSQL()
	if err != nil {
		return fmt.Errorf("unable to update user pass to db due to connection error: %v", err)
	}
	defer conn.Close()

	err = conn.UpdateObject(PASS_TABLE, pass)
	if err != nil {
		return fmt.Errorf("unable to update user pass: %v", err)
	}

	return nil
}

// DeleteUserMeta deletes the corresponding row from the user_meta tables
func DeleteUserPass(pass UserPassword) error {

	conn, err := connectSQL()
	if err != nil {
		return fmt.Errorf("unable to delete user pass to db due to connection error: %v", err)
	}
	defer conn.Close()

	err = conn.DeleteObject(PASS_TABLE, pass)
	if err != nil {
		return fmt.Errorf("unable to delete user pass: %v", err)
	}

	return nil
}

func GetHashedPass(email string) (string, User, error) {
	conn, err := connectSQL()
	if err != nil {
		return "", User{}, fmt.Errorf("unable to delete user pass to db due to connection error: %v", err)
	}
	defer conn.Close()

	userRows, err := conn.SelectFromWhere(User{}, USER_TABLE, fmt.Sprintf("email='%s'", email))
	if err != nil {
		return "", User{}, fmt.Errorf("selection failed, unable to retrieve hashed uid: %v", err)
	}

	if len(userRows) != 1 {
		return "", User{}, fmt.Errorf("cannot find email")
	}

	user := userRows[0].(User)

	passRows, err := conn.SelectFromWhere(UserPassword{}, PASS_TABLE, fmt.Sprintf("id=%v", user.Uid))
	if err != nil {
		return "", User{}, fmt.Errorf("selection failed, unable to retrieve hashed uid: %v", err)
	}

	if len(userRows) != 1 {
		return "", User{}, fmt.Errorf("cannot find hashed pass")
	}

	pass := passRows[0].(UserPassword)

	return pass.HashedPass, user, nil
}

// UniqueEmail queries the user_table in order to determine if an email is unique
func UniqueEmail(email string) (bool, error) {
	conn, err := connectSQL()
	if err != nil {
		return false, fmt.Errorf("unable to connect to database: %v", err)
	}
	defer conn.Close()

	users, err := conn.SelectFromWhere(User{}, USER_TABLE, fmt.Sprintf("email='%s'", email))
	if err != nil {
		return false, fmt.Errorf("unable to query user table: %v", err)
	}
	if len(users) > 0 {
		return false, nil
	}

	return true, nil
}

// connectSQL returns structql Connection this must be closed after the the database action is done
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

	return dbConfig, nil

}
