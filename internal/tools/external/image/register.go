package image

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	toolRegister "github.com/pardnchiu/agenvoy/internal/tools/register"
	toolTypes "github.com/pardnchiu/agenvoy/internal/tools/types"
)

func Register() {
	toolRegister.Regist(toolRegister.Def{
		Name:        "generate_image",
		SystemUse:   false,
		AlwaysLoad:  false,
		AlwaysAllow: false,
		Concurrent:  true,
		Timeout:     15 * time.Minute,
		Description: `[system-default] Generates an image from a text prompt and writes it to disk.
Use for 畫一張 / 生成圖片 / make me an image, and for edits when a reference image is given.
Reading or describing an existing image → read_files; downloading one that already exists → download_file.
Returns the saved path, not the image data.`,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"prompt": map[string]any{
					"type":        "string",
					"description": "What to draw, in English. Describe subject, composition and style; the provider rewrites vague prompts on its own.",
				},
				"aspect_ratio": map[string]any{
					"type":        "string",
					"description": "Shape of the canvas, e.g. 1:1, 16:9, 4:3, 9:16.",
					"default":     "1:1",
				},
				"size": map[string]any{
					"type":        "string",
					"description": "Longest-edge resolution: 1k, 2k or 4k.",
					"enum":        []string{"1k", "2k", "4k"},
					"default":     "1k",
				},
				"quality": map[string]any{
					"type":        "string",
					"description": "Render effort; ignored by gemini.",
					"enum":        []string{"low", "medium", "high"},
					"default":     "medium",
				},
				"reference": map[string]any{
					"type":        "string",
					"description": "Absolute path to a png/jpg/webp used as the image to edit or imitate.",
					"default":     "",
				},
				"output_file": map[string]any{
					"type":        "string",
					"description": "Save path. Absolute is used as-is; relative is joined under the working directory. Extension is added from the returned mime type when omitted.",
					"default":     "",
				},
			},
			"required": []string{"prompt"},
		},
		Handler: func(ctx context.Context, e *toolTypes.Executor, args json.RawMessage) (string, error) {
			var params struct {
				Prompt      string `json:"prompt"`
				AspectRatio string `json:"aspect_ratio"`
				Size        string `json:"size"`
				Quality     string `json:"quality"`
				Reference   string `json:"reference"`
				OutputFile  string `json:"output_file"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return "", fmt.Errorf("json.Unmarshal: %w", err)
			}

			workDir := ""
			if e != nil {
				workDir = e.WorkDir
			}
			return Generate(ctx, Request{
				Prompt:      params.Prompt,
				AspectRatio: params.AspectRatio,
				Size:        params.Size,
				Quality:     params.Quality,
				Reference:   params.Reference,
				OutputFile:  params.OutputFile,
				WorkDir:     workDir,
			})
		},
	})
}
