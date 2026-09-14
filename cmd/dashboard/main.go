package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	corelog "github.com/jasonmiller-cc/parameters-core/pkg/log"
	"github.com/jasonmiller-cc/parameters-core/pkg/middleware"
	"github.com/jasonmiller-cc/parameters-core/pkg/ui"
	"github.com/jasonmiller-cc/parameters-core/pkg/version"
)

func main() {
	var (
		configPath  = flag.String("config", "", "path to dashboard.yaml (default: auto-detect)")
		showVersion = flag.Bool("version", false, "print version and exit")
		genConfig   = flag.Bool("gen-config", false, "print a default dashboard.yaml and exit")
	)
	flag.Parse()

	if *showVersion {
		fmt.Println(version.String())
		return
	}

	if *genConfig {
		printDefaultConfig()
		return
	}

	log := corelog.New(corelog.LevelInfo, corelog.FormatJSON).With("service", "parameters-dashboard")

	var cfg *ui.Config
	var err error

	if *configPath != "" {
		cfg, err = ui.LoadConfig(*configPath)
		if err != nil {
			log.Error("load config", "err", err)
			os.Exit(1)
		}
	} else {
		// Try well-known paths, fall back to defaults.
		for _, p := range []string{"dashboard.yaml", "/etc/parameters/dashboard.yaml"} {
			if _, err := os.Stat(p); err == nil {
				cfg, err = ui.LoadConfig(p)
				if err != nil {
					log.Error("load config", "path", p, "err", err)
					os.Exit(1)
				}
				break
			}
		}
		if cfg == nil {
			log.Info("no config file found, using defaults (localhost ports 8081-8089)")
			cfg = ui.DefaultConfig()
		}
	}

	registry := ui.NewRegistry(cfg.Services)
	log.Info("dashboard starting",
		"addr", cfg.Addr(),
		"services", len(cfg.Services),
		"poll_interval", cfg.PollInterval(),
	)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Start background polling.
	go registry.StartPolling(ctx, cfg.PollInterval())

	mux := http.NewServeMux()
	dash := &ui.DashboardConfig{
		Registry:     registry,
		PollInterval: cfg.PollInterval(),
	}
	dash.Handler(mux)

	chain := middleware.Chain(
		middleware.RequestID,
		middleware.Logger(log),
		middleware.Recover(log),
		middleware.CORS("*"),
	)

	readTimeout := time.Duration(cfg.Server.ReadTimeoutS) * time.Second
	if readTimeout == 0 {
		readTimeout = 15 * time.Second
	}
	writeTimeout := time.Duration(cfg.Server.WriteTimeoutS) * time.Second
	if writeTimeout == 0 {
		writeTimeout = 30 * time.Second
	}

	srv := &http.Server{
		Addr:         cfg.Addr(),
		Handler:      chain(mux),
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	}

	go func() {
		log.Info("dashboard listening", "url", "http://"+cfg.Addr())
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	log.Info("shutting down")

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutCancel()
	_ = srv.Shutdown(shutCtx)
	log.Info("stopped")
}

func printDefaultConfig() {
	fmt.Print(`# parameters-dashboard configuration
server:
  host: 0.0.0.0
  port: 9090

poll:
  interval_s: 30

services:
  - name: dns
    display_name: DNS
    url: http://localhost:8081
    description: DNS zone and record management
    group: network

  - name: dhcp
    display_name: DHCP
    url: http://localhost:8082
    description: DHCP server management
    group: network

  - name: network
    display_name: Network
    url: http://localhost:8083
    description: Network interface and routing
    group: network

  - name: ntp
    display_name: NTP
    url: http://localhost:8084
    description: NTP service management
    group: network

  - name: ldap
    display_name: LDAP
    url: http://localhost:8085
    description: LDAP directory management
    group: identity

  - name: kerberos
    display_name: Kerberos
    url: http://localhost:8086
    description: Kerberos KDC management
    group: identity

  - name: iam
    display_name: IAM
    url: http://localhost:8087
    description: Identity and access management
    group: identity

  - name: ca
    display_name: CA
    url: http://localhost:8088
    description: Certificate authority and ACME
    group: security

  - name: fs
    display_name: Filesystem
    url: http://localhost:8089
    description: Filesystem and quota management
    group: storage
`)
}
