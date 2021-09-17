package main

import (
	"github.com/inflowml/logger"
)

func main() {
	err := InitSQL()
	if err != nil {
		logger.Fatal("failed to init db: %v", err)
	}

	logger.Info("Starting HTTP Server")
	logger.Fatal("server encountered error: %v", serve())
}
