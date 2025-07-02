package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/newrelic/go-agent/v3/newrelic"
)

// SwitchBot APIのレスポンス構造体
type SwitchBotResponse struct {
	StatusCode int    `json:"statusCode"`
	Message    string `json:"message"`
	Body       struct {
		DeviceID           string  `json:"deviceId"`
		DeviceType         string  `json:"deviceType"`		
		HubDeviceID        string  `json:"hubDeviceId"`
		Humidity           int     `json:"humidity"`
		Temperature        float64 `json:"temperature"`
		Version            string  `json:"version"`
		Battery            int     `json:"battery"`
		TemperatureScale   string  `json:"temperatureScale"`
	} `json:"body"`
}

// NewRelicApp は newrelic.Application の必要なメソッドを定義するインターフェースです。
// これにより、テスト時にモックを注入できます。
type NewRelicApp interface {
	RecordCustomEvent(eventType string, event map[string]interface{})
	Shutdown(timeout time.Duration)
	WaitForConnection(timeout time.Duration) error // ここを修正
}

// HandleRequest は Lambdaハンドラの実際のロジックを含みます。
// 依存オブジェクト (httpClient, nrApp) を引数として受け取るように変更しました。
func HandleRequest(ctx context.Context, httpClient *http.Client, nrApp NewRelicApp) (string, error) {
	log.Println("Lambda関数を開始します...")

	// --- NewRelicの初期化 (nrAppを直接使用) ---
	// 環境変数はHandleRequestの外部で設定されることを想定
	appName := os.Getenv("NEW_RELIC_APP_NAME")
	if appName == "" {
		log.Fatal("環境変数 NEW_RELIC_APP_NAME が設定されていません。")
	}
	licenseKey := os.Getenv("NEW_RELIC_LICENSE_KEY")
	if licenseKey == "" {
		log.Fatal("環境変数 NEW_RELIC_LICENSE_KEY が設定されていません。")
	}

	log.Println("NewRelicへの接続を待機中...")
	if err := nrApp.WaitForConnection(5 * time.Second); err != nil {
		log.Printf("NewRelicへの接続に失敗しました: %v", err)
		// 接続失敗は致命的ではない場合もあるので、ここではログに留める
	}
	log.Println("NewRelicへの接続が完了しました。")

	// アプリケーションが終了する前にデータを送信するのを待つ
	defer nrApp.Shutdown(10 * time.Second)

	// --- SwitchBot APIからのデータ取得 ---
	token := os.Getenv("SWITCHBOT_TOKEN")
	if token == "" {
		return "", fmt.Errorf("環境変数 SWITCHBOT_TOKEN が設定されていません")
	}
	deviceID := os.Getenv("SWITCHBOT_DEVICE_ID")
	if deviceID == "" {
		return "", fmt.Errorf("環境変数 SWITCHBOT_DEVICE_ID が設定されていません")
	}

	log.Println("SwitchBot APIを呼び出します...")
	url := fmt.Sprintf("https://api.switch-bot.com/v1.1/devices/%s/status", deviceID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("リクエストの作成に失敗しました: %w", err)
	}

	req.Header.Set("Authorization", token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("APIリクエストに失敗しました: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("レスポンスボディの読み込みに失敗しました: %w", err)
	}

	log.Printf("SwitchBot APIからのレスポンス: %s\n", string(body))

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("APIからエラーが返されました: %s", string(body))
	}

	var switchBotResponse SwitchBotResponse
	if err := json.Unmarshal(body, &switchBotResponse); err != nil {
		return "", fmt.Errorf("JSONのパースに失敗しました: %w", err)
	}

	log.Printf("パースされたデータ: %+v\n", switchBotResponse.Body)

	// --- NewRelicへのデータ送信 ---
	eventData := map[string]interface{}{
		"deviceId":    switchBotResponse.Body.DeviceID,
		"temperature": switchBotResponse.Body.Temperature,
		"humidity":    switchBotResponse.Body.Humidity,
		"battery":     switchBotResponse.Body.Battery,
	}
	log.Printf("NewRelicに送信するデータ: %+v\n", eventData)
	nrApp.RecordCustomEvent("SwitchBotSensor", eventData)

	log.Println("NewRelicへのデータ送信を要求しました。")

	return "処理が正常に完了しました。", nil
}

// main関数はLambdaの起動と依存オブジェクトの初期化を行います。
func main() {
	// New Relicアプリケーションの初期化
	appName := os.Getenv("NEW_RELIC_APP_NAME")
	licenseKey := os.Getenv("NEW_RELIC_LICENSE_KEY")

	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName(appName),
		newrelic.ConfigLicense(licenseKey),
		newrelic.ConfigDistributedTracerEnabled(true),
	)
	if err != nil {
		log.Fatalf("NewRelicアプリケーションの作成に失敗しました: %v", err)
	}

	// Lambdaハンドラを起動
	lambda.Start(func(ctx context.Context) (string, error) {
		return HandleRequest(ctx, &http.Client{}, app)
	})
}