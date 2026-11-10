// dbtool checks project database connections and applies only bundled sdb migrations.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"github.com/google/uuid"
	"github.com/naoina/toml"
	"github.com/redis/go-redis/v9"
	"ms-gateway/conf"
	"ms-gateway/migrations"
	"ms-gateway/models"
	"os"
	"strings"
	"time"
)

func main() {
	path := flag.String("config", "conf/config.toml", "project configuration file")
	apply := flag.Bool("apply", false, "apply pending story migrations to configured sdb")
	flag.Parse()
	if err := run(*path, *apply); err != nil {
		fmt.Fprintln(os.Stderr, "dbtool:", err)
		os.Exit(1)
	}
}
func run(path string, apply bool) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	cfg := new(conf.Config)
	if err = toml.NewDecoder(file).Decode(cfg); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	var story *sql.DB
	for _, key := range []string{"adb", "hdb", "idb", "sdb"} {
		dsn, err := cfg.MySQLDSN(key)
		if err != nil {
			return err
		}
		db, err := sql.Open("mysql", dsn)
		if err != nil {
			return fmt.Errorf("%s connection configuration is invalid", key)
		}
		defer db.Close()
		db.SetMaxOpenConns(2)
		if err = db.PingContext(ctx); err != nil {
			return fmt.Errorf("%s ping: %w", key, err)
		}
		fmt.Printf("MySQL %s (%s/%s): connected\n", key, cfg.DB[key]["host"], cfg.DB[key]["name"])
		if key == "sdb" {
			story = db
		}
	}
	if host, ok := cfg.DB["rdb"]["host"].(string); !ok || host == "" {
		return fmt.Errorf("rdb.host is required")
	}
	r := redis.NewClient(models.NewRedisOptions(cfg))
	defer r.Close()
	if err = r.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("Redis ping: %w", err)
	}
	key := "ms-gateway:dbcheck:" + uuid.NewString()
	ok, err := r.SetNX(ctx, key, "connection-test", time.Minute).Result()
	if err != nil {
		return fmt.Errorf("Redis SET: %w", err)
	}
	if !ok {
		return fmt.Errorf("Redis test key collision")
	}
	defer r.Del(context.Background(), key)
	value, err := r.Get(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("Redis GET: %w", err)
	}
	if value != "connection-test" {
		return fmt.Errorf("Redis round-trip mismatch")
	}
	if err = r.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("Redis DEL: %w", err)
	}
	info, err := r.Info(ctx, "server").Result()
	if err != nil {
		return fmt.Errorf("Redis INFO: %w", err)
	}
	for _, line := range strings.Split(info, "\n") {
		if strings.HasPrefix(line, "redis_version:") {
			fmt.Println("Redis", cfg.DB["rdb"]["host"], strings.TrimSpace(line), "PING/SET/GET/DEL passed")
		}
	}
	return migrations.Run(ctx, story, apply, os.Stdout)
}
