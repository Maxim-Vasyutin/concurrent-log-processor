package parser

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

type LogEntry struct {
	Timestamp time.Time
	Level     string
	Service   string
	Message   string
	RequestID string
	UserID    string
}

type LogParser struct{}

func ParseLogLine(line string) (LogEntry, error) {
	return LogParser{}.ParseLine(line)
}

func (p LogParser) ParseLine(line string) (LogEntry, error) {
	timestampText, rest, err := p.splitFirstToken(strings.TrimSpace(line))
	if err != nil {
		return LogEntry{}, err
	}

	timestamp, err := time.Parse(time.RFC3339Nano, timestampText)
	if err != nil {
		return LogEntry{}, fmt.Errorf("invalid timestamp: %w", err)
	}

	level, remainder, err := p.parseLevel(rest)
	if err != nil {
		return LogEntry{}, err
	}

	service, message, err := p.parseServiceAndMessage(remainder)
	if err != nil {
		return LogEntry{}, err
	}

	requestID, err := p.extractValue(message, "request_id")
	if err != nil {
		return LogEntry{}, err
	}

	userID, err := p.extractOptionalValue(message, "user_id")
	if err != nil {
		return LogEntry{}, err
	}

	return LogEntry{
		Timestamp: timestamp,
		Level:     level,
		Service:   service,
		Message:   message,
		RequestID: requestID,
		UserID:    userID,
	}, nil
}

func (p LogParser) splitFirstToken(line string) (string, string, error) {
	parts := strings.SplitN(line, " ", 2)
	if len(parts) != 2 || parts[0] == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", fmt.Errorf("invalid log format")
	}

	return parts[0], strings.TrimSpace(parts[1]), nil
}

func (p LogParser) parseLevel(input string) (string, string, error) {
	if !strings.HasPrefix(input, "[") {
		return "", "", fmt.Errorf("invalid log format")
	}

	end := strings.Index(input, "]")
	if end <= 1 {
		return "", "", fmt.Errorf("invalid log format")
	}

	level := input[1:end]
	remainder := strings.TrimSpace(input[end+1:])
	if remainder == "" {
		return "", "", fmt.Errorf("invalid log format")
	}

	return level, remainder, nil
}

func (p LogParser) parseServiceAndMessage(input string) (string, string, error) {
	parts := strings.SplitN(input, ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid log format")
	}

	service := strings.TrimSpace(parts[0])
	message := strings.TrimSpace(parts[1])
	if service == "" || message == "" {
		return "", "", fmt.Errorf("invalid log format")
	}

	return service, message, nil
}

func (p LogParser) extractValue(message string, key string) (string, error) {
	value, err := p.extractOptionalValue(message, key)
	if err != nil {
		return "", err
	}
	if value == "" {
		return "", fmt.Errorf("missing %s", key)
	}

	return value, nil
}

func (p LogParser) extractOptionalValue(message string, key string) (string, error) {
	pattern, err := regexp.Compile(key + `=([a-zA-Z0-9_]+)`)
	if err != nil {
		return "", fmt.Errorf("compile %s pattern: %w", key, err)
	}

	matches := pattern.FindStringSubmatch(message)
	if len(matches) < 2 {
		return "", nil
	}

	return matches[1], nil
}
