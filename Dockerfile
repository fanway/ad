FROM golang:1.15.8-alpine3.13 as builder

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN chmod +x wait-for-it.sh

ENV GO111MODULE=on
RUN CGO_ENABLED=0 GOOS=linux go build -o main 

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app/
COPY --from=builder /build .
CMD ["./wait-for-it.sh" , "database:5432" , "--timeout=300" , "--" , "./main"]
