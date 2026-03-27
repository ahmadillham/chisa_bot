package router

import (
	"strings"

	"chisa_bot/internal/config"
)

// ParseResult holds the parsed command and its arguments.
type ParseResult struct {
	Prefix  string
	Command string
	Args    []string
	RawArgs string
}

// Parse attempts to parse a command from the given text.
// Returns nil if the text does not match any known prefix.
func Parse(text string) *ParseResult {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	for _, prefix := range config.Prefixes {
		if strings.HasPrefix(text, prefix) {
			body := strings.TrimPrefix(text, prefix)
			if body == "" {
				return nil
			}
			parts := strings.Fields(body)
			if len(parts) == 0 {
				return nil
			}

			cmd := strings.ToLower(parts[0])
			args := parts[1:]
			rawArgs := ""
			if len(args) > 0 {
				rawArgs = strings.TrimSpace(strings.TrimPrefix(body, parts[0]))
			}

			return &ParseResult{
				Prefix:  prefix,
				Command: cmd,
				Args:    args,
				RawArgs: rawArgs,
			}
		}
	}
	return nil
}
