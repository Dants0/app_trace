# Estágio 1: Build da aplicação Go
FROM golang:alpine AS builder

# Define o diretório de trabalho
WORKDIR /app

# Copia os arquivos de controle de dependência primeiro (para aproveitar o cache do Docker)
COPY go.mod go.sum ./
RUN go mod download

# Copia todo o código fonte para o container
COPY . .

# Compila o binário otimizado, sem dependências de C
RUN CGO_ENABLED=0 GOOS=linux go build -o app_trace main.go

# Estágio 2: Imagem final enxuta
FROM alpine:latest

WORKDIR /app

# Adiciona certificados SSL para garantir que as requisições HTTPS para as APIs de IA (OpenAI, Gemini) funcionem
RUN apk --no-cache add ca-certificates

# Copia o binário compilado do estágio anterior
COPY --from=builder /app/app_trace .

# Copia os arquivos estáticos do frontend para que fiquem disponíveis no container
COPY index.html styles.css favicon.ico ./

# Expõe a porta definida no seu .env
EXPOSE 8070

# Comando para iniciar o servidor
CMD ["./app_trace"]