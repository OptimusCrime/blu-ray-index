FROM golang:1.25

ENV TZ=Europe/Oslo

WORKDIR /app

COPY backend/go.mod backend/go.sum ./

RUN go mod download

COPY backend/cmd ./cmd
COPY backend/internal ./internal

RUN go build -o server ./cmd/server

EXPOSE 8140

CMD ["/app/server"]
