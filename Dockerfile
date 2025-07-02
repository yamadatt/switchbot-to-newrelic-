# ビルドステージ
FROM golang:1.22-alpine as builder

WORKDIR /app

# Goモジュールの依存関係をコピーしてダウンロード
COPY go.mod go.sum ./
RUN go mod download

# ソースコードをコピー
COPY . .

# アプリケーションをビルド
# CGO_ENABLED=0 は静的リンクバイナリを生成し、外部依存をなくします
# -o /app/handler で出力先を指定します
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/handler -ldflags="-s -w" main.go

# 実行ステージ
# AWS Lambdaが提供するGo 1.x用のベースイメージを使用
FROM public.ecr.aws/lambda/provided:al2

WORKDIR /var/task

# ビルドステージからコンパイル済みのバイナリをコピー
COPY --from=builder /app/handler .

# Lambdaが実行するコマンドを設定
# CMD ["実行ファイル名"] の形式で指定します
CMD ["handler"]
