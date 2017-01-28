package common

const (
	// 5MiB max entry size
	MaxEntrySize   = 1024 * 1024 * 4
	MaxMessageSize = 1024 * 1024 * 5

	// This is the decompressed size
	MaxMessageContentSize = MaxEntrySize
)
