# Hand-Written HTTP/1.1 Server in Go

A fully functional HTTP/1.1 compliant web server built from scratch in Go, using only the standard library.

No frameworks. No external dependencies. Just raw TCP and deep understanding of the HTTP protocol.

## Features

- **HTTP/1.1 Compliant**
  - Proper request parsing (method, path, headers, body)
  - Persistent connections (Keep-Alive)
  - `Connection: keep-alive/close` header support
  - `Content-Length` and body handling

- **Routing**
  - `/` → root (200 OK)
  - `/echo/<text>` → echoes back the path segment (with URL decoding)
  - `/user-agent` → returns the client's User-Agent header
  - `/files/<filename>` → serve or create files (GET/POST)

- **File Serving**
  - GET: serve files from a configured directory
  - POST: create/overwrite files with request body
  - Safe path handling (prevents directory traversal)

- **Response Compression**
  - Automatic gzip compression when client sends `Accept-Encoding: gzip`
  - Correct `Content-Encoding` and `Content-Length`

- **Error Handling**
  - 400 Bad Request for malformed requests
  - 404 Not Found
  - 405 Method Not Allowed
  - 500 Internal Server Error
  - 201 Created for file uploads

- **Clean Architecture**
  - Separation of parsing, routing, and response building
  - Robust error handling with meaningful messages
  - Optional file directory (enabled via command-line argument)

## Why This Project?

Most Go developers use frameworks like Gin or Echo.  
This project goes deeper — implementing the HTTP protocol directly over TCP to understand:

- How HTTP messages are framed
- How persistent connections work
- How compression reduces bandwidth
- How to safely handle file paths and user input

It's inspired by challenges like [CodeCrafters HTTP Server](https://codecrafters.io) and real-world systems programming.

## Usage

### Run the Server

```bash
# Basic (no file serving)
go run app/main.go

# With file serving enabled
go run app/main.go /tmp/myfiles
```

## Examples
```bash
# Root
curl http://localhost:4221/

# Echo
curl http://localhost:4221/echo/hello-world

# With URL encoding
curl http://localhost:4221/echo/hello%20world

# User-Agent
curl -H "User-Agent: foo" http://localhost:4221/user-agent

# File upload (POST)
curl -X POST http://localhost:4221/files/greeting.txt -d "Hello from Go!"

# File download (GET)
curl http://localhost:4221/files/greeting.txt

# Compression test (raw output to see gzipped bytes)
curl -H "Accept-Encoding: gzip" --raw -o - http://localhost:4221/echo/long-text-here
