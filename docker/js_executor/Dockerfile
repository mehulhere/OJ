# -------- build stage --------
FROM golang:1.22 AS builder
WORKDIR /app
COPY . .
RUN go build -o /js_executor main.go

# -------- runtime stage --------
FROM node:22-bookworm
COPY --from=builder /js_executor /usr/local/bin/js_executor
EXPOSE 8080
ENTRYPOINT ["js_executor"] 