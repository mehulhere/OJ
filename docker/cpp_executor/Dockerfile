# -------- build stage --------
FROM golang:1.22 AS builder
WORKDIR /app
COPY . .
RUN go build -o /cpp_executor main.go

# -------- runtime stage --------
FROM gcc:13-bookworm
COPY --from=builder /cpp_executor /usr/local/bin/cpp_executor
EXPOSE 8080
ENTRYPOINT ["cpp_executor"] 