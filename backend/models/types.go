package models

// AudioConfig represents audio configuration
type AudioConfig struct {
	SampleRate int    `json:"sample_rate"`
	Channels   int    `json:"channels"`
	Format     string `json:"format"`
}

// WSMessage represents WebSocket message structure
type WSMessage struct {
	Type     string       `json:"type"`
	Data     string       `json:"data,omitempty"`
	Sequence int          `json:"sequence,omitempty"`
	IsFinal  bool         `json:"is_final,omitempty"`
	Config   *AudioConfig `json:"audio_config,omitempty"`
}

// RecognitionResult represents speech recognition result
type RecognitionResult struct {
	Type       string  `json:"type"`
	Text       string  `json:"text"`
	IsFinal    bool    `json:"is_final"`
	Confidence float64 `json:"confidence"`
	Timestamp  int64   `json:"timestamp"`
}

// ByteDance API related structures
type UserMeta struct {
	Uid        string `json:"uid,omitempty"`
	Did        string `json:"did,omitempty"`
	Platform   string `json:"platform,omitempty"`
	SDKVersion string `json:"sdk_version,omitempty"`
	APPVersion string `json:"app_version,omitempty"`
}

type AudioMeta struct {
	Format  string `json:"format,omitempty"`
	Codec   string `json:"codec,omitempty"`
	Rate    int    `json:"rate,omitempty"`
	Bits    int    `json:"bits,omitempty"`
	Channel int    `json:"channel,omitempty"`
}

type CorpusMeta struct {
	BoostingTableName string `json:"boosting_table_name,omitempty"`
	CorrectTableName  string `json:"correct_table_name,omitempty"`
	Context           string `json:"context,omitempty"`
}

type RequestMeta struct {
	ModelName       string      `json:"model_name,omitempty"`
	EnableITN       bool        `json:"enable_itn,omitempty"`
	EnablePUNC      bool        `json:"enable_punc,omitempty"`
	EnableDDC       bool        `json:"enable_ddc,omitempty"`
	ShowUtterances  bool        `json:"show_utterances"`
	EnableNonstream bool        `json:"enable_nonstream"`
	Corpus          CorpusMeta  `json:"corpus,omitempty"`
}

type AsrRequestPayload struct {
	User    UserMeta    `json:"user"`
	Audio   AudioMeta   `json:"audio"`
	Request RequestMeta `json:"request"`
}

type AsrResponsePayload struct {
	AudioInfo struct {
		Duration int `json:"duration"`
	} `json:"audio_info"`
	Result struct {
		Text       string `json:"text"`
		Utterances []struct {
			Definite  bool   `json:"definite"`
			EndTime   int    `json:"end_time"`
			StartTime int    `json:"start_time"`
			Text      string `json:"text"`
			Words     []struct {
				EndTime   int    `json:"end_time"`
				StartTime int    `json:"start_time"`
				Text      string `json:"text"`
			} `json:"words"`
		} `json:"utterances,omitempty"`
	} `json:"result"`
	Error string `json:"error,omitempty"`
}

type AsrResponse struct {
	Code            int                 `json:"code"`
	Event           int                 `json:"event"`
	IsLastPackage   bool                `json:"is_last_package"`
	PayloadSequence int32               `json:"payload_sequence"`
	PayloadSize     int                 `json:"payload_size"`
	PayloadMsg      *AsrResponsePayload `json:"payload_msg"`
}