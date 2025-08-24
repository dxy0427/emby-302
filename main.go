package main

import (
	"log"
	"net/http"

	"github.com/dxy0427/emby-302/config"
	"github.com/dxy0427/emby-302/handler"
)

func main() {
	log.Println("[INFO] 正在启动 emby-302...")

	cfg, err := config.LoadConfig("/app/config.yml")
	if err != nil {
		log.Fatalf("[FATAL] %v", err)
	}

	app, err := handler.NewAppState(cfg)
	if err != nil {
		log.Fatalf("[FATAL] %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", app.RootHandler)

	addr := ":" + cfg.Server.Port
	log.Printf("[INFO] 服务启动成功, 监听地址: http://0.0.0.0%s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("[FATAL] 服务启动失败: %v", err)
	}
}