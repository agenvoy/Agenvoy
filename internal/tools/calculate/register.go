package calculate

import (
	"context"
	"encoding/json"
	"fmt"

	toolRegister "github.com/pardnchiu/agenvoy/internal/tools/register"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

func Register() {
	toolRegister.Regist(toolRegister.Def{
		Name:        "calculate",
		AlwaysAllow: true,
		Concurrent:  true,
		Description: `
Evaluate one or more mathematical expressions — arithmetic, unit conversions, currency arithmetic.
Fetch variable data via appropriate tool before passing in. Do not persist results to summary.
Returns {expression: result}; failed expressions return an error string instead of failing the whole call.`,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"expressions": map[string]any{
					"type":        "array",
					"description": "Examples: '(100 + 200) * 3', '10 % 3', '2 ^ 10', 'sqrt(2)', 'pow(2, 10)', 'min(1, 2, 3)', 'max(...)' (variadic).",
					"items":       map[string]any{"type": "string"},
					"minItems":    1,
				},
			},
			"required": []string{"expressions"},
		},
		Handler: func(_ context.Context, _ *toolTypes.Executor, args json.RawMessage) (string, error) {
			var params struct {
				Expressions []string `json:"expressions"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return "", fmt.Errorf("json.Unmarshal: %w", err)
			}
			if len(params.Expressions) == 0 {
				return "", fmt.Errorf("expressions is required")
			}

			out := make(map[string]string, len(params.Expressions))
			for _, expr := range params.Expressions {
				result, err := Calc(expr)
				if err != nil {
					result = "error: " + err.Error()
				}
				out[expr] = result
			}

			raw, err := json.Marshal(out)
			if err != nil {
				return "", fmt.Errorf("json.Marshal: %w", err)
			}
			return string(raw), nil
		},
	})
}
