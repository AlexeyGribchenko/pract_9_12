package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"voting-app/cache"
	"voting-app/config"
	"voting-app/db"
	"voting-app/handler"
	"voting-app/models"
	"voting-app/repository"
	"voting-app/service"

	"github.com/go-chi/chi/v5"
	"github.com/nats-io/nats.go"
)

func main() {
	log.Println("Starting Gateway Service...")

	cfg := config.Load()

	// Подключаемся к БД
	database, err := db.Connect(cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Инициализируем схему БД
	err = db.InitSchema(database)
	if err != nil {
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

	// Инициализируем репозиторий и сервис
	pollRepo := repository.NewPollRepository(database)
	pollService := service.NewPollService(pollRepo, redisCache)

	// Инициализируем обработчик
	gatewayHandler := handler.NewGatewayHandler(pollService)
	gatewayHandler.SetVotePublisher(natsVotePublisher{conn: nc})

	// Создаем маршруты
	r := chi.NewRouter()

	// Health check
	r.Get("/health", gatewayHandler.Health)

	// Опросы
	r.Post("/polls", gatewayHandler.CreatePoll)
	r.Get("/polls/{pollID}", gatewayHandler.GetPoll)
	r.Get("/polls/{pollID}/results", gatewayHandler.GetResults)
	r.Post("/polls/{pollID}/close", gatewayHandler.ClosePoll)

	r.Post("/polls/{pollID}/vote", gatewayHandler.Vote)

	r.Get("/polls/{pollID}/vote-status", gatewayHandler.GetVoteStatus)

	// Создаем HTTP сервер
	server := &http.Server{
		Addr:    ":" + cfg.GatewayPort,
		Handler: r,
	}

	// Запускаем сервер в отдельной горутине
	log.Printf("Gateway Service listening on port %s", cfg.GatewayPort)
	go func() {
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error starting server: %v", err)
		}
	}()

	// Настраиваем graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigChan
	log.Printf("Received signal: %v", sig)
	log.Println("Shutting down Gateway Service...")

	// Даем серверу 10 секунд на graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = server.Shutdown(ctx)
	if err != nil {
		log.Printf("Error during server shutdown: %v", err)
	}

	log.Println("✓ Gateway Service shutdown complete")
}

type natsVotePublisher struct {
	conn *nats.Conn
}

func (p natsVotePublisher) PublishVote(event models.VoteEventMessage) error {
	eventData, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return p.conn.Publish("votes.new", eventData)
}
