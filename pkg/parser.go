package pkg

import (
	"fmt"
	"net/textproto"
	"strconv"
	"strings"
)

type Request struct {
	Method  string
	Target  string
	Version string
}

func (r *Request) SetRequest(rawStr string) error {
	parts := strings.Fields(rawStr) // handles multiple spaces safely
	if len(parts) < 3 {
		return fmt.Errorf("invalid request line: expected 3 parts, but got: %d (input: %q)", len(parts), rawStr)
	}
	r.Method = parts[0]
	r.Target = parts[1]
	r.Version = parts[2]

	return nil
}

type Headers struct {
	Values map[string]string
}

func (h *Headers) SetHeaders(rawHeaders string) error {
	if h.Values == nil {
		h.Values = make(map[string]string)
	}

	lines := strings.Split(rawHeaders, "\r\n")

	for i, line := range lines {
		if line == "" {
			continue
		}
		frag := strings.SplitN(line, ":", 2)
		if len(frag) < 2 {
			return fmt.Errorf("invalid header line %d: missing colon (line: %q)", i+1, line)
		}
		key := textproto.CanonicalMIMEHeaderKey(strings.TrimSpace(frag[0]))
		value := strings.TrimSpace(frag[1])
		h.Values[key] = value
	}
	return nil
}

type Body struct {
	Raw []byte
}

func (b *Body) SetBody(rawBody []byte, headers Headers) error {

	clStr, exists := headers.Values["Content-Length"]
	if !exists {
		b.Raw = nil
		return nil
	}

	cl, err := strconv.Atoi(clStr)
	if err != nil {
		return fmt.Errorf("invalid Content-Length: %w", err)
	}
	if cl > len(rawBody) {
		return fmt.Errorf("body too short, expected %d bytes, got %d", cl, len(rawBody))
	}
	b.Raw = rawBody[:cl]
	return nil
}

type Parser struct {
	RequestLine Request
	Headers     Headers
	Body        Body
}

func (p *Parser) Parse(httpRequest string) error {

	sections := strings.Split(httpRequest, "\r\n")
	if len(sections) == 0 {
		return fmt.Errorf("empty request")
	}

	// parse request line
	if err := p.RequestLine.SetRequest(sections[0]); err != nil {
		return fmt.Errorf("failed to parse request line: %w", err)
	}

	// parse headers
	if len(sections) > 1 {
		headersBlock := strings.Join(sections[1:], "\r\n")
		if err := p.Headers.SetHeaders(headersBlock); err != nil {
			return fmt.Errorf("failed to parse headers: %w", err)
		}
	}

	return nil
}
