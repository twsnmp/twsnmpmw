//go:build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

type mcpClient struct {
	endpoint string
	http     *http.Client
	cmd      *exec.Cmd
	nextID   atomic.Int64
}

type jsonRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int64  `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type callToolResult struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	IsError bool `json:"isError"`
}

func reserveFreePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to reserve free port: %v", err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

func resolveAppBinary(t *testing.T) string {
	t.Helper()
	if env := os.Getenv("TWSNMPMW_E2E_BIN"); env != "" {
		return env
	}
	abs, err := filepath.Abs(filepath.Join("..", "bin", "twsnmpmw-mcp"))
	if err != nil {
		t.Fatalf("failed to resolve binary path: %v", err)
	}
	if _, err := os.Stat(abs); err != nil {
		t.Fatalf("MCP binary not found at %s (run `task build:mcp` first): %v", abs, err)
	}
	return abs
}

// startAppWithMCP launches the app built with -tags mcp using an isolated config directory.
func startAppWithMCP(t *testing.T) *mcpClient {
	t.Helper()

	binPath := resolveAppBinary(t)
	port := reserveFreePort(t)
	configDir := t.TempDir()

	cmd := exec.Command(binPath)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("WAILS_MCP_PORT=%d", port),
		"WAILS_MCP_HOST=127.0.0.1",
		"WAILS_MCP_HIDE_CURSOR=1",
		"TWSNMPMW_CONFIG_DIR="+configDir,
	)

	var logBuf bytes.Buffer
	cmd.Stdout = &logBuf
	cmd.Stderr = &logBuf

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start %s: %v", binPath, err)
	}

	c := &mcpClient{
		endpoint: fmt.Sprintf("http://127.0.0.1:%d/mcp", port),
		http:     &http.Client{Timeout: 15 * time.Second},
		cmd:      cmd,
	}

	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Signal(os.Interrupt)
			done := make(chan error, 1)
			go func() { done <- cmd.Wait() }()
			select {
			case <-done:
			case <-time.After(3 * time.Second):
				_ = cmd.Process.Kill()
				<-done
			}
		}
		if t.Failed() {
			t.Logf("app output:\n%s", logBuf.String())
		}
	})

	statusURL := fmt.Sprintf("http://127.0.0.1:%d/", port)
	c.waitUntilReady(t, statusURL)
	c.initialize(t)
	c.waitForWindow(t)

	return c
}

func (c *mcpClient) waitUntilReady(t *testing.T, statusURL string) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := c.http.Get(statusURL)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(150 * time.Millisecond)
	}
	t.Fatalf("MCP server did not become ready at %s within timeout", statusURL)
}

func (c *mcpClient) initialize(t *testing.T) {
	t.Helper()
	_, err := c.rpc("initialize", map[string]any{
		"protocolVersion": "2025-06-18",
		"capabilities":    map[string]any{},
		"clientInfo": map[string]any{
			"name":    "twsnmpmw-e2e",
			"version": "1.0.0",
		},
	})
	if err != nil {
		t.Fatalf("MCP initialize failed: %v", err)
	}
}

func (c *mcpClient) waitForWindow(t *testing.T) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		text, err := c.callTool("windows_list", map[string]any{})
		if err == nil {
			var windows []map[string]any
			if json.Unmarshal([]byte(text), &windows) == nil && len(windows) > 0 {
				// Also ensure DOM is mounted.
				if _, domErr := c.callTool("dom_query", map[string]any{"selector": "div.fixed.bottom-0"}); domErr == nil {
					return
				}
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("application window or DOM did not become ready within timeout")
}

func (c *mcpClient) rpc(method string, params any) (json.RawMessage, error) {
	id := c.nextID.Add(1)
	payload, err := json.Marshal(jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, c.endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected HTTP %d: %s", resp.StatusCode, string(body))
	}

	var rpcResp jsonRPCResponse
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return nil, fmt.Errorf("decode rpc response: %w (body=%s)", err, string(body))
	}
	if rpcResp.Error != nil {
		return nil, fmt.Errorf("rpc error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}
	return rpcResp.Result, nil
}

func (c *mcpClient) callTool(name string, args map[string]any) (string, error) {
	if args == nil {
		args = map[string]any{}
	}
	raw, err := c.rpc("tools/call", map[string]any{
		"name":      name,
		"arguments": args,
	})
	if err != nil {
		return "", err
	}

	var res callToolResult
	if err := json.Unmarshal(raw, &res); err != nil {
		return "", fmt.Errorf("decode tool result: %w", err)
	}
	if len(res.Content) == 0 {
		if res.IsError {
			return "", errors.New("tool returned empty error")
		}
		return "", nil
	}
	text := res.Content[0].Text
	if res.IsError {
		return text, fmt.Errorf("tool %s failed: %s", name, text)
	}
	return text, nil
}

func (c *mcpClient) mustCallTool(t *testing.T, name string, args map[string]any) string {
	t.Helper()
	text, err := c.callTool(name, args)
	if err != nil {
		t.Fatalf("callTool(%s) failed: %v", name, err)
	}
	return text
}
