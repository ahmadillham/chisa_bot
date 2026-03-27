package config

import "testing"

func TestValidateURL(t *testing.T) {
	valid := []struct {
		name string
		url  string
	}{
		{"https url", "https://example.com"},
		{"http url", "http://example.com"},
		{"youtube url", "https://www.youtube.com/watch?v=abc123"},
		{"tiktok url", "https://vm.tiktok.com/abc123/"},
		{"url with path", "https://example.com/path/to/resource"},
		{"url with query", "https://example.com/path?q=1&b=2"},
		{"instagram url", "https://www.instagram.com/reel/abc123/"},
	}

	for _, tt := range valid {
		t.Run(tt.name, func(t *testing.T) {
			if !ValidateURL(tt.url) {
				t.Errorf("ValidateURL(%q) = false, want true", tt.url)
			}
		})
	}

	invalid := []struct {
		name string
		url  string
	}{
		{"empty string", ""},
		{"whitespace only", "   "},
		{"no scheme", "example.com"},
		{"ftp scheme", "ftp://example.com"},
		{"javascript scheme", "javascript:alert(1)"},
		{"flag injection", "--output /etc/passwd"},
		{"spaces in url", "https://example.com/ malicious"},
		{"no host", "https://"},
		{"data uri", "data:text/html,<h1>hello</h1>"},
		{"file scheme", "file:///etc/passwd"},
	}

	for _, tt := range invalid {
		t.Run(tt.name, func(t *testing.T) {
			if ValidateURL(tt.url) {
				t.Errorf("ValidateURL(%q) = true, want false", tt.url)
			}
		})
	}
}
