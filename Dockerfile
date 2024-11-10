# Use a multi-stage build to support multiple architectures
# Stage 1: Build stage
FROM golang:1.23.1-alpine AS builder
LABEL org.opencontainers.image.source=https://github.com/papawattu/cleanlog-eventstream
LABEL org.opencontainers.image.description="A simple web app log cleaning house"
LABEL org.opencontainers.image.licenses=MIT

ARG USER=nouser
ARG PORT=3000

RUN apk add --no-cache make && apk add --no-cache sudo && apk add --no-cache librdkafka-dev

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN make build


# Stage 2: Final stage
FROM alpine AS build-stage

ARG USER=nouser

WORKDIR /

COPY --from=builder /app/bin/eventstream /eventstream

RUN adduser -D $USER \
        && mkdir -p /etc/sudoers.d \
        && echo "$USER ALL=(ALL) NOPASSWD: ALL" > /etc/sudoers.d/$USER \
        && chmod 0440 /etc/sudoers.d/$USER

USER $USER

EXPOSE 3000

ENTRYPOINT ["/eventstream"]