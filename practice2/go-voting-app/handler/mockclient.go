package handler

import (
	"bytes"
	"io"
	"net/http"
)

// MockHTTPClient - мок для тестирования HTTP-запросов
type MockHTTPClient struct {
	// Функции-заглушки для разных методов
	DoFunc  func(req *http.Request) (*http.Response, error)
	GetFunc func(url string) (*http.Response, error)
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	if m.DoFunc != nil {
		return m.DoFunc(req)
	}
	return nil, nil
}

func (m *MockHTTPClient) Get(url string) (*http.Response, error) {
	if m.GetFunc != nil {
		return m.GetFunc(url)
	}
	return nil, nil
}

// Вспомогательные функции для создания мок-ответов
func mockResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     make(http.Header),
	}
}

func mockJSONResponse(statusCode int, body string) *http.Response {
	resp := mockResponse(statusCode, body)
	resp.Header.Set("Content-Type", "application/json")
	return resp
}
