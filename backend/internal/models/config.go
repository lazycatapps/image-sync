// Copyright (c) 2025 Lazycat Apps
// Licensed under the MIT License. See LICENSE file in the project root for details.

package models

// UserConfig represents user configuration for image sync
// Credentials (username/password) are base64 encoded by frontend for secure transmission
type UserConfig struct {
	SourceRegistry string `json:"sourceRegistry"` // Source image registry prefix
	DestRegistry   string `json:"destRegistry"`   // Destination image registry prefix
	SourceUsername string `json:"sourceUsername"` // Source registry username (base64 encoded)
	DestUsername   string `json:"destUsername"`   // Destination registry username (base64 encoded)
	SourcePassword string `json:"sourcePassword"` // Source registry password (base64 encoded)
	DestPassword   string `json:"destPassword"`   // Destination registry password (base64 encoded)
	SrcTLSVerify   bool   `json:"srcTLSVerify"`   // Source TLS verification
	DestTLSVerify  bool   `json:"destTLSVerify"`  // Destination TLS verification
}
