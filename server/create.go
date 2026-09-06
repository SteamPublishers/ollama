package server

import (
	"strings"

	"github.com/ollama/ollama/fs/ggml"
)

func hasEmbeddedCompatibilityTensors(f *ggml.GGML) bool {
	for _, t := range f.Tensors().Items() {
		if isEmbeddedCompatibilityTensor(t.Name) {
			return true
		}
	}
	return false
}

func isEmbeddedCompatibilityTensor(name string) bool {
	for _, prefix := range []string{"a.", "mm.", "mtp.", "s.", "v."} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}
