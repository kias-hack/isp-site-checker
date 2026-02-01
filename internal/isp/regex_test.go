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
	t.Run("проверка регулярки для парсинга webdomain", func(t *testing.T) {
		bytes, err := os.ReadFile(FILENAME)
		if err != nil {
			t.Fatalf("не удалось прочитать файл %s: %v", FILENAME, err)
		}

		re := regexp.MustCompile(MGR_WEBDOMAIN_REGEX)

		lines := strings.Split(string(bytes), "\n")
		matchedCount := 0

		for i, line := range lines {
			// Пропускаем пустые строки
			if len(strings.TrimSpace(line)) == 0 {
				continue
			}

			match := re.FindStringSubmatch(line)
			if match == nil {
				t.Fatalf("регулярка не совпала на строке %d: %s", i+1, line)
			}

			matchedCount++
		}

		if matchedCount == 0 {
			t.Fatal("не найдено ни одной строки для проверки")
		}

		t.Logf("успешно обработано строк: %d", matchedCount)
	})
}
