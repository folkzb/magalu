//go:build !embed

package blueprint

import "magalu.cloud/core/dataloader"

func GetEmbedLoader() dataloader.Loader {
	return nil
}
