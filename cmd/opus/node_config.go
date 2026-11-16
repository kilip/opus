package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	opusDirName    = ".opus"
	nodeConfigName = "node.json"
)

// NodeConfig stores worker connection credentials and master node URL.
type NodeConfig struct {
	NodeID     string `json:"nodeId"`
	Credential string `json:"credential"`
	MasterURL  string `json:"masterUrl"`
}

// opusHomeDir returns the path to ~/.opus, checking $HOME then os.UserHomeDir().
func opusHomeDir() (string, error) {
	if home := os.Getenv("HOME"); home != "" {
		return filepath.Join(home, opusDirName), nil
	}
	dir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home directory: %w", err)
	}
	return filepath.Join(dir, opusDirName), nil
}

// nodeConfigPath returns the path to node.json within the given directory.
func nodeConfigPath(dir string) string {
	return filepath.Join(dir, nodeConfigName)
}

// defaultNodeConfigPath resolves ~/.opus/node.json.
func defaultNodeConfigPath() (string, error) {
	dir, err := opusHomeDir()
	if err != nil {
		return "", err
	}
	return nodeConfigPath(dir), nil
}

// readNodeConfig reads and parses NodeConfig from the specified file path.
func readNodeConfig(path string) (*NodeConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg NodeConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse node config: %w", err)
	}
	if cfg.NodeID == "" || cfg.Credential == "" || cfg.MasterURL == "" {
		return nil, errors.New("invalid node config: missing required fields")
	}
	return &cfg, nil
}

// writeNodeConfig writes the given NodeConfig to the specified path with mode 0600,
// ensuring the directory is created with mode 0700.
func writeNodeConfig(path string, cfg *NodeConfig) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create config directory %s: %w", dir, err)
	}
	if err := os.Chmod(dir, 0700); err != nil {
		return fmt.Errorf("chmod config directory %s: %w", dir, err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal node config: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write node config to %s: %w", path, err)
	}
	if err := os.Chmod(path, 0600); err != nil {
		return fmt.Errorf("chmod node config %s: %w", path, err)
	}
	return nil
}
