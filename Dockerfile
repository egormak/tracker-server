FROM golang:1.24.2 AS builder

ENV TZ=Europe/Kiev

WORKDIR /src/

COPY . .

# RUN go get -d -v ./
RUN go build -o /out/app cmd/server/main.go

FROM ubuntu:24.10 AS bin

ENV TZ=Europe/Kiev

RUN apt update && \
    apt install -y ca-certificates

COPY --from=builder /out/app /

CMD ["/app"]