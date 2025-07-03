# SwitchBot to New Relic Exporter

このアプリケーションは、SwitchBot温湿度計からセンサーデータを定期的に取得し、New Relicにカスタムイベントとして送信するAWS Lambda関数です。

AWS SAM (Serverless Application Model) を使用して、コンテナイメージとしてパッケージ化され、デプロイされます。

## アーキテクチャ

- **アプリケーション**: Go言語で実装
- **実行環境**: AWS Lambda (コンテナイメージランタイム)
- **トリガー**: Amazon EventBridge (CloudWatch Events) により、15分ごとに定周期で実行
- **デプロイ**: AWS SAM CLI
- **設定管理**: AWS Systems Manager Parameter Store を使用して、APIキーなどの機密情報を安全に管理
- **開発環境**: `.env`ファイルを使用したローカル環境変数管理

## 前提条件

このアプリケーションをデプロイ・実行するには、以下のツールとアカウントが必要です。

### 必須ツール
- [AWS CLI](https://aws.amazon.com/cli/) (設定済みであること)
- [AWS SAM CLI](https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/serverless-sam-cli-install.html)
- [Docker](https://www.docker.com/products/docker-desktop) (実行中であること)
- Go言語 (v1.22以上, ローカルでの開発に必要)

### アカウント・認証情報
- AWSアカウント
- SwitchBotアカウントとAPIトークン、および温湿度計のデバイスID
- New Relicアカウントとライセンスキー、アカウントID

### 開発用ツール（オプション）
- [golangci-lint](https://golangci-lint.run/) (コード品質チェック用)
- [gosec](https://github.com/securecodewarrior/gosec) (セキュリティスキャン用)
- [govulncheck](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck) (脆弱性チェック用)

## セットアップ

### 1. 環境変数の設定

まず、ローカル開発用の環境変数を設定します：

```bash
# .envファイルをコピーして編集
cp .env .env.local
# .env.localファイルを編集して、実際の値を設定してください
```

`.env`ファイルの例：
```bash
# AWS Configuration
AWS_REGION=ap-northeast-1

# SwitchBot Configuration
SWITCHBOT_TOKEN_PARAMETER=/switchbot/token
SWITCHBOT_DEVICE_ID=your-device-id-here

# New Relic Configuration
NEW_RELIC_LICENSE_KEY_PARAMETER=/newrelic/license_key
NEW_RELIC_APP_NAME=switchbot-to-newrelic
NEW_RELIC_ACCOUNT_ID=your-account-id-here
```

### 2. AWS SSM Parameter Storeの設定

デプロイの前に、アプリケーションが必要とする設定値や機密情報をAWS Systems Manager (SSM) Parameter Storeに保存する必要があります。

以下のコマンド例を参考に、ご自身の値でパラメータを作成してください。機密情報であるトークンとキーは `SecureString` タイプで保存することを強く推奨します。

```bash
# AWSリージョンを設定 (例: ap-northeast-1)
export AWS_REGION=ap-northeast-1

# SwitchBotのトークン
aws ssm put-parameter --name "/switchbot/token" --value "YOUR_SWITCHBOT_TOKEN" --type "SecureString" --region $AWS_REGION

# SwitchBotのデバイスID
aws ssm put-parameter --name "/switchbot/device_id" --value "YOUR_DEVICE_ID" --type "String" --region $AWS_REGION

# New Relicのライセンスキー
aws ssm put-parameter --name "/newrelic/license_key" --value "YOUR_NEW_RELIC_LICENSE_KEY" --type "SecureString" --region $AWS_REGION

# New RelicのアカウントID
aws ssm put-parameter --name "/newrelic/account_id" --value "YOUR_NEW_RELIC_ACCOUNT_ID" --type "String" --region $AWS_REGION

# New Relicのアプリケーション名
aws ssm put-parameter --name "/newrelic/app_name" --value "switchbot-to-newrelic" --type "String" --region $AWS_REGION
```

または、Makefileを使用して簡単にセットアップできます：

```bash
# 開発環境のセットアップ（依存関係のダウンロード + テスト用SSMパラメータの作成）
make dev-setup
```

## デプロイ手順

### 本番環境へのデプロイ

1.  **事前チェック**

    デプロイ前に全てのチェックを実行します：

    ```bash
    make pre-deploy
    ```

    このコマンドは以下を実行します：
    - 依存関係のダウンロード
    - コードフォーマット
    - 静的解析（vet）
    - ユニットテスト
    - セキュリティスキャン
    - ビルド

2.  **初回デプロイ (ガイド付き)**

    初回デプロイ時は、ガイド付きデプロイを実行します：

    ```bash
    make sam-deploy-guided
    ```

    **ガイド付きデプロイで聞かれること:**

    - **Stack Name**: `switchbot-to-newrelic` のようなスタック名を入力します。
    - **AWS Region**: デプロイしたいリージョン（例: `ap-northeast-1`）を選択します。
    - **Parameter ...ParameterName**: SSMパラメータの名前を尋ねられます。`template.yaml` にデフォルト値が設定されているため、SSMパラメータ作成時に同じ名前を使っていれば、そのままEnterキーを押すだけで進められます。
    - **Confirm changes before deploy**: `y` (Yes) を選択すると、変更内容を確認できます。
    - **Allow SAM CLI IAM role creation**: `y` (Yes) を選択します。
    - **Save arguments to samconfig.toml**: `y` (Yes) を選択すると、今回の設定が `samconfig.toml` に保存され、次回以降のデプロイが `sam deploy` だけで実行できるようになります。

3.  **2回目以降のデプロイ**

    設定が保存された後は、簡単にデプロイできます：

    ```bash
    make sam-deploy
    ```

デプロイが完了すると、Lambda関数が作成され、15分ごとに自動で実行されるようになります。

## 開発・テスト

### ローカル開発環境のセットアップ

```bash
# 開発環境の初期セットアップ
make dev-setup

# 依存関係の更新
make deps-update
```

### テストの実行

```bash
# 全てのテストを実行
make test

# ユニットテストのみ
make test-unit

# ローカルでのLambda関数テスト
make test-local

# 統合テスト（モックサーバーを使用）
make test-integration

# 開発用テストスイート（フォーマット + 静的解析 + ユニットテスト + 統合テスト）
make dev-test
```

### コード品質チェック

```bash
# コードフォーマット
make fmt

# 静的解析
make vet

# Linter実行
make lint

# セキュリティスキャン
make security-scan

# カバレッジレポート生成
make coverage
```

### ローカルでのAPI Gateway起動

```bash
# ローカルでAPI Gatewayを起動（テスト用）
make sam-local-api
```

## 動作確認

デプロイ後、アプリケーションが正しく動作しているかを確認します。

### 1. CloudWatch Logsの確認

[AWS Lambdaコンソール](https://console.aws.amazon.com/lambda/)にアクセスし、デプロイした関数を選択します。「モニタリング」タブからCloudWatch Logsに移動し、関数の実行ログを確認できます。

または、SAM CLIを使用してログを確認：

```bash
make sam-logs
```

### 2. New Relicでのデータ確認

New Relicにログインし、**Query your data** に移動して以下のNRQLクエリを実行します。データが送信されていれば、`SwitchBotSensor` イベントが表示されます。（データの反映には数分かかる場合があります）

```sql
SELECT * FROM SwitchBotSensor SINCE 30 minutes ago
```

## プロジェクト構成

```
.
├── main.go                     # Lambdaハンドラを実装したGoのソースコード
├── main_test.go               # ユニットテスト
├── Dockerfile                 # Lambdaで実行するコンテナイメージを定義
├── template.yaml              # AWSリソース（Lambda関数、実行トリガーなど）を定義したSAMテンプレート
├── .env                       # 環境変数設定ファイル（ローカル開発用）
├── .gitignore                 # Gitの追跡から不要なファイルを除外
├── Makefile                   # 開発・ビルド・テスト用のタスク定義
├── samconfig.toml             # SAMデプロイ設定
├── go.mod                     # Go モジュール定義
├── go.sum                     # Go 依存関係のチェックサム
├── README.md                  # このファイル
├── .github/
│   └── workflows/             # GitHub Actions CI/CDワークフロー
│       ├── ci.yml             # 継続的インテグレーション
│       ├── sam-deploy.yml     # デプロイワークフロー
│       └── sam-pr-check.yml   # プルリクエストチェック
├── events/
│   └── test-event.json        # テスト用イベントデータ
├── scripts/                   # 開発・テスト用スクリプト
│   ├── cleanup-test-ssm.sh    # テスト用SSMパラメータのクリーンアップ
│   ├── run-integration-test.sh # 統合テスト実行
│   ├── run-local-test.sh      # ローカルテスト実行
│   └── setup-test-ssm.sh      # テスト用SSMパラメータのセットアップ
└── test/                      # テスト関連ファイル
    ├── docker-compose.test.yml # テスト用Docker Compose設定
    ├── Dockerfile.mock        # モックサーバー用Dockerfile
    ├── local-test-env.json    # ローカルテスト用環境変数
    ├── mock-server.go         # SwitchBot API モックサーバー
    └── README.md              # テスト環境の説明
```

## 利用可能なMakeタスク

プロジェクトには開発を効率化するためのMakeタスクが用意されています：

```bash
make help                 # 利用可能なタスクの一覧表示
make build               # SAMアプリケーションのビルド
make test                # 全てのテストを実行
make test-unit           # ユニットテストのみ実行
make test-local          # ローカルでのLambda関数テスト
make test-integration    # 統合テスト実行
make dev-setup           # 開発環境のセットアップ
make dev-test            # 開発用テストスイート実行
make pre-deploy          # デプロイ前チェック実行
make sam-deploy-guided   # ガイド付きデプロイ
make sam-deploy          # 設定済みデプロイ
make sam-logs            # Lambda関数のログ表示
make clean               # ビルド成果物のクリーンアップ
make fmt                 # コードフォーマット
make lint                # Linter実行
make vet                 # 静的解析実行
make security-scan       # セキュリティスキャン実行
make coverage            # カバレッジレポート生成
```

## CI/CD

このプロジェクトはGitHub Actionsを使用したCI/CDパイプラインを提供しています：

- **継続的インテグレーション** (`.github/workflows/ci.yml`): プルリクエストやpushで自動テスト実行
- **デプロイワークフロー** (`.github/workflows/sam-deploy.yml`): 手動トリガーによる本番デプロイ
- **プルリクエストチェック** (`.github/workflows/sam-pr-check.yml`): PRでの軽量チェック
