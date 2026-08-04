package domain

import (
	"errors"
	"strings"
)

const (
	StatusProcessing = "processing"
	StatusReady      = "ready"
	StatusFailed     = "failed"
)

var VariantWidths = [...]int{320, 640, 1280}

type Asset struct {
	ID             string
	Status         string
	OriginalKey    string
	OriginalMIME   string
	OriginalWidth  int
	OriginalHeight int
	OriginalBytes  int64
	SHA256         string
	AltText        string
	Decorative     bool
}

func (asset Asset) Validate() error {
	if asset.OriginalKey == "" || asset.OriginalMIME == "" ||
		asset.OriginalWidth < 1 || asset.OriginalHeight < 1 ||
		asset.OriginalBytes < 1 || len(asset.SHA256) != 64 {
		return errors.New("media asset metadata is incomplete")
	}
	if asset.Decorative {
		if asset.AltText != "" {
			return errors.New("decorative media must have empty alt text")
		}
		return nil
	}
	if strings.TrimSpace(asset.AltText) != asset.AltText || asset.AltText == "" || len(asset.AltText) > 300 {
		return errors.New("informative media requires canonical alt text")
	}
	return nil
}

type Variant struct {
	Width, Height int
	ObjectKey     string
	MIME          string
	Bytes         int64
	SHA256        string
}
