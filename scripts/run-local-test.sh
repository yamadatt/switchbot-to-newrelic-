#!/bin/bash

# SAM local invokeを使用してローカルテストを実行するスクリプト

set -e

echo "Starting SAM local test..."

# カラーコードの定義
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 必要なファイルの存在確認
if [ ! -f "template.yaml" ]; then
    echo -e "${RED}Error: template.yaml not found${NC}"
    exit 1
fi

if [ ! -f "test/local-test-env.json" ]; then
    echo -e "${RED}Error: test/local-test-env.json not found${NC}"
    exit 1
fi

if [ ! -f "events/test-event.json" ]; then
    echo -e "${RED}Error: events/test-event.json not found${NC}"
    exit 1
fi

# AWS認証情報の確認
echo -e "${YELLOW}Checking AWS credentials...${NC}"
if ! aws sts get-caller-identity > /dev/null 2>&1; then
    echo -e "${RED}Error: AWS credentials not configured${NC}"
    echo "Please run 'aws configure' or set AWS environment variables"
    exit 1
fi

echo -e "${GREEN}AWS credentials OK${NC}"

# SAMビルドの実行
echo -e "${YELLOW}Building SAM application...${NC}"
if ! sam build; then
    echo -e "${RED}Error: SAM build failed${NC}"
    exit 1
fi

echo -e "${GREEN}SAM build completed${NC}"

# テスト用SSMパラメータのセットアップ確認
echo -e "${YELLOW}Checking test SSM parameters...${NC}"
AWS_REGION=${AWS_REGION:-ap-northeast-1}

if ! aws ssm get-parameter --name "/test/switchbot/token" --region $AWS_REGION > /dev/null 2>&1; then
    echo -e "${YELLOW}Test SSM parameters not found. Setting up...${NC}"
    bash scripts/setup-test-ssm.sh
fi

echo -e "${GREEN}Test SSM parameters OK${NC}"

# SAM local invokeの実行
echo -e "${YELLOW}Running SAM local invoke...${NC}"
echo "=========================================="

sam local invoke SwitchBotToNewRelicFunction \
    --event events/test-event.json \
    --env-vars test/local-test-env.json \
    --docker-network host

echo "=========================================="
echo -e "${GREEN}SAM local test completed${NC}"

# オプション: テスト後のクリーンアップ
read -p "Do you want to clean up test SSM parameters? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo -e "${YELLOW}Cleaning up test SSM parameters...${NC}"
    bash scripts/cleanup-test-ssm.sh
    echo -e "${GREEN}Cleanup completed${NC}"
fi