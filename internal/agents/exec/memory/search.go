package memory

import (
	"context"
	"encoding/json"
	"log/slog"
	"slices"
	"sort"
	"strings"

	"github.com/pardnchiu/agenvoy/internal/runtime/torii"
)

func Search(ctx context.Context, tool, keyword string, limit int) string {
	limit = clampLimit(limit)

	if tool == "" && keyword == "" {
		return "keyword is required when tool is not specified"
	}

	db := torii.DB(torii.DBErrorMemory)

	pattern := "*"
	if tool != "" {
		pattern = tool + ":*"
	}

	if keyword != "" {
		if records := vectorSearch(ctx, db, pattern, keyword, limit); len(records) > 0 {
			return format(records, limit)
		}
	}

	records := keywordScan(ctx, db, tool, keyword, limit)
	if len(records) == 0 {
		return "NONE"
	}
	return format(records, limit)
}

func List(limit int) []Record {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	db := torii.DB(torii.DBErrorMemory)
	return scanWithFilter(context.Background(), db, "*", func(Record) bool { return true }, limit)
}

func vectorSearch(ctx context.Context, db *torii.Session, pattern, keyword string, limit int) []Record {
	keys, err := db.VSearch(ctx, keyword, pattern, limit)
	if err != nil || len(keys) == 0 {
		return nil
	}

	out := make([]Record, 0, len(keys))
	for _, key := range keys {
		entry, ok := db.Get(ctx, key)
		if !ok {
			continue
		}
		var rec Record
		if err := json.Unmarshal([]byte(entry.Value()), &rec); err != nil {
			continue
		}
		if err := db.Expire(ctx, key, ttlSeconds); err != nil {
			slog.Debug("memory.Expire",
				slog.String("key", key),
				slog.String("error", err.Error()))
		}
		out = append(out, rec)
	}
	return out
}

func keywordScan(ctx context.Context, db *torii.Session, tool, keyword string, limit int) []Record {
	if tool != "" {
		msg := getMessage(keyword)
		if msg == "unknown" {
			return nil
		}
		return scanWithFilter(ctx, db, tool+":*", func(rec Record) bool {
			return slices.Contains(rec.Keywords, "error_type:"+msg)
		}, limit)
	}

	lower := strings.ToLower(keyword)
	return scanWithFilter(ctx, db, "*", func(rec Record) bool {
		if lower == "" {
			return true
		}
		if strings.Contains(strings.ToLower(rec.ToolName), lower) ||
			strings.Contains(strings.ToLower(rec.Symptom), lower) ||
			strings.Contains(strings.ToLower(rec.Cause), lower) {
			return true
		}
		for _, kw := range rec.Keywords {
			str := strings.ToLower(kw)
			if strings.Contains(str, lower) || strings.Contains(lower, str) {
				return true
			}
		}
		return false
	}, limit)
}

func scanWithFilter(ctx context.Context, db *torii.Session, pattern string, match func(Record) bool, cap int) []Record {
	entries := db.Scan(ctx, pattern, torii.ScanOption{})
	if len(entries) == 0 {
		return nil
	}

	out := make([]Record, 0, cap)
	touched := make([]string, 0, cap)
	for _, entry := range slices.Backward(entries) {
		var rec Record
		if err := json.Unmarshal([]byte(entry.Value()), &rec); err != nil {
			continue
		}
		if !match(rec) {
			continue
		}
		touched = append(touched, entry.Key)
		out = append(out, rec)
		if len(out) >= cap {
			break
		}
	}

	if err := db.ExpireMany(ctx, touched, ttlSeconds); err != nil {
		slog.Debug("memory.ExpireMany", slog.String("error", err.Error()))
	}
	return out
}

func format(records []Record, limit int) string {
	sort.SliceStable(records, func(i, j int) bool {
		return records[i].Timestamp > records[j].Timestamp
	})
	if len(records) > limit {
		records = records[:limit]
	}

	recordBytes, err := json.Marshal(records)
	if err != nil {
		return ""
	}
	return string(recordBytes)
}

func clampLimit(limit int) int {
	if limit <= 0 {
		return 4
	}
	if limit > 16 {
		return 16
	}
	return limit
}

func Read(hash string) string {
	db := torii.DB(torii.DBErrorMemory)
	for _, entry := range db.Scan(context.Background(), "*", torii.ScanOption{Contains: hash}) {
		var record Record
		if err := json.Unmarshal([]byte(entry.Value()), &record); err != nil {
			continue
		}
		if record.ID == hash {
			return entry.Value()
		}
	}
	return "not found"
}
