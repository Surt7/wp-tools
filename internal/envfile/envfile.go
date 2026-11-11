package envfile

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"SetSecrets/internal/generator"
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

func (e Entry) IsSecretKey() bool {
	if !e.IsKey {
		return false
	}
	upper := strings.ToUpper(e.Key)
	return strings.Contains(upper, "PASSWORD") ||
		strings.Contains(upper, "PASS") ||
		strings.Contains(upper, "SECRET") ||
		strings.Contains(upper, "TOKEN") ||
		strings.HasSuffix(upper, "_KEY")
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

// MergeAndInitFromExample initializes or updates destPath (.env) from examplePath (.env.example).
// Existing valid secrets are preserved unless explicitly included in selectedKeysToRegenerate or if force is true.
func MergeAndInitFromExample(examplePath string, destPath string, selectedKeysToRegenerate []string, force bool) error {
	exampleEntries, err := LoadFile(examplePath)
	if err != nil {
		return fmt.Errorf("loading %s: %w", examplePath, err)
	}

	existingMap := make(map[string]string)
	if _, err := os.Stat(destPath); err == nil {
		existingEntries, err := LoadFile(destPath)
		if err == nil {
			existingMap = ToMap(existingEntries)
		}
	}

	regenerateSet := make(map[string]bool)
	for _, k := range selectedKeysToRegenerate {
		regenerateSet[k] = true
	}

	for i := range exampleEntries {
		if !exampleEntries[i].IsKey {
			continue
		}

		key := exampleEntries[i].Key

		// If explicitly selected by the user for generation
		if regenerateSet[key] {
			pwd, err := generator.GenerateRandomPassword(24)
			if err != nil {
				return fmt.Errorf("generating password for %s: %w", key, err)
			}
			exampleEntries[i].Value = pwd
			continue
		}

		// If key already exists in .env with a valid value (not CHANGE_ME and not empty), preserve it!
		if existingVal, exists := existingMap[key]; exists && existingVal != "CHANGE_ME" && existingVal != "" && !force {
			exampleEntries[i].Value = existingVal
			continue
		}

		// If it's a secret key that needs a value (new key or CHANGE_ME) and we are not in selective mode
		if exampleEntries[i].IsSecretKey() && (exampleEntries[i].NeedsValue() || force) && len(selectedKeysToRegenerate) == 0 {
			pwd, err := generator.GenerateRandomPassword(24)
			if err != nil {
				return fmt.Errorf("generating password for %s: %w", key, err)
			}
			exampleEntries[i].Value = pwd
		} else if existingVal, exists := existingMap[key]; exists {
			exampleEntries[i].Value = existingVal
		}
	}

	if err := SaveFile(destPath, exampleEntries); err != nil {
		return fmt.Errorf("saving %s: %w", destPath, err)
	}
	return nil
}

// InitFromExample initializes destPath (e.g. .env) from examplePath (.env.example),
// generating secure random passwords for keys that require them.
func InitFromExample(examplePath string, destPath string, force bool) error {
	return MergeAndInitFromExample(examplePath, destPath, nil, force)
}
