package config

import log "github.com/sirupsen/logrus"

func SetupLogging(config *Config) {
	// TODO: add setup
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})
	// TODO: set level
}
