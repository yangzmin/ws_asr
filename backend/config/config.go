package config

const (
	// ByteDance Speech Recognition API Configuration
	APIEndpoint = "wss://openspeech.bytedance.com/api/v3/sauc/bigmodel_async"
	AppKey      = "6760719093"
	AccessKey   = "rtoMm7nNXA3hMP5HUGKNzS4IrrGinmgM"

	// Audio Configuration
	DefaultSampleRate = 16000
	DefaultChannels   = 1
	DefaultBits       = 16
	DefaultFormat     = "pcm"
	DefaultCodec      = "raw"

	// Protocol Configuration
	ProtocolVersion = 0b0001

	// Segment Configuration
	DefaultSegmentDuration = 200 // milliseconds
)

// GetAppKey returns the app key
func GetAppKey() string {
	return AppKey
}

// GetAccessKey returns the access key
func GetAccessKey() string {
	return AccessKey
}

// GetAPIEndpoint returns the API endpoint
func GetAPIEndpoint() string {
	return APIEndpoint
}
