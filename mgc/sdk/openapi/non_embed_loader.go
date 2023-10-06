//go:build !embed

package openapi

import "magalu.cloud/core/dataloader"

func GetEmbedLoader() dataloader.Loader {
	return nil
}
