package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strconv"
)

// SwitchBot APIのモックレスポンス
type MockSwitchBotResponse struct {
	StatusCode int    `json:"statusCode"`
	Message    string `json:"message"`
	Body       struct {
		DeviceID         string  `json:"deviceId"`
		DeviceType       string  `json:"deviceType"`
		HubDeviceID      string  `json:"hubDeviceId"`
		Humidity         int     `json:"humidity"`
		Temperature      float64 `json:"temperature"`
		Version          string  `json:"version"`
		Battery          int     `json:"battery"`
		TemperatureScale string  `json:"temperatureScale"`
	} `json:"body"`
}

func main() {
	port := os.Getenv("MOCK_PORT")
	if port == "" {
		port = "8080"
	}

	// SwitchBot APIのモック
	http.HandleFunc("/v1.1/devices/", func(w http.ResponseWriter, r *http.Request) {
		slog.Info("Received request", "method", r.Method, "path", r.URL.Path)
		slog.Info("Headers", "headers", r.Header)

		// 認証ヘッダーの確認
		auth := r.Header.Get("Authorization")
		if auth == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		// デバイスIDを取得（URLパスから）
		deviceID := "test-device-123" // デフォルト値

		// 環境変数でレスポンスをカスタマイズ可能
		temperature := 25.5
		humidity := 60
		battery := 100

		if tempStr := os.Getenv("MOCK_TEMPERATURE"); tempStr != "" {
			if temp, err := strconv.ParseFloat(tempStr, 64); err == nil {
				temperature = temp
			}
		}

		if humStr := os.Getenv("MOCK_HUMIDITY"); humStr != "" {
			if hum, err := strconv.Atoi(humStr); err == nil {
				humidity = hum
			}
		}

		if batStr := os.Getenv("MOCK_BATTERY"); batStr != "" {
			if bat, err := strconv.Atoi(batStr); err == nil {
				battery = bat
			}
		}

		response := MockSwitchBotResponse{
			StatusCode: 100,
			Message:    "success",
		}
		response.Body.DeviceID = deviceID
		response.Body.DeviceType = "Meter"
		response.Body.HubDeviceID = "hub-123"
		response.Body.Humidity = humidity
		response.Body.Temperature = temperature
		response.Body.Version = "V4.2"
		response.Body.Battery = battery
		response.Body.TemperatureScale = "c"

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			slog.Error("Encode error", "error", err)
		}
		slog.Info("Sent response", "response", response)
	})

	// New Relic APIのモック（イベント送信確認用）
	http.HandleFunc("/v1/accounts/", func(w http.ResponseWriter, r *http.Request) {
		slog.Info("New Relic mock received", "method", r.Method, "path", r.URL.Path)
		slog.Info("Headers", "headers", r.Header)

		// リクエストボディをログ出力
		if r.Body != nil {
			body := make([]byte, r.ContentLength)
			if _, err := r.Body.Read(body); err != nil {
				slog.Error("Body read error", "error", err)
			}
			slog.Info("Request body", "body", string(body))
		}

		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"success": true}`)); err != nil {
			slog.Error("Write error", "error", err)
		}
	})

	// ヘルスチェック用エンドポイント
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			slog.Error("Write error", "error", err)
		}
	})

	slog.Info("Mock server starting", "port", port)
	slog.Info("SwitchBot API mock", "url", "http://localhost:"+port+"/v1.1/devices/{deviceId}/status")
	slog.Info("New Relic API mock", "url", "http://localhost:"+port+"/v1/accounts/{accountId}/events")
	slog.Info("Health check", "url", "http://localhost:"+port+"/health")

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		slog.Error("Server failed to start", "error", err)
		os.Exit(1)
	}
}
