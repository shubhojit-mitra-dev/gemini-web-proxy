package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/blackknight05/gemini-web-proxy/internal/api"
	"github.com/blackknight05/gemini-web-proxy/internal/auth"
	"github.com/blackknight05/gemini-web-proxy/internal/config"
	"github.com/blackknight05/gemini-web-proxy/internal/gemini"
	"github.com/blackknight05/gemini-web-proxy/internal/models"
	"github.com/blackknight05/gemini-web-proxy/pkg/logger"
)

const version = "1.0.0"

func main() {
	portFlag := flag.Int("port", 0, "Port to listen on")
	hostFlag := flag.String("host", "", "Host address to bind to")
	configFlag := flag.String("config", "", "Path to configuration file")
	cookieFileFlag := flag.String("cookie-file", "", "Path to cookie file")
	proxyFlag := flag.String("proxy", "", "HTTP proxy URL")
	versionFlag := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("gemini-web-proxy v%s\n", version)
		os.Exit(0)
	}

	cfg := config.Default()
	cfg.AutoLoad(*configFlag)

	if *portFlag != 0 {
		cfg.Port = *portFlag
	}
	if *hostFlag != "" {
		cfg.Host = *hostFlag
	}
	if *cookieFileFlag != "" {
		cfg.CookieFile = *cookieFileFlag
	}
	if *proxyFlag != "" {
		cfg.Proxy = *proxyFlag
	}

	xsrfManager := auth.NewXSRFManager(30 * time.Minute)
	geminiClient := gemini.NewClient(cfg, xsrfManager)
	server := api.NewServer(cfg, geminiClient)

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	httpServer := &http.Server{
		Addr:         addr,
		Handler:      server,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: time.Duration(cfg.RequestTimeoutSec) * time.Second,
	}

	// Print DX startup summary
	fmt.Printf("gemini-web-proxy v%s\n", version)
	fmt.Printf("  Listening: http://%s\n", addr)
	fmt.Printf("  Base URL:  http://localhost:%d/v1\n", cfg.Port)
	fmt.Printf("  Models:    %d registered\n", len(models.Registry))
	if cfg.CookieFile != "" {
		fmt.Printf("  Cookie:    %s\n", cfg.CookieFile)
	} else {
		fmt.Printf("  Cookie:    none (anonymous)\n")
	}
	if cfg.Proxy != "" {
		fmt.Printf("  Proxy:     %s\n", cfg.Proxy)
	} else {
		fmt.Printf("  Proxy:     direct\n")
	}
	fmt.Println()

	// Handle graceful shutdown
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server failed: %v", err)
			os.Exit(1)
		}
	}()

	<-stopChan
	logger.Info("Shutting down proxy server gracefully...")
	os.Exit(0)
}
