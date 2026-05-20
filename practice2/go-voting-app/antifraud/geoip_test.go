package antifraud

import (
	"context"
	"testing"
	"time"
)

func TestGeoIPChecker_LocalIP_ReturnsResidential(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mockCache := NewMockCache()
	checker := NewGeoIPChecker(mockCache, "http://localhost:8080", "test-key")

	tests := []struct {
		name string
		ip   string
	}{
		{"localhost", "127.0.0.1"},
		{"private 10", "10.0.0.1"},
		{"private 172", "172.16.0.1"},
		{"private 192.168", "192.168.1.1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ipType, err := checker.CheckIP(ctx, tt.ip)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if ipType != "residential" {
				t.Errorf("local IP %s should be classified as residential, got %s", tt.ip, ipType)
			}
		})
	}
}

func TestGeoIPChecker_Cache_ReturnsCachedValue(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mockCache := NewMockCache()
	checker := NewGeoIPChecker(mockCache, "http://localhost:8080", "test-key")

	ip := "8.8.8.8"
	cacheKey := "geoip:check:" + ip

	// Устанавливаем значение в кэш
	mockCache.Set(ctx, cacheKey, "datacenter", 24*time.Hour)

	// Получаем значение из кэша
	ipType, err := checker.CheckIP(ctx, ip)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ipType != "datacenter" {
		t.Errorf("expected 'datacenter' from cache, got '%s'", ipType)
	}
}

func TestGeoIPChecker_ClassifyIP_DatacenterRanges(t *testing.T) {
	// Тестируем классификацию IP напрямую через private method тестом
	// Используем структуру checker для доступа к classifyIP
	tests := []struct {
		name         string
		ip           string
		expectedType string
	}{
		{"Google DNS", "8.8.8.8", "datacenter"},
		{"Cloudflare DNS", "1.1.1.1", "datacenter"},
		{"OpenDNS", "208.67.222.222", "datacenter"},
		{"Quad9 DNS", "9.9.9.9", "datacenter"},
		{"Regular IP", "123.45.67.89", "residential"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCache := NewMockCache()
			checker := NewGeoIPChecker(mockCache, "http://localhost:8080", "test-key")

			// Используем classifyIP напрямую вместо CheckIP чтобы избежать проблем с кэшем
			ipType := checker.classifyIP(tt.ip)

			if ipType != tt.expectedType {
				t.Errorf("IP %s: expected %s, got %s", tt.ip, tt.expectedType, ipType)
			}
		})
	}
}

func TestGeoIPChecker_MultipleChecks_UseCacheAfterFirst(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mockCache := NewMockCache()
	checker := NewGeoIPChecker(mockCache, "http://localhost:8080", "test-key")

	ip := "123.45.67.89"

	// Первый запрос
	ipType1, _ := checker.CheckIP(ctx, ip)

	// Проверяем кэш
	cached, _ := mockCache.Get(ctx, "geoip:check:"+ip)
	if cached != ipType1 {
		t.Errorf("result not cached correctly")
	}

	// Второй запрос должен взять из кэша
	ipType2, _ := checker.CheckIP(ctx, ip)

	if ipType1 != ipType2 {
		t.Errorf("cached result should be same: %s vs %s", ipType1, ipType2)
	}
}

func TestGeoIPChecker_DifferentIPs_IndependentClassification(t *testing.T) {
	mockCache := NewMockCache()
	checker := NewGeoIPChecker(mockCache, "http://localhost:8080", "test-key")

	ip1 := "8.8.8.8"       // datacenter
	ip2 := "192.168.1.100" // private/local

	// Тестируем классификацию напрямую
	type1 := checker.classifyIP(ip1)
	type2 := checker.classifyIP(ip2)

	// 192.168.x.x - приватный IP, должна быть residential
	if type2 != "residential" {
		t.Errorf("private IP should be residential, got %s", type2)
	}

	// 8.8.x.x - Google, должна быть datacenter
	if type1 != "datacenter" {
		t.Errorf("Google DNS should be datacenter, got %s", type1)
	}
}

// Additional tests for better coverage
func TestGeoIPChecker_ErrorHandling(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mockCache := NewMockCache()
	checker := NewGeoIPChecker(mockCache, "http://localhost:8080", "test-key")

	// Test with empty IP
	ipType, _ := checker.CheckIP(ctx, "")
	if ipType == "" {
		t.Error("empty IP should still return classification")
	}

	// Local/private IPs should always return residential
	localIPs := []string{"127.0.0.1", "::1", "10.0.0.1", "172.16.0.1"}
	for _, ip := range localIPs {
		ipType, err := checker.CheckIP(ctx, ip)
		if err != nil {
			t.Errorf("unexpected error for %s: %v", ip, err)
		}
		if ipType != "residential" {
			t.Errorf("local IP %s should be residential, got %s", ip, ipType)
		}
	}
}
