FROM golang:1.25-alpine AS builder

# Set the working directory for container
WORKDIR /app

# Copy the dependency files
COPY /go.mod /go.sum ./

# dowload the dependency files
RUN go mod download

# Copy the rest of the code 
COPY . .

# Build the binary no need to call C libraries, so CGO_ENABLED=0
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./app/main.go

# Final lightweight stage
FROM alpine:latest AS final

# Copy the complied binary from builder stage
COPY --from=builder /app/server .

EXPOSE 4221

CMD ["./server" ]