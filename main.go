// Package main is the entry point for the Firefly framework.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/zhangpeihaoks/firefly/conf"
	"github.com/zhangpeihaoks/firefly/pkg/config"
	"github.com/zhangpeihaoks/firefly/pkg/log"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "./config/config.yaml", "configuration file path")
	flag.Parse()

	// Load configuration
	cfg := conf.DefaultBootstrap()
	c := config.New()
	if err := c.Load(*configPath, cfg); err != nil {
		fmt.Printf("failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logger, cleanup := log.New(&log.Config{
		FileName:   cfg.Log.FileName,
		MaxSize:    cfg.Log.MaxSize,
		MaxBackups: cfg.Log.MaxBackups,
		MaxAge:     cfg.Log.MaxAge,
		Level:      cfg.Log.Level,
		JSONFormat: cfg.Log.JSONFormat,
		Location:   cfg.Log.Location,
	})
	defer cleanup()

	logger.Info("firefly server starting",
		"name", cfg.Name,
		"version", cfg.Version,
	)

	// TODO: Initialize servers and run application
	// This will be implemented in subsequent tasks

	logger.Info("firefly server initialized successfully")
}
