# Dockerfile para o worker de processamento de vídeo
FROM golang:1.24.5-alpine AS builder

WORKDIR /app
COPY . .

# Baixar dependências
RUN go mod download

# Compilar o binário
RUN go build -o video-processor main.go

# Imagem final
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/video-processor .

# Instalar ffmpeg
RUN apk add --no-cache ffmpeg

# Comando padrão
CMD ["./video-processor"]
