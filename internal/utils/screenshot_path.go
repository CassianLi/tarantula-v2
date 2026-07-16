package utils

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// JoinObjectKey builds an OSS object key from optional prefix + filename.
func JoinObjectKey(prefix, filename string) string {
	p := strings.Trim(strings.ReplaceAll(prefix, "\\", "/"), "/")
	if p == "" {
		return filename
	}
	return p + "/" + filename
}

// OSSObjectKey uses config oss.path as the bucket object key prefix.
func OSSObjectKey(filename string) string {
	return JoinObjectKey(viper.GetString("oss.path"), filename)
}

// SaveScreenshotLocal writes filename under screenshot-local-path when enabled.
func SaveScreenshotLocal(filename string, data []byte) error {
	if !viper.GetBool("save-screenshot-on-disk") {
		return nil
	}
	dir := viper.GetString("screenshot-local-path")
	if dir == "" {
		dir = "."
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, filename), data, 0644)
}
