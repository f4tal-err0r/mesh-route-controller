FROM golang:1.18-alpine
WORKDIR /app
ADD . /app
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /app/route-generator

FROM alpine:latest
WORKDIR /app
COPY --from=0 /app/route-generator /app/route-generator
CMD ["/app/route-generator"]