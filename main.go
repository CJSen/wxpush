package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"wxpush/internal/config"
	"wxpush/internal/handler"
	"wxpush/internal/web"
)

func main() {
	// 读取配置并初始化依赖
	cfg, err := config.LoadFromFile("config.yaml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config error: %v\n", err)
		os.Exit(1)
	}
	web.SetMessagePagePath(cfg.MessageHtml)
	h, err := handler.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "init error: %v\n", err)
		os.Exit(1)
	}

	// 为每个入口挂载请求日志
	http.Handle("/", handler.WrapWithLogging("root", http.HandlerFunc(h.HandleRoot)))
	http.Handle("/wxsend", handler.WrapWithLogging("wxsend", http.HandlerFunc(h.HandleWxSend)))
	http.Handle("/msg", handler.WrapWithLogging("msg", http.HandlerFunc(h.HandleMsg)))

	addr := fmt.Sprintf(":%d", cfg.Port)

	server := &http.Server{
		Addr: addr,
	}
	log.Printf("Starting server on http://localhost%s\n", addr)

	if err := server.ListenAndServe(); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
