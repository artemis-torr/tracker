package main

import (
	"os"
	"strconv"
)

type ConfigType struct {
	HTTPPort  string
	UDPPort   string
	MongoURI  string
	RateLimit int
}

var Config ConfigType

func initConfig() {
	Config.HTTPPort = getEnv("TRACKER_PORT", "8080")
	Config.UDPPort = getEnv("TRACKER_UDP_PORT", "8081")
	Config.MongoURI = getEnv("MONGO_URI", "mongodb://localhost:27017")
	Config.RateLimit = getEnvAsInt("RATE_LIMIT", 10)
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultVal int) int {
	if valueStr, exists := os.LookupEnv(key); exists {
		value, err := strconv.Atoi(valueStr)
		if err == nil {
			return value

		}
	}
	return defaultVal
}
