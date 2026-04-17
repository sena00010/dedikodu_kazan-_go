package main

import (
	"context"
	"log"

	"dedikodu-kazani/backend/internal/ai"
	"dedikodu-kazani/backend/internal/auth"
	"dedikodu-kazani/backend/internal/config"
	"dedikodu-kazani/backend/internal/database"
	"dedikodu-kazani/backend/internal/httpx"
	"dedikodu-kazani/backend/internal/realtime"
	"dedikodu-kazani/backend/internal/repository"
)

func main() {
	cfg := config.Load()
	db, err := database.OpenMySQL(cfg.MySQLDSN)
	if err != nil {
		log.Fatalf("mysql: %v", err)
	}
	redis := database.OpenRedis(cfg.RedisAddr, cfg.RedisPassword)
	useRedis := true
	if err := redis.Ping(context.Background()).Err(); err != nil {
		log.Printf("redis unavailable, websocket pub/sub across processes will not work: %v", err)
		useRedis = false
	}
	firebaseVerifier, err := auth.NewFirebaseVerifier(context.Background(), cfg.FirebaseProjectID, cfg.FirebaseCredentialsFile)
	if err != nil {
		log.Printf("firebase disabled until configured: %v", err)
	}
	router := httpx.New(cfg, repository.New(db), firebaseVerifier, ai.New(cfg), realtime.NewHub(redis, useRedis))
	log.Printf("%s API listening on %s", cfg.AppName, cfg.HTTPAddr)
	if err := router.Run(cfg.HTTPAddr); err != nil {
		log.Fatal(err)
	}
}
