# -------- build stage --------
FROM golang:1.22 AS builder
WORKDIR /app
COPY . .
RUN go build -o /java_executor main.go

# -------- runtime stage --------
FROM eclipse-temurin:21-jdk
COPY --from=builder /java_executor /usr/local/bin/java_executor
EXPOSE 8080
ENTRYPOINT ["java_executor"] 