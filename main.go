package main

import (
	"context"
	"log"
	"os"
	"strings"

	"stock_agent/internal/auth"
	"stock_agent/internal/config"
	"stock_agent/internal/reply"
	"stock_agent/internal/store"
	"stock_agent/router"
)

func main() {
	cfg := config.Load()

	chatStore, err := store.NewMySQL(cfg.MySQL.DSN())
	if err != nil {
		log.Fatalf("failed to create mysql store: %v", err)
	}

	if err := chatStore.EnsureSchema(context.Background()); err != nil {
		log.Fatalf("failed to ensure mysql schema: %v", err)
	}

	if _, err := os.Stat(cfg.LegacyChatFile); err == nil {
		log.Printf("legacy chat file %s still exists; this version now uses MySQL and ignores the old global chat file", cfg.LegacyChatFile)
	}

	server := router.New(
		normalizeHostPort(cfg.Port),
		chatStore,
		reply.NewService(),
		auth.NewService(cfg.DefaultAvatar),
		cfg.SessionMaxAge,
		cfg.AvatarUploadDir,
	)
	log.Printf("fund assistant server listening on %s", cfg.Port)
	server.Spin()
}

func normalizeHostPort(port string) string {
	if strings.HasPrefix(port, ":") {
		return port
	}
	if strings.Contains(port, ":") {
		return port
	}
	return ":" + port
}
