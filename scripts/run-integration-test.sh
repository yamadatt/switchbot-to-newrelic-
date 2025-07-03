#!/bin/bash

# 統合テスト用スクリプト（モックサーバーを使用）

set -e

# カラーコードの定義
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}Starting Integration Test with Mock Server${NC}"
echo "=============================================="

# 設定
MOCK_PORT=8080
MOCK_PID=""

# クリーンアップ関数
cleanup() {
    echo -e "\n${YELLOW}Cleaning up...${NC}"
    if [ ! -z "$MOCK_PID" ]; then
        echo "Stopping mock server (PID: $MOCK_PID)"
        kill $MOCK_PID 2>/dev/null || true
        wait $MOCK_PID 2>/dev/null || true
    fi
    
    # テスト用SSMパラメータのクリーンアップ
    echo "Cleaning up test SSM parameters..."
    bash scripts/cleanup-test-ssm.sh 2>/dev/null || true
    
    echo -e "${GREEN}Cleanup completed${NC}"
}

# シグナルハンドラーの設定
trap cleanup EXIT INT TERM

# 必要なツールの確認
echo -e "${YELLOW}Checking prerequisites...${NC}"

if ! command -v go &> /dev/null; then
    echo -e "${RED}Error: Go is not installed${NC}"
    exit 1
fi

if ! command -v sam &> /dev/null; then
    echo -e "${RED}Error: SAM CLI is not installed${NC}"
    exit 1
fi

if ! aws sts get-caller-identity > /dev/null 2>&1; then
    echo -e "${RED}Error: AWS credentials not configured${NC}"
    exit 1
fi

echo -e "${GREEN}Prerequisites OK${NC}"

# モックサーバーのビルドと起動
echo -e "${YELLOW}Building and starting mock server...${NC}"
cd test
go build -o mock-server mock-server.go

# モックサーバーを環境変数付きで起動
MOCK_PORT=$MOCK_PORT \
MOCK_TEMPERATURE=23.5 \
MOCK_HUMIDITY=65 \
MOCK_BATTERY=85 \
./mock-server &

MOCK_PID=$!
cd ..

# モックサーバーの起動待機
echo "Waiting for mock server to start..."
for i in {1..10}; do
    if curl -s http://localhost:$MOCK_PORT/health > /dev/null 2>&1; then
        echo -e "${GREEN}Mock server is running (PID: $MOCK_PID)${NC}"
        break
    fi
    if [ $i -eq 10 ]; then
        echo -e "${RED}Error: Mock server failed to start${NC}"
        exit 1
    fi
    sleep 1
done

# テスト用環境変数ファイルの作成（モックサーバー用）
echo -e "${YELLOW}Creating test environment configuration...${NC}"
cat > test/integration-test-env.json << EOF
{
  "SwitchBotToNewRelicFunction": {
    "SWITCHBOT_TOKEN_PARAMETER": "/test/switchbot/token",
    "NEW_RELIC_LICENSE_KEY_PARAMETER": "/test/newrelic/license_key",
    "SWITCHBOT_DEVICE_ID": "test-device-123",
    "NEW_RELIC_APP_NAME": "switchbot-to-newrelic-integration-test",
    "NEW_RELIC_ACCOUNT_ID": "1234567890",
    "AWS_REGION": "ap-northeast-1",
    "SWITCHBOT_API_BASE_URL": "http://host.docker.internal:$MOCK_PORT",
    "NEW_RELIC_API_BASE_URL": "http://host.docker.internal:$MOCK_PORT"
  }
}
EOF

# テスト用SSMパラメータのセットアップ
echo -e "${YELLOW}Setting up test SSM parameters...${NC}"
bash scripts/setup-test-ssm.sh

# SAMビルド
echo -e "${YELLOW}Building SAM application...${NC}"
sam build

# 統合テストの実行
echo -e "${YELLOW}Running integration test...${NC}"
echo "=========================================="

# SAM local invokeを実行
sam local invoke SwitchBotToNewRelicFunction \
    --event events/test-event.json \
    --env-vars test/integration-test-env.json \
    --docker-network host

TEST_EXIT_CODE=$?

echo "=========================================="

if [ $TEST_EXIT_CODE -eq 0 ]; then
    echo -e "${GREEN}✅ Integration test PASSED${NC}"
else
    echo -e "${RED}❌ Integration test FAILED${NC}"
fi

# テスト結果の表示
echo -e "\n${BLUE}Test Summary:${NC}"
echo "- Mock server provided realistic API responses"
echo "- Lambda function processed the request"
echo "- Exit code: $TEST_EXIT_CODE"

exit $TEST_EXIT_CODE