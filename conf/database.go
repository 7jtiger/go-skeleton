package conf

import (
	"fmt"
	"github.com/go-sql-driver/mysql"
	"time"
)

// MySQLDSN uses only the named database in this project's TOML configuration.
// Callers must never log the returned DSN, which contains credentials.
func (c *Config) MySQLDSN(name string) (string, error) {
	values, ok := c.DB[name]
	if !ok {
		return "", fmt.Errorf("missing database configuration: %s", name)
	}
	get := func(key string) string { value, _ := values[key].(string); return value }
	m := mysql.NewConfig()
	m.User, m.Passwd = get("user"), get("pass")
	m.Net, m.Addr, m.DBName = "tcp", get("host"), get("name")
	if m.User == "" || m.Addr == "" || m.DBName == "" {
		return "", fmt.Errorf("database %s requires user, host and name", name)
	}
	m.ParseTime = true
	m.Timeout = 5 * time.Second
	m.ReadTimeout = 2 * time.Minute
	m.WriteTimeout = 10 * time.Second
	m.Params = map[string]string{"charset": "utf8"}
	return m.FormatDSN(), nil
}
