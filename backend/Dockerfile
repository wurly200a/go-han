FROM golang:1.25

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod tidy

COPY . .

RUN go build -o backend main.go

EXPOSE 8080

CMD ["./backend"]
