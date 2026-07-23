package usage

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"time"
)

var linePattern = regexp.MustCompile(`^\[(\d{4}-\d{2}-\d{2} [^\]]+)\]\[([^\]]+)\]\s+in/\s*(\d+)\s+out/\s*(\d+)\s+write/\s*(\d+)\s+hit/\s*(\d+)`)

const timestampLayout = "2006-01-02 15:04:05.000"

type ModelUsage struct {
	Input  uint64 `json:"input"`
	Output uint64 `json:"output"`
	Write  uint64 `json:"write"`
	Hit    uint64 `json:"hit"`
}

func Usage(path string, days int, now time.Time) (map[string]ModelUsage, error) {
	if days < 1 {
		return nil, fmt.Errorf("days must be positive")
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	cutoff := now.Add(-time.Duration(days) * 24 * time.Hour)
	result := make(map[string]ModelUsage)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		matches := linePattern.FindStringSubmatch(scanner.Text())
		if len(matches) != 7 {
			continue
		}
		timestamp, parseErr := time.ParseInLocation(timestampLayout, matches[1], now.Location())
		if parseErr != nil || timestamp.Before(cutoff) || timestamp.After(now) {
			continue
		}

		model := matches[2]

		values := [4]uint64{}
		valid := true
		for i := range values {
			values[i], err = strconv.ParseUint(matches[i+3], 10, 64)
			if err != nil {
				valid = false
				break
			}
		}
		if !valid {
			continue
		}

		u := result[model]
		u.Input += values[0]
		u.Output += values[1]
		u.Write += values[2]
		u.Hit += values[3]
		result[model] = u
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return result, nil
}
