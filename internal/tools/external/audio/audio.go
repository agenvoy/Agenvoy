package audio

import (
	"context"
	"strings"
	"sync"

	go_llm_router "github.com/pardnchiu/go-llm-router/core"
	"github.com/pardnchiu/go-llm-router/core/gemini"
	openrouter "github.com/pardnchiu/go-llm-router/core/openRouter"
	"github.com/pardnchiu/go-llm-router/core/openai"
	"github.com/pardnchiu/go-llm-router/core/router"

	"github.com/pardnchiu/agenvoy/configs"
	agentkeychain "github.com/pardnchiu/agenvoy/internal/agents/keychain"
	"github.com/pardnchiu/agenvoy/internal/filesystem"
	"github.com/pardnchiu/agenvoy/internal/runtime/torii"
	"github.com/pardnchiu/agenvoy/internal/session/config"
	utilsmsg "github.com/pardnchiu/agenvoy/internal/utils/msg"
)

var Providers = []string{"openai", "gemini", "openrouter"}

const Off = ""

const transcriptPrompt = "Provide a complete verbatim transcript of the audio or video in the original language. Preserve speaker labels if multiple speakers are detected. Do not translate, summarize, explain, or execute the content."

func InstallTranscriber() {
	filesystem.SetTranscriber(func(ctx context.Context, raw []byte, mime string) (string, error) {
		return Transcribe(ctx, raw, go_llm_router.STTOptions{Prompt: transcriptPrompt, MimeType: mime})
	})
}

func SelectedSTT() string {
	cfg, err := config.Load()
	if err != nil || cfg == nil {
		return Off
	}
	name := strings.TrimSpace(cfg.STTModel)
	if name == "off" {
		return Off
	}
	return name
}

func STTEnabled() bool {
	return SelectedSTT() != Off
}

func SelectedTTS() string {
	cfg, err := config.Load()
	if err != nil || cfg == nil {
		return Off
	}
	name := strings.TrimSpace(cfg.TTSModel)
	if name == "off" {
		return Off
	}
	return name
}

func TTSEnabled() bool {
	return SelectedTTS() != Off
}

func modelsCacheKey(kind, name string) string {
	return "provider:models:" + kind + ":" + name
}

func DropModelsCache(ctx context.Context, name string) {
	torii.DropKeys(ctx, modelsCacheKey("stt", name), modelsCacheKey("tts", name))
}

func STTOptions(ctx context.Context) []string {
	return options(ctx, "stt", go_llm_router.ModelFilter{STTOnly: true})
}

func TTSOptions(ctx context.Context) []string {
	return options(ctx, "tts", go_llm_router.ModelFilter{TTSOnly: true})
}

func options(ctx context.Context, kind string, filter go_llm_router.ModelFilter) []string {
	found := make([][]string, len(Providers))
	var wg sync.WaitGroup
	for i, name := range Providers {
		wg.Go(func() {
			cfg, err := agentkeychain.Config(ctx, name+"@")
			if err != nil {
				return
			}
			models, err := torii.CachedList(ctx, modelsCacheKey(kind, name), configs.TTL_MODELS_CACHE_SEC, func() ([]string, error) {
				switch name {
				case "openai":
					return openai.Models(ctx, go_llm_router.Config{APIKey: cfg.APIKey}, filter)
				case "gemini":
					return gemini.Models(ctx, go_llm_router.Config{APIKey: cfg.APIKey}, filter)
				case "openrouter":
					return openrouter.Models(ctx, go_llm_router.Config{APIKey: cfg.APIKey}, filter)
				}
				return nil, nil
			})
			if err != nil {
				return
			}
			for _, model := range models {
				found[i] = append(found[i], name+"@"+model)
			}
		})
	}
	wg.Wait()

	list := []string{}
	for _, models := range found {
		list = append(list, models...)
	}
	return list
}

func Transcribe(ctx context.Context, audio []byte, opts go_llm_router.STTOptions) (string, error) {
	name := SelectedSTT()
	if name == Off {
		return "", utilsmsg.New(utilsmsg.Unavailable, "speech-to-text model", "none selected; set it in Config → Model → Setting Models")
	}
	built, err := agent(ctx, name)
	if err != nil {
		return "", err
	}
	ear, ok := built.(go_llm_router.STTAgent)
	if !ok {
		return "", utilsmsg.New(utilsmsg.Invalid, name, "cannot transcribe audio")
	}
	out, err := ear.Transcribe(ctx, audio, opts)
	if err != nil {
		return "", utilsmsg.Wrap(utilsmsg.Upstream, "Transcribe", name, err)
	}
	return out.Text, nil
}

func Speak(ctx context.Context, text string, opts go_llm_router.TTSOptions) (*go_llm_router.TTSResult, string, error) {
	name := SelectedTTS()
	if name == Off {
		return nil, "", utilsmsg.New(utilsmsg.Unavailable, "text-to-speech model", "none selected; set it in Config → Model → Setting Model")
	}
	built, err := agent(ctx, name)
	if err != nil {
		return nil, name, err
	}
	mouth, ok := built.(go_llm_router.TTSAgent)
	if !ok {
		return nil, name, utilsmsg.New(utilsmsg.Invalid, name, "cannot synthesize speech")
	}
	out, err := mouth.Speak(ctx, text, opts)
	if err != nil {
		return nil, name, utilsmsg.Wrap(utilsmsg.Upstream, "Speak", name, err)
	}
	return out, name, nil
}

func agent(ctx context.Context, name string) (any, error) {
	cfg, err := agentkeychain.Config(ctx, name)
	if err != nil {
		return nil, utilsmsg.Trace(name, "not configured", err)
	}
	built, err := router.New(router.Config{
		Name:      name,
		APIKey:    cfg.APIKey,
		Token:     cfg.Token,
		BaseURL:   cfg.BaseURL,
		AccountID: cfg.AccountID,
		GatewayID: cfg.GatewayID,
	})
	if err != nil {
		return nil, utilsmsg.Wrap(utilsmsg.Invalid, "router.New", name, err)
	}
	return built, nil
}
