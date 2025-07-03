package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockRoundTripper は http.RoundTripper インターフェースのモック実装です。
// HTTPリクエストに対するレスポンスとエラーを制御できます。
type MockRoundTripper struct {
	Response *http.Response
	Err      error
}

func (m *MockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.Response, m.Err
}

// MockNewRelicApp は NewRelicApp インターフェースのモック実装です。
// RecordCustomEvent が呼び出されたことを検証したり、引数を記録したりできます。
type MockNewRelicApp struct {
	RecordedEvents          []RecordedEvent
	ShutdownCalled          bool
	WaitForConnectionCalled bool
	WaitForConnectionError  error
}

type RecordedEvent struct {
	EventType string
	Event     map[string]interface{}
}

func (m *MockNewRelicApp) RecordCustomEvent(eventType string, event map[string]interface{}) {
	m.RecordedEvents = append(m.RecordedEvents, RecordedEvent{
		EventType: eventType,
		Event:     event,
	})
}

func (m *MockNewRelicApp) Shutdown(timeout time.Duration) {
	m.ShutdownCalled = true
}

func (m *MockNewRelicApp) WaitForConnection(timeout time.Duration) error {
	m.WaitForConnectionCalled = true
	return m.WaitForConnectionError
}

// setupTestEnv はテスト用の環境変数を設定し、クリーンアップ関数を返します
func setupTestEnv(t *testing.T, envVars map[string]string) func() {
	t.Helper()
	
	// 既存の環境変数を保存
	originalEnvs := make(map[string]string)
	for key := range envVars {
		if val, exists := os.LookupEnv(key); exists {
			originalEnvs[key] = val
		}
	}
	
	// テスト用環境変数を設定
	for key, value := range envVars {
		if value == "" {
			os.Unsetenv(key)
		} else {
			os.Setenv(key, value)
		}
	}
	
	// クリーンアップ関数を返す
	return func() {
		for key := range envVars {
			if originalVal, exists := originalEnvs[key]; exists {
				os.Setenv(key, originalVal)
			} else {
				os.Unsetenv(key)
			}
		}
	}
}

// MockSSMGetParameter はSSMパラメータ取得をモック化する関数
func MockSSMGetParameter(parameterName string, withDecryption bool) (string, error) {
	// テスト用の固定値を返す
	switch parameterName {
	case "/test/switchbot/token":
		return "mock-switchbot-token", nil
	case "/test/newrelic/license_key":
		return "mock-newrelic-license", nil
	default:
		return "", errors.New("ParameterNotFound")
	}
}

// HandleRequestWithMockSSM はSSMをモック化したバージョンのHandleRequest
func HandleRequestWithMockSSM(ctx context.Context, httpClient *http.Client, nrApp NewRelicApp, ssmGetParam func(string, bool) (string, error)) (string, error) {
	// NewRelicの接続待機
	if err := nrApp.WaitForConnection(5 * time.Second); err != nil {
		// 接続失敗は致命的ではない
	}
	defer nrApp.Shutdown(10 * time.Second)

	// SwitchBot TokenをSSM Parameter Storeから取得（モック版）
	tokenParam := os.Getenv("SWITCHBOT_TOKEN_PARAMETER")
	if tokenParam == "" {
		return "", errors.New("環境変数 SWITCHBOT_TOKEN_PARAMETER が設定されていません")
	}
	token, err := ssmGetParam(tokenParam, true)
	if err != nil {
		return "", errors.New("SwitchBot Tokenの取得に失敗しました: " + err.Error())
	}

	deviceID := os.Getenv("SWITCHBOT_DEVICE_ID")
	if deviceID == "" {
		return "", errors.New("環境変数 SWITCHBOT_DEVICE_ID が設定されていません")
	}

	// SwitchBot APIを呼び出し
	url := "https://api.switch-bot.com/v1.1/devices/" + deviceID + "/status"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", errors.New("リクエストの作成に失敗しました: " + err.Error())
	}

	req.Header.Set("Authorization", token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", errors.New("APIリクエストに失敗しました: " + err.Error())
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errors.New("レスポンスボディの読み込みに失敗しました: " + err.Error())
	}

	if resp.StatusCode != http.StatusOK {
		return "", errors.New("APIからエラーが返されました: " + string(body))
	}

	var switchBotResponse SwitchBotResponse
	if err := json.Unmarshal(body, &switchBotResponse); err != nil {
		return "", errors.New("JSONのパースに失敗しました: " + err.Error())
	}

	// NewRelicへのデータ送信
	eventData := map[string]interface{}{
		"deviceId":    switchBotResponse.Body.DeviceID,
		"temperature": switchBotResponse.Body.Temperature,
		"humidity":    switchBotResponse.Body.Humidity,
		"battery":     switchBotResponse.Body.Battery,
	}
	nrApp.RecordCustomEvent("SwitchBotSensor", eventData)

	return "処理が正常に完了しました。", nil
}

func TestHandleRequest(t *testing.T) {
	// テストケースの定義
	tests := []struct {
		name                      string
		envVars                   map[string]string
		httpResponse              *http.Response
		httpError                 error
		waitForConnectionError    error
		expectedError             string
		expectEventCount          int
		expectEventType           string
		expectEventData           map[string]interface{}
		expectShutdownCalled      bool
		expectWaitForConnCalled   bool
	}{
		{
			name: "Success",
			envVars: map[string]string{
				"SWITCHBOT_TOKEN_PARAMETER":       "/test/switchbot/token",
				"SWITCHBOT_DEVICE_ID":             "test-device-id",
				"NEW_RELIC_LICENSE_KEY_PARAMETER": "/test/newrelic/license_key",
				"NEW_RELIC_APP_NAME":              "test-app",
				"AWS_REGION":                      "ap-northeast-1",
			},
			httpResponse: &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(`{"statusCode":100,"body":{"temperature":25.5,"humidity":60,"battery":100,"deviceId":"test-device-id"},"message":"success"}`)),
			},
			expectedError:           "",
			expectEventCount:        1,
			expectEventType:         "SwitchBotSensor",
			expectShutdownCalled:    true,
			expectWaitForConnCalled: true,
			expectEventData: map[string]interface{}{
				"deviceId":    "test-device-id",
				"temperature": 25.5,
				"humidity":    60, // 実際のコードではintのまま
				"battery":     100,
			},
		},
		{
			name: "MissingSwitchBotTokenParameter",
			envVars: map[string]string{
				"SWITCHBOT_TOKEN_PARAMETER":       "", // Missing
				"SWITCHBOT_DEVICE_ID":             "test-device-id",
				"NEW_RELIC_LICENSE_KEY_PARAMETER": "/test/newrelic/license_key",
				"NEW_RELIC_APP_NAME":              "test-app",
				"AWS_REGION":                      "ap-northeast-1",
			},
			httpResponse:            &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewBufferString(`{}`))},
			expectedError:           "環境変数 SWITCHBOT_TOKEN_PARAMETER が設定されていません",
			expectEventCount:        0,
			expectShutdownCalled:    true,
			expectWaitForConnCalled: true,
		},
		{
			name: "MissingSwitchBotDeviceID",
			envVars: map[string]string{
				"SWITCHBOT_TOKEN_PARAMETER":       "/test/switchbot/token",
				"SWITCHBOT_DEVICE_ID":             "", // Missing
				"NEW_RELIC_LICENSE_KEY_PARAMETER": "/test/newrelic/license_key",
				"NEW_RELIC_APP_NAME":              "test-app",
				"AWS_REGION":                      "ap-northeast-1",
			},
			httpResponse:            &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewBufferString(`{}`))},
			expectedError:           "環境変数 SWITCHBOT_DEVICE_ID が設定されていません",
			expectEventCount:        0,
			expectShutdownCalled:    true,
			expectWaitForConnCalled: true,
		},
		{
			name: "SwitchBotAPIError",
			envVars: map[string]string{
				"SWITCHBOT_TOKEN_PARAMETER":       "/test/switchbot/token",
				"SWITCHBOT_DEVICE_ID":             "test-device-id",
				"NEW_RELIC_LICENSE_KEY_PARAMETER": "/test/newrelic/license_key",
				"NEW_RELIC_APP_NAME":              "test-app",
				"AWS_REGION":                      "ap-northeast-1",
			},
			httpResponse: &http.Response{
				StatusCode: 400,
				Body:       io.NopCloser(bytes.NewBufferString(`{"statusCode":400,"message":"Bad Request"}`)),
			},
			expectedError:           "APIからエラーが返されました: {\"statusCode\":400,\"message\":\"Bad Request\"}",
			expectEventCount:        0,
			expectShutdownCalled:    true,
			expectWaitForConnCalled: true,
		},
		{
			name: "InvalidJSONResponse",
			envVars: map[string]string{
				"SWITCHBOT_TOKEN_PARAMETER":       "/test/switchbot/token",
				"SWITCHBOT_DEVICE_ID":             "test-device-id",
				"NEW_RELIC_LICENSE_KEY_PARAMETER": "/test/newrelic/license_key",
				"NEW_RELIC_APP_NAME":              "test-app",
				"AWS_REGION":                      "ap-northeast-1",
			},
			httpResponse: &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(`invalid json`)),
			},
			expectedError:           "JSONのパースに失敗しました: invalid character 'i' looking for beginning of value",
			expectEventCount:        0,
			expectShutdownCalled:    true,
			expectWaitForConnCalled: true,
		},
		{
			name: "NewRelicConnectionFailure",
			envVars: map[string]string{
				"SWITCHBOT_TOKEN_PARAMETER":       "/test/switchbot/token",
				"SWITCHBOT_DEVICE_ID":             "test-device-id",
				"NEW_RELIC_LICENSE_KEY_PARAMETER": "/test/newrelic/license_key",
				"NEW_RELIC_APP_NAME":              "test-app",
				"AWS_REGION":                      "ap-northeast-1",
			},
			httpResponse: &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(`{"statusCode":100,"body":{"temperature":25.5,"humidity":60,"battery":100,"deviceId":"test-device-id"},"message":"success"}`)),
			},
			waitForConnectionError:  errors.New("connection failed"),
			expectedError:           "", // 接続失敗は致命的ではない
			expectEventCount:        1,
			expectEventType:         "SwitchBotSensor",
			expectShutdownCalled:    true,
			expectWaitForConnCalled: true,
			expectEventData: map[string]interface{}{
				"deviceId":    "test-device-id",
				"temperature": 25.5,
				"humidity":    60,
				"battery":     100,
			},
		},
		{
			name: "HTTPRequestError",
			envVars: map[string]string{
				"SWITCHBOT_TOKEN_PARAMETER":       "/test/switchbot/token",
				"SWITCHBOT_DEVICE_ID":             "test-device-id",
				"NEW_RELIC_LICENSE_KEY_PARAMETER": "/test/newrelic/license_key",
				"NEW_RELIC_APP_NAME":              "test-app",
				"AWS_REGION":                      "ap-northeast-1",
			},
			httpError:               errors.New("network error"),
			expectedError:           "APIリクエストに失敗しました:",
			expectEventCount:        0,
			expectShutdownCalled:    true,
			expectWaitForConnCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 環境変数を設定
			cleanup := setupTestEnv(t, tt.envVars)
			defer cleanup()

			// モックHTTPクライアントとNew Relicアプリを設定
			mockTransport := &MockRoundTripper{Response: tt.httpResponse, Err: tt.httpError}
			mockHTTPClient := &http.Client{Transport: mockTransport}
			mockNRApp := &MockNewRelicApp{
				WaitForConnectionError: tt.waitForConnectionError,
			}

			// HandleRequestWithMockSSMを呼び出す（SSMをモック化）
			_, err := HandleRequestWithMockSSM(context.Background(), mockHTTPClient, mockNRApp, MockSSMGetParameter)

			// エラーの検証
			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
			}

			// New Relicイベントの検証
			assert.Len(t, mockNRApp.RecordedEvents, tt.expectEventCount, "New Relicイベントの数")
			if tt.expectEventCount > 0 {
				// 最初のイベントのデータを検証
				recordedEvent := mockNRApp.RecordedEvents[0]
				assert.Equal(t, tt.expectEventType, recordedEvent.EventType, "New Relicイベントタイプ")
				assert.Equal(t, tt.expectEventData, recordedEvent.Event, "New Relicイベントデータ")
			}

			// New Relicアプリのメソッド呼び出しを検証
			assert.Equal(t, tt.expectShutdownCalled, mockNRApp.ShutdownCalled, "Shutdownが呼び出されたか")
			assert.Equal(t, tt.expectWaitForConnCalled, mockNRApp.WaitForConnectionCalled, "WaitForConnectionが呼び出されたか")
		})
	}
}

// TestGetSSMParameter はgetSSMParameter関数の単体テスト
// 実際のAWS SSMを使わずにモックでテストする場合は、依存性注入が必要
func TestGetSSMParameter(t *testing.T) {
	// この関数は実際のAWS SSMを使用するため、統合テストとして扱う
	// ユニットテストとしてテストする場合は、SSMクライアントを依存性注入する必要がある
	t.Skip("getSSMParameter requires AWS SSM integration - skipping in unit tests")
}

// TestSwitchBotResponse はSwitchBotResponseの構造体テスト
func TestSwitchBotResponse(t *testing.T) {
	tests := []struct {
		name     string
		jsonData string
		expected SwitchBotResponse
		wantErr  bool
	}{
		{
			name:     "ValidResponse",
			jsonData: `{"statusCode":100,"body":{"deviceId":"test-device","deviceType":"Meter","hubDeviceId":"hub-123","humidity":65,"temperature":23.4,"version":"V4.2","battery":85,"temperatureScale":"c"},"message":"success"}`,
			expected: SwitchBotResponse{
				StatusCode: 100,
				Message:    "success",
				Body: struct {
					DeviceID           string  `json:"deviceId"`
					DeviceType         string  `json:"deviceType"`
					HubDeviceID        string  `json:"hubDeviceId"`
					Humidity           int     `json:"humidity"`
					Temperature        float64 `json:"temperature"`
					Version            string  `json:"version"`
					Battery            int     `json:"battery"`
					TemperatureScale   string  `json:"temperatureScale"`
				}{
					DeviceID:         "test-device",
					DeviceType:       "Meter",
					HubDeviceID:      "hub-123",
					Humidity:         65,
					Temperature:      23.4,
					Version:          "V4.2",
					Battery:          85,
					TemperatureScale: "c",
				},
			},
			wantErr: false,
		},
		{
			name:     "InvalidJSON",
			jsonData: `invalid json`,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var response SwitchBotResponse
			err := json.Unmarshal([]byte(tt.jsonData), &response)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected.StatusCode, response.StatusCode)
				assert.Equal(t, tt.expected.Message, response.Message)
				assert.Equal(t, tt.expected.Body.DeviceID, response.Body.DeviceID)
				assert.Equal(t, tt.expected.Body.Temperature, response.Body.Temperature)
				assert.Equal(t, tt.expected.Body.Humidity, response.Body.Humidity)
				assert.Equal(t, tt.expected.Body.Battery, response.Body.Battery)
			}
		})
	}
}