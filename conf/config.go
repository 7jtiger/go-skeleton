package conf

import (
	"os"

	"github.com/naoina/toml"
)

type Works struct {
	Name     string
	Desc     string
	Execute  string
	Duration int
	Start    int
	Args     string
}

type Networks struct {
	Rpc string
}

type Config struct {
	Server struct {
		Mode string
		Port string
	}

	Network Networks

	DB map[string]map[string]interface{}

	// Works []Work
	Works []Works

	LogInfo struct {
		Fpath      string
		MaxAgeHour int
		RotateHour int
	}

	// WhiteList map[string]string
	WhiteList struct {
		Ips []string
	}
	// WhiteList map[string]string
}

func NewConfig(fpath string) *Config {
	c := new(Config)

	if file, err := os.Open(fpath); err != nil {
		panic(err)
	} else {
		defer file.Close()
		if err := toml.NewDecoder(file).Decode(c); err != nil {
			panic(err)
		} else {
			//c.sanitize()
			return c
		}
	}
}
