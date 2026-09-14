package envfile

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

type Entry struct {
	Key     string
	Value   string
	RawLine string
	IsKey   bool
}

func (e Entry) NeedsValue() bool {
	if !e.IsKey {
		return false
	}
	return e.Value == "CHANGE_ME" || e.Value == ""
}

func Parse(r io.Reader) ([]Entry, error) {
	var entries []Entry
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			entries = append(entries, Entry{
				RawLine: line,
				IsKey:   false,
			})
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			entries = append(entries, Entry{
				Key:     parts[0],
				Value:   parts[1],
				RawLine: line,
				IsKey:   true,
			})
		} else {
			entries = append(entries, Entry{
				RawLine: line,
				IsKey:   false,
			})
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading issue: %w", err)
	}

	return entries, nil
}

func Write(w io.Writer, entries []Entry) error {
	writer := bufio.NewWriter(w)

	for _, entry := range entries {
		var line string
		if entry.IsKey {
			line = fmt.Sprintf("%s=%s", entry.Key, entry.Value)
		} else {
			line = entry.RawLine
		}

		if _, err := fmt.Fprintln(writer, line); err != nil {
			return fmt.Errorf("writing error: %w", err)
		}
	}
	return writer.Flush()
}

func LoadFile(filePath string) ([]Entry, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("opening file: %w", err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			fmt.Printf("closing file: %v", err)
		}
	}(file)
	return Parse(file)
}

func SaveFile(filePath string, entries []Entry) error {
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("creating/ opening file %s: %w", filePath, err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			fmt.Printf("closing file: %v", err)
		}
	}(file)
	return Write(file, entries)
}

func SetEntry(entries []Entry, key string, value string) []Entry {
	found := false

	for i := range entries {
		if entries[i].IsKey && entries[i].Key == key {
			entries[i].Value = value
			found = true
			break
		}
	}

	if !found {
		entries = append(entries, Entry{IsKey: true, Key: key, Value: value})
	}

	return entries
}

func ToMap(entries []Entry) map[string]string {
	result := make(map[string]string)
	for _, entry := range entries {
		if entry.IsKey {
			result[entry.Key] = entry.Value
		}
	}
	return result
}
