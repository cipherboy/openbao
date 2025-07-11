// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package mfa

import (
	"errors"
	"fmt"

	"google.golang.org/protobuf/proto"
)

func (c *Config) Clone() (*Config, error) {
	if c == nil {
		return nil, errors.New("nil config")
	}

	marshaledConfig, err := proto.Marshal(c)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	var clonedConfig Config
	err = proto.Unmarshal(marshaledConfig, &clonedConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &clonedConfig, nil
}

func (c *MFAEnforcementConfig) Clone() (*MFAEnforcementConfig, error) {
	if c == nil {
		return nil, errors.New("nil config")
	}

	marshaledConfig, err := proto.Marshal(c)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	var clonedConfig MFAEnforcementConfig
	err = proto.Unmarshal(marshaledConfig, &clonedConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &clonedConfig, nil
}
