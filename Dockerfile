# Build stage
FROM golang:1.20-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go env -w GOPROXY=https://proxy.golang.org,direct
COPY . .
RUN go build -o /app ./cmd/server

# Final image
FROM alpine:3.18
RUN apk add --no-cache ca-certificates
COPY --from=build /app /app
EXPOSE 8080
ENV ADDR=":8080"
ENTRYPOINT ["/app"]
