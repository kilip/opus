package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/kilip/opus/internal/node"
	"github.com/kilip/opus/internal/platform/httpx"
	"github.com/kilip/opus/internal/worker"
	"github.com/spf13/cobra"
)

func newJoinCmd() *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:   "join <url>",
		Short: "Join an Opus cluster as a worker node",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runJoin(cmd.Context(), args[0], name)
		},
	}

	hostname, _ := os.Hostname()
	cmd.Flags().StringVar(&name, "name", hostname, "node name (defaults to hostname)")
	return cmd
}

func parseJoinURL(rawURL string) (string, string, error) {
	if rawURL == "" {
		return "", "", errors.New("join URL is required")
	}
	if !strings.Contains(rawURL, "://") {
		rawURL = "http://" + rawURL
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", "", fmt.Errorf("invalid join URL: %w", err)
	}
	if u.Host == "" {
		return "", "", errors.New("invalid join URL: missing host")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", "", fmt.Errorf("unsupported URL scheme: %s (must be http or https)", u.Scheme)
	}

	p := strings.TrimRight(u.Path, "/")
	idx := strings.LastIndex(p, "/join/")
	if idx == -1 {
		return "", "", errors.New("invalid join URL: expected format <master-url>/join/<token>")
	}

	basePath := p[:idx]
	token := p[idx+len("/join/"):]
	if token == "" {
		return "", "", errors.New("invalid join URL: missing token")
	}

	masterURL := fmt.Sprintf("%s://%s%s", u.Scheme, u.Host, basePath)
	return masterURL, token, nil
}

func runJoin(ctx context.Context, rawURL, name string) error {
	configPath, err := defaultNodeConfigPath()
	if err != nil {
		return err
	}

	// Pre-flight check: if node.json already exists, exit non-zero without network call.
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("node is already joined (%s already exists); run 'opus worker' or remove the file to rejoin", configPath)
	} else if !errors.Is(err, os.ErrNotExist) && !os.IsNotExist(err) {
		return fmt.Errorf("check node config %s: %w", configPath, err)
	}

	masterURL, token, err := parseJoinURL(rawURL)
	if err != nil {
		return err
	}

	if name == "" {
		if h, err := os.Hostname(); err == nil && h != "" {
			name = h
		} else {
			name = "worker"
		}
	}

	netInfo := worker.DetectNetInfo(masterURL)
	joinURL := strings.TrimRight(masterURL, "/") + "/api/nodes/join"
	joinReq := node.JoinNodeRequest{
		Token:      token,
		Name:       name,
		Hostname:   netInfo.Hostname,
		IPAddress:  netInfo.IPAddress,
		MACAddress: netInfo.MACAddress,
	}
	body, err := json.Marshal(joinReq)
	if err != nil {
		return fmt.Errorf("marshal join request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, joinURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create join request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("unreachable master at %s: %w", masterURL, err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read master response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		var errEnvelope httpx.ErrorResponse
		_ = json.Unmarshal(respBytes, &errEnvelope)
		errMsg := errEnvelope.Error
		if errMsg == "" {
			errMsg = strings.TrimSpace(string(respBytes))
		}

		switch resp.StatusCode {
		case http.StatusUnauthorized:
			return fmt.Errorf("join rejected (401 Unauthorized): invalid, expired, or already used token")
		case http.StatusConflict:
			return fmt.Errorf("join rejected (409 Conflict): node name %q is already in use; specify a different name with --name", name)
		case http.StatusBadRequest:
			return fmt.Errorf("join rejected (400 Bad Request): %s", errMsg)
		case http.StatusTooManyRequests:
			return fmt.Errorf("join rejected (429 Too Many Requests): rate limit exceeded")
		default:
			return fmt.Errorf("join failed with status %d: %s", resp.StatusCode, errMsg)
		}
	}

	var envelope httpx.Response[node.JoinNodeResponse]
	if err := json.Unmarshal(respBytes, &envelope); err != nil {
		return fmt.Errorf("decode join response: %w", err)
	}
	if envelope.Data.NodeID == "" || envelope.Data.Credential == "" {
		return errors.New("invalid join response: missing nodeId or credential")
	}

	nodeCfg := &NodeConfig{
		NodeID:     envelope.Data.NodeID,
		Credential: envelope.Data.Credential,
		MasterURL:  masterURL,
	}
	if err := writeNodeConfig(configPath, nodeCfg); err != nil {
		return fmt.Errorf("write node config: %w", err)
	}

	w := worker.New(worker.Config{
		MasterURL:  nodeCfg.MasterURL,
		NodeID:     nodeCfg.NodeID,
		Credential: nodeCfg.Credential,
		Logger:     slog.Default(),
	})
	return w.Run(ctx)
}
