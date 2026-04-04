package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type SendResult struct {
	StatusCode int
	Body       string
}

type ServerConfig struct {
	CollectionIntervalSeconds int    `json:"collection_interval_seconds"`
	MaxContainers             int    `json:"max_containers"`
	RetentionDays             int    `json:"retention_days"`
	OrgName                   string `json:"org_name"`
	PlanName                  string `json:"plan_name"`
}

type PreflightRequest struct {
	HostName       string `json:"host_name"`
	ContainerCount int    `json:"container_count"`
}

type PreflightResponse struct {
	CanProceed        bool   `json:"can_proceed"`
	Reason            string `json:"reason,omitempty"`
	CurrentContainers int    `json:"current_containers"`
	MaxContainers     int    `json:"max_containers"`
	CurrentHosts      int    `json:"current_hosts"`
	MaxHosts          int    `json:"max_hosts"`
}

type Sender struct {
	serverURL string
	token     string
	client    *http.Client
}

func NewSender(serverURL, token string) *Sender {
	return &Sender{
		serverURL: strings.TrimSuffix(strings.TrimSpace(serverURL), "/"),
		token:     strings.TrimSpace(token),
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (s *Sender) Send(ctx context.Context, reqBody IngestRequest) (*SendResult, error) {
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("cannot encode ingest payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.serverURL+"/v1/ingest", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("cannot create ingest request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	return &SendResult{StatusCode: resp.StatusCode, Body: strings.TrimSpace(string(respBody))}, nil
}

func (s *Sender) FetchConfig(ctx context.Context) (*ServerConfig, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.serverURL+"/v1/agent/config", nil)
	if err != nil {
		return nil, fmt.Errorf("cannot create config request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.token)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("config endpoint returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var envelope struct {
		Data ServerConfig `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, fmt.Errorf("cannot decode config response: %w", err)
	}
	return &envelope.Data, nil
}

func (s *Sender) Preflight(ctx context.Context, hostName string, containerCount int) (*PreflightResponse, error) {
	body, err := json.Marshal(PreflightRequest{HostName: hostName, ContainerCount: containerCount})
	if err != nil {
		return nil, fmt.Errorf("cannot encode preflight payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.serverURL+"/v1/agent/preflight", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("cannot create preflight request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("preflight endpoint returned %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}

	var envelope struct {
		Data PreflightResponse `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, fmt.Errorf("cannot decode preflight response: %w", err)
	}
	return &envelope.Data, nil
}
