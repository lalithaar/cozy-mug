// Package settings holds cozy-mug's minimal environment-based configuration
// convention. There is no config file: environment variables, read at boot, are
// the whole story (12-factor). Defaults make a fresh app run with zero config.
package settings

import "os"

// Config is the small, closed set of knobs a cozy-mug app reads. Anything
// outside it is an app concern.
type Config struct {
	DatabaseURL string
	Dev         bool
	Port        string
}

// Load reads the process environment into Config.
func Load() Config {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "app.db"
	}
	dev := true
	if os.Getenv("APP_ENV") == "production" || os.Getenv("DEV") == "0" {
		dev = false
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	return Config{DatabaseURL: dsn, Dev: dev, Port: port}
}
