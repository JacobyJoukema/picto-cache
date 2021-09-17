package main

import (
	"github.com/inflowml/logger"
)

func main() {
	logger.Info("Starting Server")
	logger.Fatal("server encountered error: %v", serve())
}
