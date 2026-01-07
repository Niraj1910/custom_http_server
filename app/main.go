package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Niraj1910/custom_http_server/pkg"
)

func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	fmt.Println("Logs from your program will appear here!")

	var directory string
	if len(os.Args) >= 2 {
		directory = os.Args[1]
		fmt.Printf("Serving files from directory: %s\n", directory)

		// create if missing
		if err := os.MkdirAll(directory, 0755); err != nil {
			fmt.Printf("failed to create directory: %s", directory)
			os.Exit(1)
		}
	} else {
		fmt.Println("No directory provided — file endpoints disabled")
	}

	listner, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}
	defer listner.Close()

	for {
		conn, err := listner.Accept()
		if err != nil {
			fmt.Printf("Error accepting connection: %v\n", err)
			continue
		}
		go handleConnection(conn, directory)
	}
}

func ServeHTTP(conn net.Conn, parser pkg.Parser, directory string) {
	var response string
	target := parser.RequestLine.Target
	method := parser.RequestLine.Method

	switch {
	case target == "/":
		response = "HTTP/1.1 200 OK\r\n\r\n"
	case strings.HasPrefix(target, "/echo/"):
		echoStr := strings.TrimPrefix(target, "/echo/")
		decode, _ := url.PathUnescape(echoStr)
		response = fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(decode), decode)

	case target == "/user-agent":
		ua := parser.Headers.Values["User-Agent"]
		if ua == "" {
			ua = "unknown"
		}
		response = fmt.Sprintf(
			"HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s",
			len(ua), ua,
		)
	case strings.HasPrefix(target, "/files/"):
		fileName := strings.TrimPrefix(target, "/files/")
		fullPath := filepath.Join(directory, fileName)

		if method == "GET" {
			data, err := os.ReadFile(fullPath)
			if err != nil {
				response = "HTTP/1.1 404 Not Found\r\n\r\n"
			} else {
				response = fmt.Sprintf(
					"HTTP/1.1 200 OK\r\nContent-Type: application/octet-stream\r\nContent-Length: %d\r\n\r\n%s",
					len(data), string(data),
				)
			}
		} else if method == "POST" {
			err := os.WriteFile(fullPath, parser.Body.Raw, 06444)
			if err != nil {
				fmt.Printf("Failed to write file %s: %v\n", fullPath, err)
				response = "HTTP/1.1 500 Internal Server Error\r\n\r\n"
			} else {
				response = "HTTP/1.1 201 Created\r\n\r\n"
			}
		} else {
			response = "HTTP/1.1 405 Method Not Allowed\r\n\r\n"
		}
	default:
		response = "HTTP/1.1 404 Not Found\r\n\r\n"
	}
	conn.Write([]byte(response))
}

func handleConnection(conn net.Conn, directory string) {
	defer conn.Close()

	reader := bufio.NewReader(conn)

	var sb strings.Builder
	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			fmt.Println("Error reading line: ", err)
			return
		}
		sb.WriteString(line)

		// if we read a blank line then we are done with the headers and break
		if len(line) <= 2 { // \r\n or \r\n
			// end of reading headers
			break
		}
	}

	headerStr := sb.String()
	// remove the final empty line
	headerStr = strings.TrimSuffix(headerStr, "\r\n\r\n")
	headerStr = strings.TrimSuffix(headerStr, "\n\n")

	var parser pkg.Parser
	if err := parser.Parse(headerStr); err != nil {
		fmt.Printf("Parse error: %v\n", err)
		conn.Write([]byte("HTTP/1.1 400 Bad Request\r\n\r\n"))
		return
	}

	// calculate how much body to read
	contentLength := 0
	if clStr, ok := parser.Headers.Values["Content-Length"]; ok {
		if cl, err := strconv.Atoi(strings.TrimSpace(clStr)); err == nil && cl > 0 {
			contentLength = cl
		}
	}

	var body []byte
	if contentLength > 0 {
		buf := make([]byte, contentLength)
		n, err := reader.Read(buf)
		if err != nil {
			fmt.Printf("Body read error: %v\n", err)
			conn.Write([]byte("HTTP/1.1 400 Bad Request\r\n\r\n"))
			return
		}
		body = buf[:n]
	}

	if err := parser.Body.SetBody(body, parser.Headers); err != nil {
		fmt.Printf("Body parse error: %v\n", err)
		conn.Write([]byte("HTTP/1.1 400 Bad Request\r\n\r\n"))
		return
	}

	// handle http routes
	ServeHTTP(conn, parser, directory)

	fmt.Println("------------------ Request Line -------------------")
	fmt.Printf("Method: %s \nPath: %s \nVersion: %s\n", parser.RequestLine.Method, parser.RequestLine.Target, parser.RequestLine.Version)

	fmt.Println()

	fmt.Println("------------------ Headers Line -------------------")
	headers := parser.Headers.Values
	for key, val := range headers {
		fmt.Println(key + ": " + val)
	}

	fmt.Println()

	fmt.Println("------------------ Body Line -------------------")
	fmt.Println(string(parser.Body.Raw))

}
