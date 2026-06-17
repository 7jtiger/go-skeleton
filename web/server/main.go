package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"

	"livein-web/conf"
	"livein-web/controller"
	"livein-web/router"
	"livein-web/store"
)

//go:embed dist/*
var embeddedDist embed.FS

func main() {
	configPath := flag.String("config", "./conf/config.toml", "config file path")
	useEmbed := flag.Bool("embed", false, "serve embedded static files")
	flag.Parse()

	cfg, err := conf.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	s := store.New(cfg.Data.ContentPath, cfg.Data.InquiryPath)
	ctrl := controller.New(cfg, s)

	var distFS fs.FS
	if *useEmbed {
		sub, err := fs.Sub(embeddedDist, "dist")
		if err != nil {
			log.Fatalf("embed dist: %v", err)
		}
		distFS = sub
	}

	r := router.New(cfg, ctrl, distFS)
	engine := r.Setup()

	addr := cfg.Server.Port
	fmt.Printf("Livein web server listening on %s\n", addr)
	if err := engine.Run(addr); err != nil {
		log.Fatalf("server error: %v", err)
		os.Exit(1)
	}
}
