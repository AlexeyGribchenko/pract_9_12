package main

import (
	"context"
	"database/sql"
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
	"voting-app/models"
	"voting-app/repository"
	"voting-app/service"

	"github.com/go-chi/chi/v5"
)

func main() {
	log.Println("Starting Poll Manager Service...")

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

	// Инициализируем репозиторий и сервис
	pollRepo := repository.NewPollRepository(database)
	pollService := service.NewPollService(pollRepo, redisCache)

	// Создаем маршруты
	r := chi.NewRouter()

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "ok",
			"service": "poll-manager",
		})
	})

	// Получить опрос
	r.Get("/polls/{pollID}", func(w http.ResponseWriter, r *http.Request) {
		pollID := chi.URLParam(r, "pollID")

		poll, options, err := pollService.GetPollWithOptions(pollID)
		if err != nil {
			log.Printf("Error getting poll: %v", err)
			http.Error(w, "Error getting poll", http.StatusInternalServerError)
			return
		}

		if poll == nil {
			http.Error(w, "Poll not found", http.StatusNotFound)
			return
		}

		// Возвращаем опрос с вариантами ответов
		pollWithOptions := models.PollWithOptions{
			ID:        poll.ID,
			Title:     poll.Title,
			Status:    poll.Status,
			CreatedAt: poll.CreatedAt,
			ClosedAt:  poll.ClosedAt,
			Options:   options,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(pollWithOptions)
	})

	// Получить конфигурацию опроса
	r.Get("/polls/{pollID}/config", func(w http.ResponseWriter, r *http.Request) {
		pollID := chi.URLParam(r, "pollID")

		config, err := pollService.GetPollConfig(pollID)
		if err != nil {
			log.Printf("Error getting poll config: %v", err)
			http.Error(w, "Error getting poll config", http.StatusInternalServerError)
			return
		}

		if config == nil {
			http.Error(w, "Poll not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(config)
	})

	// Получить результаты
	r.Get("/polls/{pollID}/results", func(w http.ResponseWriter, r *http.Request) {
		pollID := chi.URLParam(r, "pollID")

		results, err := pollService.GetResults(pollID)
		if err != nil {
			log.Printf("Error getting results: %v", err)
			http.Error(w, "Error getting results", http.StatusInternalServerError)
			return
		}

		if results == nil {
			http.Error(w, "Poll not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(results)
	})

	// Создать опрос (внутренний API)
	r.Post("/polls", func(w http.ResponseWriter, r *http.Request) {
		var req models.CreatePollRequest

		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		if req.Title == "" || len(req.Options) == 0 {
			http.Error(w, "Title and options are required", http.StatusBadRequest)
			return
		}

		resp, err := pollService.CreatePoll(req.Title, req.Options)
		if err != nil {
			log.Printf("Error creating poll: %v", err)
			http.Error(w, "Error creating poll", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(resp)
	})

	// Закрыть опрос
	r.Post("/polls/{pollID}/close", func(w http.ResponseWriter, r *http.Request) {
		pollID := chi.URLParam(r, "pollID")

		var req models.ClosePollRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		if req.AdminKey == "" {
			http.Error(w, "admin_key is required", http.StatusBadRequest)
			return
		}

		err = pollService.ClosePoll(pollID, req.AdminKey)
		if err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "Invalid admin key", http.StatusUnauthorized)
			} else {
				log.Printf("Error closing poll: %v", err)
				http.Error(w, "Error closing poll", http.StatusInternalServerError)
			}
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Poll closed",
		})
	})

	// Создаем HTTP сервер
	server := &http.Server{
		Addr:    ":" + cfg.PollManagerPort,
		Handler: r,
	}

	// Запускаем сервер в отдельной горутине
	log.Printf("Poll Manager Service listening on port %s", cfg.PollManagerPort)
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
	log.Println("Shutting down Poll Manager Service...")

	// Даем серверу 10 секунд на graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = server.Shutdown(ctx)
	if err != nil {
		log.Printf("Error during server shutdown: %v", err)
	}

	log.Println("✓ Poll Manager Service shutdown complete")
}
