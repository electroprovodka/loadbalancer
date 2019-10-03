package main

import log "github.com/sirupsen/logrus"

func setupLogging(config *Config) {
	// TODO: add setup
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})
	// TODO: set level
}
