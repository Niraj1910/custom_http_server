package pkg

import "testing"

func TestRequest(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantMethod  string
		wantTarget  string
		wantVersion string
		expectValid bool // true if we want parsing to be succeed (all fields filled)
	}{
		{
			name:        "Valid GET request line",
			input:       "GET / HTTP/1.1",
			wantMethod:  "GET",
			wantTarget:  "/",
			wantVersion: "HTTP/1.1",
			expectValid: true,
		},
		{
			name:        "Valid POST with path",
			input:       "POST /submit HTTP/1.0",
			wantMethod:  "POST",
			wantTarget:  "/submit",
			wantVersion: "HTTP/1.0",
			expectValid: true,
		},
		{
			name:        "Path with query string",
			input:       "GET /search?q=go+testing HTTP/1.1",
			wantMethod:  "GET",
			wantTarget:  "/search?q=go+testing",
			wantVersion: "HTTP/1.1",
			expectValid: true,
		},
		{
			name:        "Extra spaces request (should be handled by strings.Fields())",
			input:       "GET      /testing-spaces-request   HTTP/1.1",
			wantMethod:  "GET",
			wantTarget:  "/testing-spaces-request",
			wantVersion: "HTTP/1.1",
			expectValid: true,
		},
		{
			name:        "Invalid test case",
			input:       "GET /",
			wantMethod:  "",
			wantTarget:  "",
			wantVersion: "",
			expectValid: false,
		},
		{
			name:        "Invalid: empty string",
			input:       "",
			wantMethod:  "",
			wantTarget:  "",
			wantVersion: "",
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var r Request
			r.SetRequest(tt.input)

			if tt.expectValid {
				// exect successful parse -> check values
				if r.Method != tt.wantMethod {
					t.Errorf("Method mismatch\ngot: %s\nwant: %s", r.Method, tt.wantMethod)
				}
				if r.Target != tt.wantTarget {
					t.Errorf("Target mismatch\ngot: %s\nwant: %s", r.Target, tt.wantTarget)
				}
				if r.Version != tt.wantVersion {
					t.Errorf("Version mismatch\ngot: %s\nwant: %s", r.Version, tt.wantVersion)
				}
			} else {
				// expect failed parse -> fields should remain empty
				if r.Method != "" {
					t.Errorf("Method should be empty for invalid input, got %q", r.Method)
				}
				if r.Target != "" {
					t.Errorf("Target should be empty for invalid input, got %q", r.Target)
				}
				if r.Version != "" {
					t.Errorf("Version should be empty for invalid input, got %q", r.Version)
				}
			}
		})
	}
}
