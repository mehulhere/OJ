# -------- build stage --------
FROM golang:1.22 AS builder
WORKDIR /app
COPY . .
RUN go build -o /python_executor main.go

# -------- runtime stage --------
FROM python:3.12-bookworm
COPY --from=builder /python_executor /usr/local/bin/python_executor
EXPOSE 8080
ENTRYPOINT ["python_executor"] 