//go:build ignore

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func main() {
	files, err := filepath.Glob("locales/*.json")
	if err != nil {
		fatal(err)
	}
	if len(files) == 0 {
		fatal(fmt.Errorf("no locale files found"))
	}
	sort.Strings(files)

	reference := readLocale("locales/en.json")
	failed := false
	for _, file := range files {
		messages := readLocale(file)
		for key := range reference {
			if strings.TrimSpace(messages[key]) == "" {
				fmt.Fprintf(os.Stderr, "%s: missing or empty key %q\n", file, key)
				failed = true
			}
		}
		for key := range messages {
			if _, ok := reference[key]; !ok {
				fmt.Fprintf(os.Stderr, "%s: unknown key %q\n", file, key)
				failed = true
			}
		}
	}
	if failed {
		os.Exit(1)
	}
}

func readLocale(path string) map[string]string {
	data, err := os.ReadFile(path)
	if err != nil {
		fatal(err)
	}
	var messages map[string]string
	if err := json.Unmarshal(data, &messages); err != nil {
		fatal(fmt.Errorf("%s: %w", path, err))
	}
	return messages
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
