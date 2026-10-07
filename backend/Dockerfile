FROM golang:1.26-alpine AS build
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/worker ./cmd/worker \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/manage ./cmd/manage

FROM alpine:3.23 AS runtime
RUN apk add --no-cache ca-certificates tzdata \
 && addgroup -S -g 10001 app \
 && adduser -S -u 10001 -G app app
WORKDIR /app
COPY --from=build /out/ /app/
COPY backend/migrations/ /app/migrations/
USER app
EXPOSE 8080
CMD ["/app/api"]
