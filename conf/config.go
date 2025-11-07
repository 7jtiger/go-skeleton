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

// ServerConf is for server config parameters
type ServerConf struct {
	Name      string
	Mode      string
	Port      string
	HCheck    string
	HCPort    string
	BaseKey   string
	JWTSecret string // JWT 토큰을 위한 비밀키
	// Logger    *log.Logger
}

type Config struct {
	Server struct {
		Name      string
		Mode      string
		Port      string
		HCheck    string
		HCPort    string
		BaseKey   string
		JWTSecret string
	}

	WebRTC struct {
		SignalingServerUrl string
		StunServers        []string
		MaxVideoWidth      int
		MaxVideoHeight     int
		MaxVideoFrameRate  int
		VideoBitrate       int
		AudioBitrate       int
		SessionTimeout     int // seconds
		HeartbeatInterval  int // seconds
	}

	HAChecker struct {
		Checker    bool
		ServerPort string
		PeerIP     string
		PeerPort   string
		SetStatus  string
	}
	// Network Networks

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
