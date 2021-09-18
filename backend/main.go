package main

import (
	"github.com/inflowml/logger"
)

func main() {

	// Initialize connection to SQL and establish tables
	err := InitSQL()
	if err != nil {
		logger.Fatal("failed to init db: %v", err)
	}

	// Serve HTTP server and report fatal errors
	logger.Fatal("Server encountered unrecoverable error: %v", serve())
}
