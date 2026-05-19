package handler

import (
	"net"
	"net/http"
)

// GetClientIP извлекает IP адрес клиента из запроса
func GetClientIP(r *http.Request) string {
	// Проверяем X-Forwarded-For (для прокси/балансировщиков)
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		return forwarded
	}

	// Проверяем X-Real-IP
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	// Получаем из RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return ip
}
