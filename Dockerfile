FROM golang:1.25.1-alpine AS builder

ENV TZ=Europe/Kiev
ENV CGO_ENABLED=0
ENV GOOS=linux

WORKDIR /src/

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o /out/app cmd/server/main.go

FROM alpine:3.21 AS bin

ENV TZ=Europe/Kiev

RUN apk add --no-cache ca-certificates tzdata && \
    cp /usr/share/zoneinfo/$TZ /etc/localtime && \
    echo $TZ > /etc/timezone

COPY --from=builder /out/app /app

CMD ["/app"]