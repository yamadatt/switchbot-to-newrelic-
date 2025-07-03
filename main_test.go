package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
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
	RecordedEvents          []map[string]interface{}
	ShutdownCalled          bool
	WaitForConnectionCalled bool
}

func (m *MockNewRelicApp) RecordCustomEvent(eventType string, event map[string]interface{}) {
	// イベントタイプはここでは検証しないが、必要であれば追加可能
	m.RecordedEvents = append(m.RecordedEvents, event)
}

func (m *MockNewRelicApp) Shutdown(timeout time.Duration) {
	m.ShutdownCalled = true
}

func (m *MockNewRelicApp) WaitForConnection(timeout time.Duration) error {
	m.WaitForConnectionCalled = true
	return nil // テストでは常に成功を返す
}

func TestHandleRequest(t *testing.T) {
	assert := require.New(t)

	// テスト実行前に環境変数を保存し、テスト後に復元するヘルパー関数
	setEnv := func(key, value string) {
		if err := os.Setenv(key, value); err != nil {
			t.Fatalf("os.Setenv(%s) failed: %v", key, err)
		}
	}
	unsetEnv := func(key string) {
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("os.Unsetenv(%s) failed: %v", key, err)
		}
	}

	// テストケースの定義
	tests := []struct {
		name                string
		switchBotToken      string
		switchBotDeviceID   string
		newRelicAppName     string
		newRelicLicenseKey  string
		switchBotTokenParam string
		httpResponse        *http.Response
		httpError           error
		expectedError       string
		expectEventCount    int
		expectEventData     map[string]interface{}
	}{
		{
			name:                "Success",
			switchBotToken:      "test-token",
			switchBotDeviceID:   "test-device-id",
			newRelicAppName:     "test-app",
			newRelicLicenseKey:  "test-license",
			switchBotTokenParam: "dummy-param",
			httpResponse: &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(`{"statusCode":100,"body":{"temperature":25.5,"humidity":60,"battery":100,"deviceId":"test-device-id"},"message":"success"}`)),
			},
			expectedError:    "",
			expectEventCount: 1,
			expectEventData: map[string]interface{}{
				"deviceId":    "test-device-id",
				"temperature": 25.5,
				"humidity":    60,
				"battery":     100,
			},
		},
		{
			name:                "MissingSwitchBotToken",
			switchBotToken:      "", // Missing
			switchBotDeviceID:   "test-device-id",
			newRelicAppName:     "test-app",
			newRelicLicenseKey:  "test-license",
			switchBotTokenParam: "dummy-param",
			httpResponse:        &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewBufferString(`{}`))},
			expectedError:       "SwitchBot Tokenの取得に失敗しました",
			expectEventCount:    0,
		},
		{
			name:                "MissingSwitchBotDeviceID",
			switchBotToken:      "test-token",
			switchBotDeviceID:   "", // Missing
			newRelicAppName:     "test-app",
			newRelicLicenseKey:  "test-license",
			switchBotTokenParam: "dummy-param",
			httpResponse:        &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewBufferString(`{}`))},
			expectedError:       "環境変数 SWITCHBOT_DEVICE_ID が設定されていません",
			expectEventCount:    0,
		},
		{
			name:                "SwitchBotAPIError",
			switchBotToken:      "test-token",
			switchBotDeviceID:   "test-device-id",
			newRelicAppName:     "test-app",
			newRelicLicenseKey:  "test-license",
			switchBotTokenParam: "dummy-param",
			httpResponse: &http.Response{
				StatusCode: 400,
				Body:       io.NopCloser(bytes.NewBufferString(`{"statusCode":400,"message":"Bad Request"}`)),
			},
			expectedError:    "APIからエラーが返されました: {\"statusCode\":400,\"message\":\"Bad Request\"}",
			expectEventCount: 0,
		},
		{
			name:                "InvalidJSONResponse",
			switchBotToken:      "test-token",
			switchBotDeviceID:   "test-device-id",
			newRelicAppName:     "test-app",
			newRelicLicenseKey:  "test-license",
			switchBotTokenParam: "dummy-param",
			httpResponse: &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(`invalid json`)),
			},
			expectedError:    "JSONのパースに失敗しました: invalid character 'i' looking for beginning of value",
			expectEventCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 環境変数を設定
			setEnv("SWITCHBOT_TOKEN_PARAMETER", tt.switchBotTokenParam)
			setEnv("SWITCHBOT_TOKEN", tt.switchBotToken)
			setEnv("SWITCHBOT_DEVICE_ID", tt.switchBotDeviceID)
			setEnv("NEW_RELIC_APP_NAME", tt.newRelicAppName)
			setEnv("NEW_RELIC_LICENSE_KEY", tt.newRelicLicenseKey)

			// モックHTTPクライアントとNew Relicアプリを設定
			mockTransport := &MockRoundTripper{Response: tt.httpResponse, Err: tt.httpError}
			mockHTTPClient := &http.Client{Transport: mockTransport}
			mockNRApp := &MockNewRelicApp{}

			// getSSMParameterのモック関数
			mockGetSSMParameter := func(parameterName string, withDecryption bool) (string, error) {
				if tt.switchBotToken == "" {
					return "", fmt.Errorf("SwitchBot Tokenの取得に失敗しました: パラメータが空です")
				}
				return tt.switchBotToken, nil
			}

			// HandleRequestを呼び出す
			_, err := HandleRequest(context.Background(), mockHTTPClient, mockNRApp, mockGetSSMParameter)

			// エラーの検証
			if tt.expectedError != "" {
				assert.ErrorContains(err, tt.expectedError)
			} else {
				assert.NoError(err)
			}

			// New Relicイベントの検証
			assert.Len(mockNRApp.RecordedEvents, tt.expectEventCount, "New Relicイベントの数")
			if tt.expectEventCount > 0 {
				// 最初のイベントのデータを検証
				recordedEvent := mockNRApp.RecordedEvents[0]
				if diff := cmp.Diff(tt.expectEventData, recordedEvent); diff != "" {
					t.Errorf("New Relicイベントデータに差異がありました (-want +got):\n%s", diff)
				}
			}

			// 環境変数をクリーンアップ
			unsetEnv("SWITCHBOT_TOKEN_PARAMETER")
			unsetEnv("SWITCHBOT_TOKEN")
			unsetEnv("SWITCHBOT_DEVICE_ID")
			unsetEnv("NEW_RELIC_APP_NAME")
			unsetEnv("NEW_RELIC_LICENSE_KEY")
		})
	}
}
