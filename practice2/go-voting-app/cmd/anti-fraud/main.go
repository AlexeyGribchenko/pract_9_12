package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
	"voting-app/antifraud"
	"voting-app/cache"
	"voting-app/config"
	"voting-app/db"
	"voting-app/models"
	"voting-app/repository"

	"github.com/nats-io/nats.go"
)

func main() {
	log.Println("Starting Anti-Fraud Service...")

	cfg := config.Load()

	// Подключаемся к БД
	database, err := db.Connect(cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	if err := db.InitSchema(database); err != nil {
		log.Fatalf("Failed to init schema: %v", err)
	}

	// Подключаемся к Redis
	redisCache, err := cache.NewRedisCache(cfg.RedisHost, cfg.RedisPort)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisCache.Close()

	// Подключаемся к NATS
	nc, err := nats.Connect(cfg.NatsURL)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	log.Println("✓ Connected to NATS")

	// Инициализируем репозиторий
	pollRepo := repository.NewPollRepository(database)

	// Инициализируем компоненты Anti-Fraud
	geoipChecker := antifraud.NewGeoIPChecker(redisCache, cfg.GeoIPAPIURL, cfg.GeoIPAPIKey)
	rateLimiter := antifraud.NewRateLimiter(redisCache)
	deduplicator := antifraud.NewDeduplicator(redisCache)
	voteProcessor := antifraud.NewVoteProcessor(pollRepo, redisCache, geoipChecker, rateLimiter, deduplicator)

	// Подписываемся на события голосования
	sub, err := nc.QueueSubscribe("votes.new", "anti-fraud-group", func(msg *nats.Msg) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		var voteEvent models.VoteEventMessage
		err := json.Unmarshal(msg.Data, &voteEvent)
		if err != nil {
			log.Printf("Error unmarshaling vote event: %v", err)
			return
		}

		log.Printf("Processing vote event: %+v", voteEvent)

		// Обрабатываем голос
		success, message, err := voteProcessor.ProcessVote(ctx, &voteEvent)
		if err != nil {
			log.Printf("Error processing vote: %v", err)
		}

		if success {
			log.Printf("Vote processed successfully: %s", message)
		} else {
			log.Printf("Vote rejected: %s", message)
		}
	})

	if err != nil {
		log.Fatalf("Failed to subscribe: %v", err)
	}

	log.Println("✓ Subscribed to votes.new")
	log.Println("Anti-Fraud Service is running, waiting for vote events...")

	// Настраиваем graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigChan
	log.Printf("Received signal: %v", sig)
	log.Println("Shutting down Anti-Fraud Service...")

	// Отписываемся от NATS
	err = sub.Unsubscribe()
	if err != nil {
		log.Printf("Error unsubscribing: %v", err)
	}

	log.Println("✓ Anti-Fraud Service shutdown complete")
}
