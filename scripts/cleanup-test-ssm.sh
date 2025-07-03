#!/bin/bash

# テスト用のSSMパラメータをクリーンアップするスクリプト

set -e

echo "Cleaning up test SSM parameters..."

# AWS_REGIONが設定されていない場合はデフォルトを使用
AWS_REGION=${AWS_REGION:-ap-northeast-1}

echo "Using AWS Region: $AWS_REGION"

# テスト用パラメータを削除
aws ssm delete-parameter \
    --name "/test/switchbot/token" \
    --region $AWS_REGION || echo "Parameter /test/switchbot/token not found or already deleted"

aws ssm delete-parameter \
    --name "/test/newrelic/license_key" \
    --region $AWS_REGION || echo "Parameter /test/newrelic/license_key not found or already deleted"

echo "Test SSM parameters have been cleaned up!"