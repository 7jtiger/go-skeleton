package logger

import (
	"bytes"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"runtime/debug"
	"strings"

	"time"

	"basesk/common/utils"
	"basesk/conf"

	// "github.com/gin-gonic/gin"

	"github.com/gin-gonic/gin"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var logger *zap.Logger
var stag string

func InitLogger(cfg *conf.Config) error {
	now := time.Now()
	lPath := fmt.Sprintf("%s_%s.log", cfg.LogInfo.Fpath, now.Format("2006-01-02"))

	rotator, err := rotatelogs.New(
		lPath,
		rotatelogs.WithMaxAge(time.Duration(cfg.LogInfo.MaxAgeHour)*time.Hour),
		rotatelogs.WithRotationTime(time.Duration(cfg.LogInfo.RotateHour)*time.Hour))
	if err != nil {
		return err
	}

	encCfg := zapcore.EncoderConfig{
		TimeKey:        "date",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	w := zapcore.AddSync(rotator)
	cw := zapcore.AddSync(os.Stdout)
	var core zapcore.Core

	stag = cfg.Server.Mode
	if stag == "dev" {
		core = zapcore.NewTee(
			zapcore.NewCore(zapcore.NewJSONEncoder(encCfg), w, zap.DebugLevel),
			zapcore.NewCore(zapcore.NewConsoleEncoder(encCfg), cw, zap.DebugLevel),
		)
	} else {
		core = zapcore.NewCore(zapcore.NewJSONEncoder(encCfg), w, zap.InfoLevel)
	}
	logger = zap.New(core)

	logger.Info("logging init file start")
	return nil
}

// msg := fmt.Sprint(ctx...)
// switch level {
// case zapcore.DebugLevel:
// 	logger.Debug("debug", zap.String("Debug", msg))

func Debug(ctx ...interface{}) {
	var b bytes.Buffer
	for _, str := range ctx {
		b.WriteString(fmt.Sprintf("%v", str))
	}

	logger.Debug("debug", zap.String("Debug", b.String()))
}

// Info is a convenient alias for Root().Info
func Info(ctx ...interface{}) {
	var b bytes.Buffer

	for _, str := range ctx {
		b.WriteString(fmt.Sprintf("%v", str))
	}
	// logger.Info("info", zap.String("Info", b.String()))
	logger.Info("info", zap.String("Info", b.String()))
}

// Warn is a convenient alias for Root().Warn
func Warn(ctx ...interface{}) {
	var b bytes.Buffer
	for _, str := range ctx {
		b.WriteString(fmt.Sprintf("%v", str))
	}

	logger.Warn("warn", zap.String("Warn", b.String()))
}

// Error is a convenient alias for Root().Error
func Error(ctx ...interface{}) {
	var b bytes.Buffer
	for _, str := range ctx {
		b.WriteString(fmt.Sprintf("%v", str))
	}

	logger.Error("error", zap.String("Err", b.String()))
	if stag != "dev" {
		go utils.SendTelegramAlert(stag, b.String())
	}
}

func Crit(ctx ...interface{}) {
	var b bytes.Buffer
	for _, str := range ctx {
		b.WriteString(fmt.Sprintf("%v", str))
	}

	logger.Panic("panic", zap.String("Crit", b.String()))
	if stag != "dev" {
		go utils.SendTelegramAlert(stag, b.String())
	}
}

// func Debug(ctx ...interface{}) { log(zapcore.DebugLevel, ctx...) }
// func Info(ctx ...interface{}) { log(zapcore.InfoLevel, ctx...) }
// func Warn(ctx ...interface{}) { log(zapcore.WarnLevel, ctx...) }
// func Error(ctx ...interface{}) { log(zapcore.ErrorLevel, ctx...) }
// func Crit(ctx ...interface{}) { log(zapcore.PanicLevel, ctx...) }

func GinLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		c.Next()

		cost := time.Since(start)
		logger.Info(path,
			zap.Int("status", c.Writer.Status()),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("ip", c.ClientIP()),
			zap.String("user-agent", c.Request.UserAgent()),
			zap.String("errors", c.Errors.ByType(gin.ErrorTypePrivate).String()),
			zap.Duration("cost", cost),
		)
	}
}

func GinRecovery(stack bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Check for a broken connection, as it is not really a
				// condition that warrants a panic stack trace.
				var brokenPipe bool
				if ne, ok := err.(*net.OpError); ok {
					if se, ok := ne.Err.(*os.SyscallError); ok {
						if strings.Contains(strings.ToLower(se.Error()), "broken pipe") || strings.Contains(strings.ToLower(se.Error()), "connection reset by peer") {
							brokenPipe = true
						}
					}
				}

				httpRequest, _ := httputil.DumpRequest(c.Request, false)
				if brokenPipe {
					logger.Error(c.Request.URL.Path,
						zap.Any("error", err),
						zap.String("request", string(httpRequest)),
					)
					// If the connection is dead, we can't write a status to it.
					c.Error(err.(error)) // nolint: errcheck
					c.Abort()
					return
				}

				if stack {
					logger.Error("[Recovery from panic]",
						zap.Any("error", err),
						zap.String("request", string(httpRequest)),
						zap.String("stack", string(debug.Stack())),
					)
				} else {
					logger.Error("[Recovery from panic]",
						zap.Any("error", err),
						zap.String("request", string(httpRequest)),
					)
				}
				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()
		c.Next()
	}
}
