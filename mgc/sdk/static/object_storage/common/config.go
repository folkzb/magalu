package common

import "github.com/MagaluCloud/magalu/mgc/core/config"

type Config struct {
	Workers   int    `json:"workers,omitempty" jsonschema:"description=Number of routines that spawn to do parallel operations within object_storage,default=5,minimum=1,required"`
	ChunkSize uint64 `json:"chunkSize,omitempty" jsonschema:"description=Chunk size to consider when doing multipart requests. Specified in Mb,default=8,minimum=8,maximum=5120,required"`
	Region    string `json:"region,omitempty" jsonschema:"description=Region to reach the service,default=br-se1"`
	Retries   int    `json:"retries,omitempty" jsonschema:"description=Maximum number of retries for transient network errors,default=5,minimum=0"`
	
	// See more about the 'squash' directive here: https://pkg.go.dev/github.com/mitchellh/mapstructure#hdr-Embedded_Structs_and_Squashing
	config.NetworkConfig `json:",squash"` // nolint
}

func (c *Config) chunkSizeInBytes() uint64 {
	if c.ChunkSize <= MIN_CHUNK_SIZE {
		return MIN_CHUNK_SIZE
	}
	if c.ChunkSize >= MAX_CHUNK_SIZE {
		return MAX_CHUNK_SIZE
	}

	return c.ChunkSize * (1024 * 1024)
}
