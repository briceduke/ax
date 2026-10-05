FROM golang:1.25-bookworm
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /usr/local/bin/ax ./cmd/ax
CMD ["ax"]
