FROM node:20-alpine AS frontend-build
WORKDIR /src/frontend

COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci

COPY frontend ./
RUN npm run build

FROM golang:1.25-bookworm AS go-build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=frontend-build /src/frontend/build ./frontend/build

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/checkycheck .

FROM debian:bookworm-slim
WORKDIR /data

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates tzdata \
    && rm -rf /var/lib/apt/lists/*

COPY --from=go-build /out/checkycheck /usr/local/bin/checkycheck

EXPOSE 8070

CMD ["sh", "-c", "exec checkycheck -mode=serve -config=${CHECKYCHECK_CONFIG_DIR:-/config} -port=${CHECKYCHECK_PORT:-8070}"]
