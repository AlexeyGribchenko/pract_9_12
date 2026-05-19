package antifraud

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// GeoIPChecker проверяет тип IP-адреса
type GeoIPChecker struct {
	cache  CacheInterface
	apiURL string
	apiKey string
	client *http.Client
}

func NewGeoIPChecker(cache CacheInterface, apiURL string, apiKey string) *GeoIPChecker {
	return &GeoIPChecker{
		cache:  cache,
		apiURL: apiURL,
		apiKey: apiKey,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// CheckIP проверяет IP-адрес (residential/datacenter/vpn)
func (gc *GeoIPChecker) CheckIP(ctx context.Context, ip string) (string, error) {
	// Проверяем локальные IP
	if isLocalIP(ip) {
		return "residential", nil
	}

	// Пытаемся получить из кэша
	cacheKey := "geoip:check:" + ip
	cachedType, err := gc.cache.Get(ctx, cacheKey)
	if err == nil {
		log.Printf("GeoIP check for %s: %s (from cache)", ip, cachedType)
		return cachedType, nil
	}

	// Используем реальный GeoIP API ipapi.is
	ipType, err := gc.fetchFromGeoIPAPI(ip)
	if err != nil {
		log.Printf("Error fetching from GeoIP API: %v, using default classification", err)
		// Fallback на классификацию по диапазонам
		ipType = gc.classifyIP(ip)
	}

	if err := gc.cache.Set(ctx, cacheKey, ipType, 24*time.Hour); err != nil {
		log.Printf("Error caching GeoIP result: %v", err)
	}

	log.Printf("GeoIP check for %s: %s", ip, ipType)
	return ipType, nil
}

// classifyIP классифицирует IP адрес на основе диапазонов (fallback)
func (gc *GeoIPChecker) classifyIP(ip string) string {
	// Известные диапазоны дата-центров (упрощенный список)
	dcRanges := []string{
		"8.8.",    // Google
		"1.1.",    // Cloudflare
		"208.67.", // OpenDNS
		"9.9.",    // Quad9
	}

	for _, dcRange := range dcRanges {
		if len(ip) >= len(dcRange) && ip[:len(dcRange)] == dcRange {
			return "datacenter"
		}
	}

	// Остальные - жилые IP
	return "residential"
}

// IPAPIResponse - ответ от ipapi.is
type IPAPIResponse struct {
	IP            string `json:"ip"`
	IsDatacenter  bool   `json:"is_datacenter"`
	IsVPN         bool   `json:"is_vpn"`
	IsResidential bool   `json:"is_residential"`
	Country       string `json:"country"`
	CountryCode   string `json:"country_code"`
	City          string `json:"city"`
	Asn           int    `json:"asn"`
	Organization  string `json:"organization"`
	Hostname      string `json:"hostname"`
}

// fetchFromGeoIPAPI вызывает реальный GeoIP API (ipapi.is)
func (gc *GeoIPChecker) fetchFromGeoIPAPI(ip string) (string, error) {
	if gc.apiKey == "" || gc.apiURL == "" {
		return gc.classifyIP(ip), nil
	}

	// Формируем URL запроса для ipapi.is
	url := fmt.Sprintf("%s?ip=%s&key=%s", gc.apiURL, ip, gc.apiKey)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", "VotingSystem/1.0")

	resp, err := gc.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("GeoIP API returned status %d: %s", resp.StatusCode, string(body))
		return "", fmt.Errorf("GeoIP API returned status %d", resp.StatusCode)
	}

	var result IPAPIResponse
	err = json.Unmarshal(body, &result)
	if err != nil {
		log.Printf("Error unmarshaling GeoIP response: %v", err)
		return "", err
	}

	log.Printf("GeoIP API response for %s: datacenter=%v, vpn=%v, residential=%v", ip, result.IsDatacenter, result.IsVPN, result.IsResidential)

	// Определяем тип IP на основе ответа API
	if result.IsDatacenter {
		return "datacenter", nil
	}
	if result.IsVPN {
		return "vpn", nil
	}
	if result.IsResidential {
		return "residential", nil
	}

	// Если API не смог определить, используем fallback
	return gc.classifyIP(ip), nil
}

// isLocalIP проверяет, является ли IP локальным
func isLocalIP(ip string) bool {
	localRanges := []string{
		"127.",     // localhost
		"192.168.", // Private
		"10.",      // Private
		"172.16.",  // Private (172.16.0.0 - 172.31.255.255)
		"172.17.",
		"172.18.",
		"172.19.",
		"172.20.",
		"172.21.",
		"172.22.",
		"172.23.",
		"172.24.",
		"172.25.",
		"172.26.",
		"172.27.",
		"172.28.",
		"172.29.",
		"172.30.",
		"172.31.",
	}

	for _, localRange := range localRanges {
		if len(ip) >= len(localRange) && ip[:len(localRange)] == localRange {
			return true
		}
	}

	return false
}
