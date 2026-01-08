package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
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

var statusTexts = map[int]string{
	200: "HTTP/1.1 200 OK",
	201: "HTTP/1.1 201 Created",
	400: "HTTP/1.1 400 Bad Request",
	404: "HTTP/1.1 404 Not Found",
	405: "HTTP/1.1 405 Method Not Allowed",
	500: "HTTP/1.1 500 Internal Server Error",
}

func BuildResponse(statusCode int, contentType string, bodyBytes []byte, acceptGzip bool) string {
	statusLine, ok := statusTexts[statusCode]
	if !ok {
		statusLine = "HTTP/1.1 500 Internal Server Error" // fallback
	}
	var headers []string
	headers = append(headers, statusLine)

	if contentType != "" {
		headers = append(headers, "Content-Type: "+contentType)
	}

	finalBody := bodyBytes
	if acceptGzip {
		compressed, err := CompressBody(string(bodyBytes))
		if err == nil {
			finalBody = compressed
			headers = append(headers, "Content-Encoding: gzip")
		}
	}
	headers = append(headers, fmt.Sprintf("Content-Length: %d", len(finalBody)))
	headers = append(headers, "")
	headerStr := strings.Join(headers, "\r\n")
	return headerStr + "\r\n" + string(finalBody)
}

func CompressBody(body string) ([]byte, error) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write([]byte(body)); err != nil {
		fmt.Println("failed to compress into gzip: ", err)
		return nil, err
	}
	if err := gz.Close(); err != nil {
		fmt.Println("failed to Close() after compressing into gzip: ", err)
		return nil, err
	}
	return buf.Bytes(), nil
}

func ServeHTTP(conn net.Conn, parser pkg.Parser, directory string) {
	var response string
	target := parser.RequestLine.Target
	method := parser.RequestLine.Method
	acceptGzip := strings.Contains(parser.Headers.Values["Accept-Encoding"], "gzip")

	switch {
	case target == "/":
		response = BuildResponse(200, "", nil, acceptGzip)

	case strings.HasPrefix(target, "/echo/"):
		echoStr := strings.TrimPrefix(target, "/echo/")
		decode, _ := url.PathUnescape(echoStr)
		response = BuildResponse(200, "text/plain", []byte(decode), acceptGzip)

	case target == "/user-agent":
		ua := parser.Headers.Values["User-Agent"]
		if ua == "" {
			ua = "unknown"
		}
		response = BuildResponse(200, "text/plain", []byte(ua), acceptGzip)

	case strings.HasPrefix(target, "/files/"):

		if directory == "" {
			response = BuildResponse(500, "text/plain", []byte("File directory not configured"), acceptGzip)
		} else {
			fileName := strings.TrimPrefix(target, "/files/")
			fullPath := filepath.Join(directory, fileName)

			if method == "GET" {
				data, err := os.ReadFile(fullPath)
				if err != nil {
					response = BuildResponse(404, "", nil, acceptGzip)
				} else {
					response = BuildResponse(200, "application/octet-stream", data, acceptGzip)
				}
			} else if method == "POST" {
				err := os.WriteFile(fullPath, parser.Body.Raw, 06444)
				if err != nil {
					fmt.Printf("Failed to write file %s: %v\n", fullPath, err)
					response = BuildResponse(500, "", nil, acceptGzip)
				} else {
					response = BuildResponse(201, "", parser.Body.Raw, acceptGzip)
				}
			} else {
				response = BuildResponse(405, "", nil, acceptGzip)
			}
		}
	default:
		response = BuildResponse(404, "", nil, acceptGzip)
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
