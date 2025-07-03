package main

import (
	"encoding/json"
	"log"
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
		log.Printf("Received request: %s %s", r.Method, r.URL.Path)
		log.Printf("Headers: %v", r.Header)

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
			log.Printf("Encode error: %v", err)
		}
		log.Printf("Sent response: %+v", response)
	})

	// New Relic APIのモック（イベント送信確認用）
	http.HandleFunc("/v1/accounts/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("New Relic mock received: %s %s", r.Method, r.URL.Path)
		log.Printf("Headers: %v", r.Header)

		// リクエストボディをログ出力
		if r.Body != nil {
			body := make([]byte, r.ContentLength)
			if _, err := r.Body.Read(body); err != nil {
				log.Printf("Body read error: %v", err)
			}
			log.Printf("Request body: %s", string(body))
		}

		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"success": true}`)); err != nil {
			log.Printf("Write error: %v", err)
		}
	})

	// ヘルスチェック用エンドポイント
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			log.Printf("Write error: %v", err)
		}
	})

	log.Printf("Mock server starting on port %s", port)
	log.Printf("SwitchBot API mock: http://localhost:%s/v1.1/devices/{deviceId}/status", port)
	log.Printf("New Relic API mock: http://localhost:%s/v1/accounts/{accountId}/events", port)
	log.Printf("Health check: http://localhost:%s/health", port)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
