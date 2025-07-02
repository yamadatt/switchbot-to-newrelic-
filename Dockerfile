# ビルドステージ
FROM golang:1.22 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

# アプリケーションをビルド
# CGO_ENABLED=0 は静的リンクバイナリを生成し、外部依存をなくします
# -o /app/bootstrap で出力先を指定します（provided:al2ランタイム用）
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bootstrap -ldflags="-s -w" main.go

# 実行ステージ
# AWS Lambda provided:al2ベースイメージを使用
FROM public.ecr.aws/lambda/provided:al2

# ビルドステージからコンパイル済みのバイナリをコピー
COPY --from=builder /app/bootstrap /var/runtime/

# 実行権限を付与
RUN chmod 755 /var/runtime/bootstrap

# Lambdaが実行するコマンドを設定
CMD ["bootstrap"]
