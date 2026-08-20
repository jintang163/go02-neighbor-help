package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Addr            string
	DataPath        string
	SessionTTL      time.Duration
	SeedAdmin       bool
	SeedDemo        bool
	AdminUsername   string
	AdminPassword   string
	ShutdownTimeout time.Duration
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	CORSOrigins     []string
	MaxOpenPosts    int
}

func Load() (Config, error) {
	get := envOr("APP_", "")
	c := Config{
		Addr:            get("ADDR", ":8080"),
		DataPath:        get("DATA_PATH", "data/store.json"),
		SeedAdmin:       envBool("APP_SEED_ADMIN", true),
		SeedDemo:        envBool("APP_SEED_DEMO", true),
		AdminUsername:   get("ADMIN_USERNAME", "admin"),
		AdminPassword:   get("ADMIN_PASSWORD", "admin123"),
		ShutdownTimeout: envDur("APP_SHUTDOWN_TIMEOUT", 10*time.Second),
		ReadTimeout:     envDur("APP_READ_TIMEOUT", 15*time.Second),
		WriteTimeout:    envDur("APP_WRITE_TIMEOUT", 15*time.Second),
		CORSOrigins:     envList("APP_CORS_ORIGINS"),
		MaxOpenPosts:    envInt("APP_MAX_OPEN_POSTS", 10),
	}
	if c.MaxOpenPosts <= 0 {
		c.MaxOpenPosts = 10
	}
	ttl, err := envDurErr("APP_SESSION_TTL", 24*time.Hour)
	if err != nil {
		return c, fmt.Errorf("parse APP_SESSION_TTL: %w", err)
	}
	c.SessionTTL = ttl
	return c, nil
}

func envOr(prefix, _ string) func(key, def string) string {
	return func(key, def string) string {
		v := strings.TrimSpace(os.Getenv(prefix + key))
		if v == "" {
			return def
		}
		return v
	}
}

func envBool(key string, def bool) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	switch v {
	case "1", "true", "yes":
		return true
	case "0", "false", "no":
		return false
	default:
		return def
	}
}

func envDur(key string, def time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}

func envDurErr(key string, def time.Duration) (time.Duration, error) {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def, err
	}
	return d, nil
}

func envList(key string) []string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func envInt(key string, def int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}
