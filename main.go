package main

import (
	"flag"

	"github.com/electroprovodka/loadbalancer/config"
	"github.com/electroprovodka/loadbalancer/proxy"
	log "github.com/sirupsen/logrus"
)

func parseFlags() string {
	var config string
	flag.StringVar(&config, "config", "", "path to the proxy config")
	flag.Parse()
	if config == "" {
		log.Fatal("--config is required field to start the server")
	}
	return config
}

func main() {
	// TODO: multiple targets
	// TODO: redirect rules
	// TODO: file config
	// TODO: command line params
	// TODO: hot reload
	// TODO: logging
	// TODO: tests
	// TODO: graceful shutdown
	// TODO: signals processing
	// TODO: https
	// TODO: metrics?
	// TODO: healthchecks?
	// TODO: targets autodiscovery?
	// TODO: API for controlling
	// TODO: Docker image

	configPath := parseFlags()

	cfg, err := config.ReadConfig(configPath)
	if err != nil {
		log.Fatal(err)
		return
	}

	config.SetupLogging(cfg)

	ps, err := proxy.NewProxyServer(cfg)
	if err != nil {
		log.Fatal(err)
		return
	}

	log.Warnf("Starting the server on port :%d\n", cfg.Port)
	ps.Start()
	log.Warn("Server stopped")
}
