FROM golang:1.24.1

WORKDIR /app

COPY . .
RUN go mod download
RUN go build -o actorfreq .

EXPOSE 8080

CMD ["./actorfreq"]
