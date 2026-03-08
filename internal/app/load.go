package app

import (
	"bytes"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

func LoadCampaign(path string) (Campaign, error) {
	var campaign Campaign

	payload, err := os.ReadFile(path)
	if err != nil {
		return campaign, fmt.Errorf("read campaign file: %w", err)
	}

	decoder := yaml.NewDecoder(bytes.NewReader(payload))
	decoder.KnownFields(true)
	if err := decoder.Decode(&campaign); err != nil {
		return campaign, fmt.Errorf("decode campaign file: %w", err)
	}

	return campaign, nil
}
