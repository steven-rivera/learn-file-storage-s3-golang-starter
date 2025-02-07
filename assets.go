package main

import (
	"crypto/rand"
	"encoding/base64"
	"os"
	"strings"
)

func (cfg apiConfig) ensureAssetsDir() error {
	if _, err := os.Stat(cfg.assetsRoot); os.IsNotExist(err) {
		return os.Mkdir(cfg.assetsRoot, 0755)
	}
	return nil
}


func getAssetPath(mediaType string) string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		panic("failed to generate random bytes")
	}

	assetName := base64.RawURLEncoding.EncodeToString(bytes)
	return assetName + mediaTypeToExt(mediaType)
}

func mediaTypeToExt(mediaType string) string {
	parts := strings.Split(mediaType, "/")
	if len(parts) != 2 {
		return ".bin"
	}
	return "." + parts[1]
}

func isValidImageType(mediaType string) bool {
	switch mediaType {
		case "image/jpeg":
			return true
		case "image/png":
			return true
		default:
			return false
	}
}