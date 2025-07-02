# SwitchBot to New Relic Exporter

このアプリケーションは、SwitchBot温湿度計からセンサーデータを定期的に取得し、New Relicにカスタムイベントとして送信するAWS Lambda関数です。

AWS SAM (Serverless Application Model) を使用して、コンテナイメージとしてパッケージ化され、デプロイされます。

## アーキテクチャ

- **アプリケーション**: Go言語で実装
- **実行環境**: AWS Lambda (コンテナイメージランタイム)
- **トリガー**: Amazon EventBridge (CloudWatch Events) により、15分ごとに定周期で実行
- **デプロイ**: AWS SAM CLI
- **設定管理**: AWS Systems Manager Parameter Store を使用して、APIキーなどの機密情報を安全に管理

## 前提条件

このアプリケーションをデプロイ・実行するには、以下のツールとアカウントが必要です。

- [AWS CLI](https://aws.amazon.com/cli/) (設定済みであること)
- [AWS SAM CLI](https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/serverless-sam-cli-install.html)
- [Docker](https://www.docker.com/products/docker-desktop) (実行中であること)
- Go言語 (v1.22以上, ローカルでの開発に必要)
- AWSアカウント
- SwitchBotアカウントとAPIトークン、および温湿度計のデバイスID
- New Relicアカウントとライセンスキー、アカウントID

## セットアップ

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

## デプロイ手順

1.  **SAMビルド**

    以下のコマンドを実行して、Goアプリケーションをビルドし、デプロイ用のコンテナイメージを作成します。

    ```bash
    sam build
    ```

2.  **SAMデプロイ (ガイド付き)**

    次に、ガイド付きデプロイを実行します。このプロセスでは、スタック名やデプロイ先のリージョン、そして先ほど設定したSSMパラメータの名前などを対話形式で設定できます。

    ```bash
    sam deploy --guided
    ```

    **ガイド付きデプロイで聞かれること:**

    - **Stack Name**: `switchbot-to-newrelic-app` のようなスタック名を入力します。
    - **AWS Region**: デプロイしたいリージョン（例: `ap-northeast-1`）を選択します。
    - **Parameter ...ParameterName**: SSMパラメータの名前を尋ねられます。`template.yaml` にデフォルト値が設定されているため、SSMパラメータ作成時に同じ名前を使っていれば、そのままEnterキーを押すだけで進められます。
    - **Confirm changes before deploy**: `y` (Yes) を選択すると、変更内容を確認できます。
    - **Allow SAM CLI IAM role creation**: `y` (Yes) を選択します。
    - **Save arguments to samconfig.toml**: `y` (Yes) を選択すると、今回の設定が `samconfig.toml` に保存され、次回以降のデプロイが `sam deploy` だけで実行できるようになります。

デプロイが完了すると、Lambda関数が作成され、15分ごとに自動で実行されるようになります。

## 動作確認

デプロイ後、アプリケーションが正しく動作しているかを確認します。

1.  **CloudWatch Logsの確認**

    [AWS Lambdaコンソール](https://console.aws.amazon.com/lambda/)にアクセスし、デプロイした関数を選択します。「モニタリング」タブからCloudWatch Logsに移動し、関数の実行ログを確認できます。

2.  **New Relicでのデータ確認**

    New Relicにログインし、**Query your data** に移動して以下のNRQLクエリを実行します。データが送信されていれば、`SwitchBotSensor` イベントが表示されます。（データの反映には数分かかる場合があります）

    ```sql
    SELECT * FROM SwitchBotSensor SINCE 30 minutes ago
    ```

## プロジェクト構成

- `main.go`: Lambdaハンドラを実装したGoのソースコード
- `Dockerfile`: Lambdaで実行するコンテナイメージを定義
- `template.yaml`: AWSリソース（Lambda関数、実行トリガーなど）を定義したSAMテンプレート
- `.gitignore`: Gitの追跡から不要なファイルを除外
- `README.md`: このファイル
