package router

import (
	"testing"
)

func TestParse_ValidCommands(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantCmd string
		wantArgs []string
	}{
		{"dot prefix", ".menu", "menu", nil},
		{"bang prefix", "!menu", "menu", nil},
		{"slash prefix", "/menu", "menu", nil},
		{"command with args", ".dl https://example.com", "dl", []string{"https://example.com"}},
		{"command with multiple args", ".warn @user reason", "warn", []string{"@user", "reason"}},
		{"command case insensitive", ".MENU", "menu", nil},
		{"command mixed case", ".DL https://example.com", "dl", []string{"https://example.com"}},
		{"sticker shorthand", ".s", "s", nil},
		{"text sticker with text", ".ts MENGANCAM", "ts", []string{"MENGANCAM"}},
		{"leading whitespace", "  .menu", "menu", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Parse(tt.input)
			if result == nil {
				t.Fatalf("Parse(%q) = nil, want command %q", tt.input, tt.wantCmd)
			}
			if result.Command != tt.wantCmd {
				t.Errorf("Command = %q, want %q", result.Command, tt.wantCmd)
			}
			if len(result.Args) != len(tt.wantArgs) {
				t.Errorf("Args = %v (len %d), want %v (len %d)", result.Args, len(result.Args), tt.wantArgs, len(tt.wantArgs))
			} else {
				for i, arg := range result.Args {
					if arg != tt.wantArgs[i] {
						t.Errorf("Args[%d] = %q, want %q", i, arg, tt.wantArgs[i])
					}
				}
			}
		})
	}
}

func TestParse_InvalidInputs(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"whitespace only", "   "},
		{"no prefix", "menu"},
		{"prefix only dot", "."},
		{"prefix only bang", "!"},
		{"prefix only slash", "/"},
		{"prefix with space only", ". "},
		{"regular text", "hello world"},
		{"number", "123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Parse(tt.input)
			if result != nil {
				t.Errorf("Parse(%q) = %+v, want nil", tt.input, result)
			}
		})
	}
}

func TestParse_RawArgs(t *testing.T) {
	result := Parse(".dl https://example.com/path?q=1")
	if result == nil {
		t.Fatal("Parse returned nil")
	}
	if result.RawArgs != "https://example.com/path?q=1" {
		t.Errorf("RawArgs = %q, want %q", result.RawArgs, "https://example.com/path?q=1")
	}
}

func TestParse_AllPrefixes(t *testing.T) {
	prefixes := []string{".", "!", "/"}
	for _, prefix := range prefixes {
		result := Parse(prefix + "test")
		if result == nil {
			t.Errorf("Parse(%q) = nil, want valid result", prefix+"test")
			continue
		}
		if result.Prefix != prefix {
			t.Errorf("Prefix = %q, want %q", result.Prefix, prefix)
		}
	}
}
