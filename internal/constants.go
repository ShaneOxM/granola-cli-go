package internal

// Application constants
const (
	// Default configuration values
	DefaultBatchSize = 10   // Default batch size for embedding processing
	DefaultLimit     = 100  // Default number of results to return
	MaxLimit         = 1000 // Maximum allowed limit
	MaxRetries       = 3    // HTTP retry count
	BaseRetryDelay   = 250  // Base delay for exponential backoff (ms)

	// Inference constants
	DefaultTemperature = 0.7
	DefaultTopP        = 0.9
	DefaultMaxTokens   = 1000
)

// Embedding constants
const (
	OllamaEmbeddingDimensions = 768  // nomic-embed-text model dimension
	OllamaContextLimit        = 8192 // Maximum context length in characters
)

// File and path constants
const (
	LockFilePrefix = ".granola_lock"
	PIDLineFormat  = "%d\n"
)

// Config keys
const (
	ConfigPathEnv  = "GRANOLA_CONFIG_PATH"
	BaseURLEnv     = "GRANOLA_BASE_URL"
	ModelEnv       = "GRANOLA_MODEL"
	SSHHostEnv     = "GRANOLA_SSH_HOST"
	SSHIPEnv       = "GRANOLA_SSH_IP"
	TailscaleIPEnv = "GRANOLA_TAILSCALE_IP"
)
