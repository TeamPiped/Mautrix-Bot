FROM golang:alpine AS build

WORKDIR /app/

COPY . .

RUN go build -ldflags "-s -w" bot.go

FROM alpine:edge

WORKDIR /app/

COPY --from=build /app/bot /app/piped-mautrix

CMD ./piped-mautrix
