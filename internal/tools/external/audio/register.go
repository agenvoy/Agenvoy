package audio

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	toolRegister "github.com/pardnchiu/agenvoy/internal/tools/register"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

func Register() {
	InstallTranscriber()

	toolRegister.Regist(toolRegister.Def{
		Name:        "generate_audio",
		SystemUse:   false,
		AlwaysLoad:  false,
		AlwaysAllow: false,
		Concurrent:  true,
		Timeout:     5 * time.Minute,
		Description: `[system-default] Speaks text with the configured text-to-speech model and writes the audio to disk.
Use for 唸出來 / 生成語音 / 轉成語音檔 / read this aloud.
Transcribing an existing recording → read_files; drawing an image → generate_image.
Returns the saved path, not the audio data.`,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"text": map[string]any{
					"type":        "string",
					"description": "Exactly the words to speak, written as they should be read aloud — not a description of them.",
				},
				"voice": map[string]any{
					"type":        "string",
					"description": "Provider voice name, e.g. alloy for openai, Kore for gemini. Blank uses the provider default.",
					"default":     "",
				},
				"output_file": map[string]any{
					"type":        "string",
					"description": "File name saved under the download directory; any directory part is dropped and the extension is forced to .wav, since that is the only format produced. Blank auto-names it audio-<timestamp>.wav.",
					"default":     "",
				},
			},
			"required": []string{"text"},
		},
		Handler: func(ctx context.Context, e *toolTypes.Executor, args json.RawMessage) (string, error) {
			var params struct {
				Text       string `json:"text"`
				Voice      string `json:"voice"`
				OutputFile string `json:"output_file"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return "", fmt.Errorf("json.Unmarshal: %w", err)
			}
			return Generate(ctx, Request{
				Text:       params.Text,
				Voice:      params.Voice,
				OutputFile: params.OutputFile,
			})
		},
	})
}
