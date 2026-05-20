package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type ConsulConfig struct {
	Addr           string
	ServiceName    string
	ServiceAddress string
	HTTPAddr       string
}

type ConsulRegistration struct {
	client    *http.Client
	addr      string
	serviceID string
}

func (r *ConsulRegistration) Ping(ctx context.Context) error {
	if r == nil || r.client == nil || r.addr == "" || r.serviceID == "" {
		return fmt.Errorf("consul registration is not initialized")
	}
	checkURL := fmt.Sprintf("http://%s/v1/agent/services", r.addr)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, checkURL, nil)
	if err != nil {
		return err
	}
	resp, err := r.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("consul services lookup failed: %s", resp.Status)
	}
	var services map[string]json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&services); err != nil {
		return err
	}
	if _, ok := services[r.serviceID]; !ok {
		return fmt.Errorf("service %s is not registered", r.serviceID)
	}
	return nil
}

func (r *ConsulRegistration) Deregister(ctx context.Context) error {
	if r == nil || r.client == nil || r.addr == "" || r.serviceID == "" {
		return fmt.Errorf("consul registration is not initialized")
	}
	deregisterURL := fmt.Sprintf("http://%s/v1/agent/service/deregister/%s", r.addr, r.serviceID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, deregisterURL, nil)
	if err != nil {
		return err
	}
	resp, err := r.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("consul deregister failed: %s", resp.Status)
	}
	return nil
}

func RegisterConsulService(ctx context.Context, cfg ConsulConfig) (*ConsulRegistration, error) {
	if cfg.Addr == "" {
		return nil, fmt.Errorf("consul addr is required")
	}
	if cfg.ServiceName == "" {
		return nil, fmt.Errorf("service name is required")
	}
	if cfg.ServiceAddress == "" {
		return nil, fmt.Errorf("service address is required")
	}

	_, portText, err := net.SplitHostPort(cfg.HTTPAddr)
	if err != nil {
		if strings.HasPrefix(cfg.HTTPAddr, ":") {
			portText = strings.TrimPrefix(cfg.HTTPAddr, ":")
		} else {
			return nil, fmt.Errorf("parse http addr %q: %w", cfg.HTTPAddr, err)
		}
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		return nil, fmt.Errorf("parse http port %q: %w", portText, err)
	}

	serviceID := fmt.Sprintf("%s-%s-%d", cfg.ServiceName, strings.ReplaceAll(cfg.ServiceAddress, ".", "-"), port)
	payload := map[string]any{
		"ID":      serviceID,
		"Name":    cfg.ServiceName,
		"Address": cfg.ServiceAddress,
		"Port":    port,
		"Check": map[string]string{
			"HTTP":                           fmt.Sprintf("http://%s:%d/readyz", cfg.ServiceAddress, port),
			"Interval":                       "10s",
			"Timeout":                        "2s",
			"DeregisterCriticalServiceAfter": "1m",
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 5 * time.Second}
	registerURL := fmt.Sprintf("http://%s/v1/agent/service/register", cfg.Addr)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, registerURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("consul register failed: %s", resp.Status)
	}

	return &ConsulRegistration{client: client, addr: cfg.Addr, serviceID: serviceID}, nil
}
