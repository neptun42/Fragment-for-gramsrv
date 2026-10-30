package config

import (
	"errors"
	"os"
	"strconv"
)

type Config struct {
	Addr         string
	DatabaseURL  string
	SecretKey    string
	AdminToken   string
	AutoApprove  bool
	WelcomeBonus int64
	CORSOrigin   string
	StaticDir    string
	ServersFile  string
}

// всё берём из env, дефолты под локальный запуск
func Load() (Config, error) {
	c := Config{
		Addr:        env("LUNAX_ADDR", ":8080"),
		DatabaseURL: env("LUNAX_DATABASE_URL", "postgres://lunax:lunax@localhost:5432/lunax?sslmode=disable"),
		SecretKey:   os.Getenv("LUNAX_SECRET_KEY"),
		AdminToken:  os.Getenv("LUNAX_ADMIN_TOKEN"),
		CORSOrigin:  os.Getenv("LUNAX_CORS_ORIGIN"),
		StaticDir:   os.Getenv("LUNAX_STATIC_DIR"),
		ServersFile: os.Getenv("LUNAX_SERVERS_FILE"),
	}
	var err error
	if c.AutoApprove, err = boolEnv("LUNAX_AUTO_APPROVE", false); err != nil {
		return c, err
	}
	if c.WelcomeBonus, err = intEnv("LUNAX_WELCOME_BONUS", 0); err != nil {
		return c, err
	}
	if c.SecretKey == "" {
		return c, errors.New("LUNAX_SECRET_KEY is required, it encrypts server tokens")
	}
	return c, nil
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func boolEnv(key string, def bool) (bool, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def, errors.New(key + ": want true or false")
	}
	return b, nil
}

func intEnv(key string, def int64) (int64, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil || n < 0 {
		return def, errors.New(key + ": want a positive number")
	}
	return n, nil
}
