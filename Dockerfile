FROM --platform=$BUILDPLATFORM node:22-alpine AS frontend
WORKDIR /src/web
COPY web/package*.json ./
RUN npm ci --no-audit --no-fund
COPY web/ ./
RUN npm run build

FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /src/web/dist ./web/dist
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags='-s -w' -o /wordweave ./cmd/wordweave

FROM alpine:3.22
WORKDIR /app
COPY --from=build /wordweave /app/wordweave
ENV WORDWEAVE_DATA_DIR=/data
EXPOSE 8080
ENTRYPOINT ["/app/wordweave", "-listen", "0.0.0.0:8080"]
