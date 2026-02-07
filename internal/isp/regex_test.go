package isp

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

const (
	FILENAME = "testdata/webdomain.txt"
)

func TestRegexWebDomain(t *testing.T) {
	t.Run("webdomain regex parsing", func(t *testing.T) {
		bytes, err := os.ReadFile(FILENAME)
		if err != nil {
			t.Fatalf("failed to read file %s: %v", FILENAME, err)
		}

		re := regexp.MustCompile(MGR_WEBDOMAIN_REGEX)

		lines := strings.Split(string(bytes), "\n")
		matchedCount := 0

		for i, line := range lines {
			// Skip empty lines
			if len(strings.TrimSpace(line)) == 0 {
				continue
			}

			match := re.FindStringSubmatch(line)
			if match == nil {
				t.Fatalf("regex did not match line %d: %s", i+1, line)
			}

			matchedCount++
		}

		if matchedCount == 0 {
			t.Fatal("no lines to check")
		}

		t.Logf("successfully matched %d lines", matchedCount)
	})
}
