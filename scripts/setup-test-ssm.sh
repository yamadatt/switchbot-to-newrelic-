#!/bin/bash

# テスト用のSSMパラメータをセットアップするスクリプト
# sam local invoke実行前に、このスクリプトを実行してテスト用のパラメータを作成します

set -e

echo "Setting up test SSM parameters..."

# AWS_REGIONが設定されていない場合はデフォルトを使用
AWS_REGION=${AWS_REGION:-ap-northeast-1}

echo "Using AWS Region: $AWS_REGION"

# テスト用のSwitchBotトークン（ダミー値）
aws ssm put-parameter \
    --name "/test/switchbot/token" \
    --value "test-switchbot-token-12345" \
    --type "SecureString" \
    --region $AWS_REGION \
    --overwrite \
    --description "Test SwitchBot token for local development"

# テスト用のNew Relicライセンスキー（ダミー値）
aws ssm put-parameter \
    --name "/test/newrelic/license_key" \
    --value "test-newrelic-license-key-67890" \
    --type "SecureString" \
    --region $AWS_REGION \
    --overwrite \
    --description "Test New Relic license key for local development"

echo "Test SSM parameters have been created successfully!"
echo ""
echo "Created parameters:"
echo "  - /test/switchbot/token"
echo "  - /test/newrelic/license_key"
echo ""
echo "You can now run: sam local invoke --env-vars test/local-test-env.json"