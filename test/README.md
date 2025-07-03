# SAM Local Testing Guide

このディレクトリには、`sam local invoke`を使用してLambda関数をローカルでテストするためのファイルが含まれています。

## ファイル構成

```
test/
├── README.md                    # このファイル
├── local-test-env.json         # ローカルテスト用環境変数
├── integration-test-env.json   # 統合テスト用環境変数（自動生成）
├── mock-server.go              # SwitchBot/New Relic APIモックサーバー
├── Dockerfile.mock             # モックサーバー用Dockerfile
└── docker-compose.test.yml     # テスト環境用Docker Compose

events/
└── test-event.json             # テスト用イベントファイル

scripts/
├── setup-test-ssm.sh          # テスト用SSMパラメータセットアップ
├── cleanup-test-ssm.sh        # テスト用SSMパラメータクリーンアップ
├── run-local-test.sh           # 基本的なローカルテスト実行
└── run-integration-test.sh     # モックサーバーを使った統合テスト実行
```

## 前提条件

- AWS CLI（設定済み）
- SAM CLI
- Docker
- Go 1.22+

## テスト方法

### 1. 基本的なローカルテスト

最もシンプルなテスト方法です。実際のSwitchBot/New Relic APIを呼び出します。

```bash
# テスト実行
./scripts/run-local-test.sh
```

このスクリプトは以下を自動で行います：
- AWS認証情報の確認
- SAMアプリケーションのビルド
- テスト用SSMパラメータの作成
- `sam local invoke`の実行
- オプションでテスト用パラメータのクリーンアップ

### 2. 統合テスト（推奨）

モックサーバーを使用して、外部APIに依存しないテストを実行します。

```bash
# 統合テスト実行
./scripts/run-integration-test.sh
```

このスクリプトは以下を自動で行います：
- モックサーバーのビルドと起動
- テスト用環境の設定
- SAMアプリケーションのビルド
- `sam local invoke`の実行
- 自動クリーンアップ

### 3. 手動テスト

より細かい制御が必要な場合は、手動でテストを実行できます。

```bash
# 1. テスト用SSMパラメータの作成
./scripts/setup-test-ssm.sh

# 2. SAMビルド
sam build

# 3. Lambda関数の実行
sam local invoke SwitchBotToNewRelicFunction \
    --event events/test-event.json \
    --env-vars test/local-test-env.json

# 4. クリーンアップ
./scripts/cleanup-test-ssm.sh
```

### 4. Docker Composeを使用したテスト

完全に分離されたテスト環境を構築する場合：

```bash
# テスト環境の起動
cd test
docker-compose -f docker-compose.test.yml up -d

# ヘルスチェック
curl http://localhost:8080/health

# テスト環境の停止
docker-compose -f docker-compose.test.yml down
```

## 環境変数の設定

### local-test-env.json

基本的なローカルテスト用の環境変数設定：

```json
{
  "SwitchBotToNewRelicFunction": {
    "SWITCHBOT_TOKEN_PARAMETER": "/test/switchbot/token",
    "NEW_RELIC_LICENSE_KEY_PARAMETER": "/test/newrelic/license_key",
    "SWITCHBOT_DEVICE_ID": "test-device-123",
    "NEW_RELIC_APP_NAME": "switchbot-to-newrelic-test",
    "NEW_RELIC_ACCOUNT_ID": "1234567890",
    "AWS_REGION": "ap-northeast-1"
  }
}
```

### モックサーバーの設定

モックサーバーは以下の環境変数でレスポンスをカスタマイズできます：

- `MOCK_TEMPERATURE`: 温度値（デフォルト: 25.5）
- `MOCK_HUMIDITY`: 湿度値（デフォルト: 60）
- `MOCK_BATTERY`: バッテリー値（デフォルト: 100）
- `MOCK_PORT`: ポート番号（デフォルト: 8080）

## トラブルシューティング

### よくある問題

1. **AWS認証エラー**
   ```bash
   aws configure
   # または
   export AWS_ACCESS_KEY_ID=your_key
   export AWS_SECRET_ACCESS_KEY=your_secret
   export AWS_REGION=ap-northeast-1
   ```

2. **Dockerネットワークエラー**
   ```bash
   # Dockerデーモンが起動していることを確認
   docker ps
   
   # SAM local invokeでhost.docker.internalが使用できない場合
   # docker-compose.test.ymlを使用してください
   ```

3. **SSMパラメータアクセスエラー**
   ```bash
   # IAM権限を確認
   aws ssm get-parameter --name "/test/switchbot/token" --with-decryption
   ```

4. **ポート競合**
   ```bash
   # 使用中のポートを確認
   lsof -i :8080
   
   # 別のポートを使用
   MOCK_PORT=8081 ./scripts/run-integration-test.sh
   ```

### ログの確認

- SAM local invokeのログは標準出力に表示されます
- モックサーバーのログも統合テストスクリプト実行時に表示されます
- より詳細なログが必要な場合は、`sam local invoke`に`--debug`オプションを追加してください

## カスタマイズ

### 新しいテストケースの追加

1. `events/`ディレクトリに新しいイベントファイルを作成
2. 必要に応じて環境変数ファイルを作成
3. テストスクリプトを修正または新規作成

### モックサーバーの拡張

`test/mock-server.go`を編集して、新しいエンドポイントやレスポンスパターンを追加できます。