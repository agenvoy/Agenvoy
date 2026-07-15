package provider

type Models struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

type CopilotModels struct {
	Data []struct {
		ID                 string `json:"id"`
		ModelPickerEnabled bool   `json:"model_picker_enabled"`
		Policy             struct {
			State string `json:"state"`
		} `json:"policy"`
	} `json:"data"`
}

type GeminiModels struct {
	Models []struct {
		Name string `json:"name"`
	} `json:"models"`
}

type CloudFlareModels struct {
	Result []struct {
		Name string `json:"name"`
		Task struct {
			Name string `json:"name"`
		} `json:"task"`
	} `json:"result"`
}
