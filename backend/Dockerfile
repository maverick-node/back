FROM golang:alpine as builder
WORKDIR /app
COPY . .
RUN apk add --no-cache gcc musl-dev sqlite-dev
ENV CGO_ENABLED=1
RUN go mod tidy
RUN go build -o main .

FROM alpine:latest
WORKDIR /app
LABEL version="1.0.0"
LABEL maintainer="Social-Network"
LABEL description="Social Network Backend Service"
COPY --from=builder /app/main /app/main
COPY --from=builder /app/db/migrations /app/db/migrations
RUN apk add --no-cache sqlite-dev
EXPOSE 8080
CMD ["./main"]
