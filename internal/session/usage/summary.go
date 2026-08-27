package usage

import (
	"context"
	"fmt"
	"time"
)

type ModelUsage struct {
	Input  uint64 `json:"input"`
	Output uint64 `json:"output"`
	Write  uint64 `json:"write"`
	Hit    uint64 `json:"hit"`
}

func Usage(sessionID string, days int, now time.Time) (map[string]ModelUsage, error) {
	from, err := window(days, now)
	if err != nil {
		return nil, err
	}
	if sessionID == "" {
		return make(map[string]ModelUsage), nil
	}

	return aggregate(`
	SELECT model, SUM(input), SUM(output), SUM(write), SUM(hit)
	FROM usage
	WHERE session_id = ? AND send_at >= ? AND send_at <= ?
	GROUP BY model`, sessionID, from, now.Unix())
}

func Total(days int, now time.Time) (map[string]ModelUsage, error) {
	from, err := window(days, now)
	if err != nil {
		return nil, err
	}

	return aggregate(`
	SELECT model, SUM(input), SUM(output), SUM(write), SUM(hit)
	FROM usage
	WHERE send_at >= ? AND send_at <= ?
	GROUP BY model`, from, now.Unix())
}

func window(days int, now time.Time) (int64, error) {
	if days < 1 {
		return 0, fmt.Errorf("days must be positive")
	}
	return now.Add(-time.Duration(days) * 24 * time.Hour).Unix(), nil
}

func aggregate(query string, args ...any) (map[string]ModelUsage, error) {
	result := make(map[string]ModelUsage)
	if conn == nil {
		return result, nil
	}

	rows, err := conn.QueryContext(context.Background(), query, args...)
	if err != nil {
		return nil, fmt.Errorf("sql.DB QueryContext [SELECT usage]: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var model string
		var input, output, write, hit int64
		if err := rows.Scan(&model, &input, &output, &write, &hit); err != nil {
			return nil, fmt.Errorf("sql.Rows Scan [SELECT usage]: %w", err)
		}
		result[model] = ModelUsage{
			Input:  uint64(max(input, 0)),
			Output: uint64(max(output, 0)),
			Write:  uint64(max(write, 0)),
			Hit:    uint64(max(hit, 0)),
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sql.Rows Err [SELECT usage]: %w", err)
	}
	return result, nil
}
