FROM golang:1.26

WORKDIR /service

COPY go.mod go.sum ./
RUN go mod download

RUN go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

COPY . .

RUN go build -o server ./cmd/server

EXPOSE 8080

CMD ["./server"]
