//go:build e2e

package e2e

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type siteModel struct {
	Id   string `json:"id"`
	Name string `json:"name"`
	Url  string `json:"url"`
	Type string `json:"type"`
}

func TestE2E_SiteCRUDAndPersistence(t *testing.T) {
	mockSite := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer mockSite.Close()

	mcp := startAppWithMCP(t)

	// 1. Verify initial window and empty site list.
	winJSON := mcp.mustCallTool(t, "windows_list", nil)
	if !strings.Contains(winJSON, "window-1") {
		t.Fatalf("expected window-1 in windows_list, got: %s", winJSON)
	}

	sitesRaw := mcp.mustCallTool(t, "call_bound_method", map[string]any{
		"name": "main.Twsnmp.GetSites",
	})
	var sites []siteModel
	if err := json.Unmarshal([]byte(sitesRaw), &sites); err != nil {
		t.Fatalf("failed to unmarshal GetSites: %v (raw=%s)", err, sitesRaw)
	}
	if len(sites) != 0 {
		t.Fatalf("expected 0 initial sites, got %d", len(sites))
	}

	// 2. Click "登録" (3rd button in BottomNav) to open EditSite form.
	mcp.mustCallTool(t, "mouse_click", map[string]any{
		"selector": "div.fixed.bottom-0 button:nth-of-type(3)",
	})

	// 3. Fill EditSite form.
	mcp.mustCallTool(t, "keyboard_type", map[string]any{
		"selector": `input[placeholder*="サイト名"]`,
		"text":     "E2E Test Site",
		"delay_ms": 5,
	})
	mcp.mustCallTool(t, "keyboard_type", map[string]any{
		"selector": `input[type="url"]`,
		"text":     mockSite.URL,
		"delay_ms": 5,
	})
	mcp.mustCallTool(t, "keyboard_type", map[string]any{
		"selector": `input[placeholder*="ユーザーID"]`,
		"text":     "admin",
		"delay_ms": 5,
	})
	mcp.mustCallTool(t, "keyboard_type", map[string]any{
		"selector": `input[type="password"]`,
		"text":     "secret",
		"delay_ms": 5,
	})

	// 4. Save the site.
	mcp.mustCallTool(t, "mouse_click", map[string]any{
		"selector": "form button.bg-blue-700",
	})

	// 5. Verify site appears in the UI list and in Go backend state.
	waitUntil(t, 5*time.Second, func() bool {
		q, err := mcp.callTool("dom_query", map[string]any{"selector": "ul li"})
		return err == nil && strings.Contains(q, "E2E Test Site")
	}, "site did not appear in list UI")

	sitesRaw = mcp.mustCallTool(t, "call_bound_method", map[string]any{
		"name": "main.Twsnmp.GetSites",
	})
	if err := json.Unmarshal([]byte(sitesRaw), &sites); err != nil {
		t.Fatalf("failed to unmarshal GetSites after save: %v", err)
	}
	if len(sites) != 1 || sites[0].Name != "E2E Test Site" || sites[0].Url != mockSite.URL {
		t.Fatalf("unexpected sites after save: %+v", sites)
	}

	// 6. Delete the site via UI delete button.
	mcp.mustCallTool(t, "mouse_click", map[string]any{
		"selector": `button[title="削除"]`,
	})

	waitUntil(t, 5*time.Second, func() bool {
		raw, err := mcp.callTool("call_bound_method", map[string]any{"name": "main.Twsnmp.GetSites"})
		if err != nil {
			return false
		}
		var current []siteModel
		return json.Unmarshal([]byte(raw), &current) == nil && len(current) == 0
	}, "site was not deleted")
}

func TestE2E_ConfigDarkMode(t *testing.T) {
	mcp := startAppWithMCP(t)

	// Click "設定" (4th button in BottomNav).
	mcp.mustCallTool(t, "mouse_click", map[string]any{
		"selector": "div.fixed.bottom-0 button:nth-of-type(4)",
	})

	// Toggle Dark mode checkbox via its label.
	mcp.mustCallTool(t, "mouse_click", map[string]any{
		"selector": "form label.inline-flex",
	})

	// Save config.
	mcp.mustCallTool(t, "mouse_click", map[string]any{
		"selector": "form button.bg-blue-700",
	})

	// Verify dark class on <html> and IsDark() in Go backend.
	waitUntil(t, 5*time.Second, func() bool {
		res, err := mcp.callTool("js_eval", map[string]any{
			"js": "return document.documentElement.classList.contains('dark');",
		})
		return err == nil && strings.TrimSpace(res) == "true"
	}, "dark class was not applied to documentElement")

	isDark := mcp.mustCallTool(t, "call_bound_method", map[string]any{
		"name": "main.Twsnmp.IsDark",
	})
	if strings.TrimSpace(isDark) != "true" {
		t.Fatalf("expected IsDark() == true, got %s", isDark)
	}
}

func domHTML(mcp *mcpClient, selector string) (string, error) {
	raw, err := mcp.callTool("dom_html", map[string]any{
		"selector": selector,
	})
	if err != nil {
		return "", err
	}
	var payload struct {
		HTML string `json:"html"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return raw, nil
	}
	return payload.HTML, nil
}

func TestE2E_ConfigMapStylePresetsAndValidation(t *testing.T) {
	mcp := startAppWithMCP(t)

	// 1. Open Settings view ("設定" tab is 4th button in BottomNav).
	mcp.mustCallTool(t, "mouse_click", map[string]any{
		"selector": "div.fixed.bottom-0 button:nth-of-type(4)",
	})

	waitUntil(t, 5*time.Second, func() bool {
		html, err := domHTML(mcp, `[data-testid="map-style-preset-default-empty"]`)
		return err == nil && strings.Contains(html, `aria-pressed="true"`)
	}, "default-empty preset should initially be active")

	// 2. Click GSI Pale preset shortcut and verify textarea + active state update.
	mcp.mustCallTool(t, "mouse_click", map[string]any{
		"selector": `[data-testid="map-style-preset-gsi-pale"]`,
	})

	waitUntil(t, 5*time.Second, func() bool {
		val, err := mcp.callTool("js_eval", map[string]any{
			"js": "return document.getElementById('map-style')?.value || '';",
		})
		if err != nil || !strings.Contains(val, "cyberjapandata.gsi.go.jp/xyz/pale/") {
			return false
		}
		html, err := domHTML(mcp, `[data-testid="map-style-preset-gsi-pale"]`)
		return err == nil && strings.Contains(html, `aria-pressed="true"`)
	}, "GSI pale preset did not populate textarea and active state")

	// 3. Save config and verify GetMapStyle persists the JSON style.
	mcp.mustCallTool(t, "mouse_click", map[string]any{
		"selector": "form button.bg-blue-700",
	})

	waitUntil(t, 5*time.Second, func() bool {
		raw, err := mcp.callTool("call_bound_method", map[string]any{
			"name": "main.Twsnmp.GetMapStyle",
		})
		return err == nil && strings.Contains(raw, "cyberjapandata.gsi.go.jp/xyz/pale/")
	}, "GetMapStyle did not persist GSI pale JSON style")

	// 4. Switch to Map tab and verify MapLibre mounts with parsed JSON style.
	mcp.mustCallTool(t, "mouse_click", map[string]any{
		"selector": "div.fixed.bottom-0 button:nth-of-type(2)",
	})

	waitUntil(t, 5*time.Second, func() bool {
		q, err := mcp.callTool("dom_query", map[string]any{
			"selector": "canvas.maplibregl-canvas",
		})
		return err == nil && q != ""
	}, "MapLibre canvas did not mount with JSON map style")

	// 5. Return to Settings, clear via default-empty shortcut, type invalid JSON, and submit.
	mcp.mustCallTool(t, "mouse_click", map[string]any{
		"selector": "div.fixed.bottom-0 button:nth-of-type(4)",
	})
	mcp.mustCallTool(t, "mouse_click", map[string]any{
		"selector": `[data-testid="map-style-preset-default-empty"]`,
	})
	waitUntil(t, 5*time.Second, func() bool {
		val, err := mcp.callTool("js_eval", map[string]any{
			"js": "return document.getElementById('map-style')?.value ?? 'nonempty';",
		})
		return err == nil && val == ""
	}, "default-empty preset did not clear textarea")

	mcp.mustCallTool(t, "keyboard_type", map[string]any{
		"selector": "#map-style",
		"text":     "{ invalid_json ",
		"delay_ms": 5,
	})
	mcp.mustCallTool(t, "mouse_click", map[string]any{
		"selector": "form button.bg-blue-700",
	})

	waitUntil(t, 5*time.Second, func() bool {
		errHTML, err := domHTML(mcp, `[data-testid="map-style-error"]`)
		if err != nil || !strings.Contains(errHTML, "スタイルは、URLかJSONで指定してください。") {
			return false
		}
		invalid, err := mcp.callTool("js_eval", map[string]any{
			"js": "return document.getElementById('map-style')?.getAttribute('aria-invalid') || '';",
		})
		return err == nil && strings.Contains(invalid, "true")
	}, "validation error was not shown for invalid JSON map style")

	// Verify backend still holds the valid GSI style (save was blocked).
	persisted := mcp.mustCallTool(t, "call_bound_method", map[string]any{
		"name": "main.Twsnmp.GetMapStyle",
	})
	if !strings.Contains(persisted, "cyberjapandata.gsi.go.jp/xyz/pale/") {
		t.Fatalf("expected previously saved style to remain untouched, got: %s", persisted)
	}

	// 6. Click OSM Bright shortcut to fix error and save.
	mcp.mustCallTool(t, "mouse_click", map[string]any{
		"selector": `[data-testid="map-style-preset-osm-bright-ja"]`,
	})
	waitUntil(t, 5*time.Second, func() bool {
		invalid, err := mcp.callTool("js_eval", map[string]any{
			"js": "return document.getElementById('map-style')?.getAttribute('aria-invalid') || '';",
		})
		if err != nil || !strings.Contains(invalid, "false") {
			return false
		}
		_, err = domHTML(mcp, `[data-testid="map-style-error"]`)
		return err != nil
	}, "error state did not clear after selecting OSM Bright preset")

	mcp.mustCallTool(t, "mouse_click", map[string]any{
		"selector": "form button.bg-blue-700",
	})

	waitUntil(t, 5*time.Second, func() bool {
		raw, err := mcp.callTool("call_bound_method", map[string]any{
			"name": "main.Twsnmp.GetMapStyle",
		})
		return err == nil && strings.Contains(raw, "https://tile.openstreetmap.jp/styles/osm-bright-ja/style.json")
	}, "GetMapStyle did not update to OSM Bright URL")
}

func TestE2E_UpdaterUI(t *testing.T) {
	var currentBody atomic.Value
	currentBody.Store("E2E release notes for v0.1.1 (startup)")
	dummyDigest := fmt.Sprintf("%x", sha256.Sum256([]byte("dummy-release-artifact")))

	var serverURL string
	mockGitHub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/twsnmp/twsnmpmw/releases/latest":
			tag := "v0.1.1"
			body := currentBody.Load().(string)
			payload := map[string]any{
				"tag_name": tag,
				"name":     "Release " + tag,
				"body":     body,
				"assets": []map[string]any{
					{"id": 1, "name": "SHA256SUMS", "size": 256, "browser_download_url": serverURL + "/download/SHA256SUMS"},
					{"id": 2, "name": "twsnmpmw-darwin-arm64.zip", "size": 1024, "browser_download_url": serverURL + "/download/twsnmpmw-darwin-arm64.zip"},
					{"id": 3, "name": "twsnmpmw-windows-amd64.zip", "size": 1024, "browser_download_url": serverURL + "/download/twsnmpmw-windows-amd64.zip"},
					{"id": 4, "name": "twsnmpmw-linux-amd64.tar.gz", "size": 1024, "browser_download_url": serverURL + "/download/twsnmpmw-linux-amd64.tar.gz"},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(payload)
		case "/download/SHA256SUMS":
			w.Header().Set("Content-Type", "text/plain")
			fmt.Fprintf(w, "%s  twsnmpmw-darwin-arm64.zip\n%s  twsnmpmw-windows-amd64.zip\n%s  twsnmpmw-linux-amd64.tar.gz\n", dummyDigest, dummyDigest, dummyDigest)
		default:
			http.NotFound(w, r)
		}
	}))
	defer mockGitHub.Close()
	serverURL = mockGitHub.URL

	mcp := startAppWithMCPEnv(t, "TWSNMPMW_GITHUB_BASE_URL="+mockGitHub.URL)

	// 1. Startup background check should surface badge on Settings button.
	waitUntil(t, 10*time.Second, func() bool {
		q, err := mcp.callTool("dom_html", map[string]any{
			"selector": `[data-testid="update-badge"]`,
		})
		return err == nil && strings.Contains(q, "update-badge")
	}, "update badge did not appear after startup check")

	// 2. Open Settings view ("設定" tab is 4th button in BottomNav).
	mcp.mustCallTool(t, "mouse_click", map[string]any{
		"selector": "div.fixed.bottom-0 button:nth-of-type(4)",
	})

	// 3. Verify initial version v0.1.0, detected update v0.1.1, and apply button appear.
	waitUntil(t, 5*time.Second, func() bool {
		sec, err := mcp.callTool("dom_html", map[string]any{
			"selector": `[data-testid="updater-section"]`,
		})
		if err != nil || !strings.Contains(sec, "v0.1.1") || !strings.Contains(sec, "E2E release notes for v0.1.1 (startup)") {
			return false
		}
		ver, err := mcp.callTool("dom_html", map[string]any{
			"selector": `[data-testid="app-version"]`,
		})
		if err != nil || !strings.Contains(ver, "v0.1.0") {
			return false
		}
		btn, err := mcp.callTool("dom_html", map[string]any{
			"selector": `[data-testid="apply-update-btn"]`,
		})
		return err == nil && strings.Contains(btn, "アップデートを実施")
	}, "updater section did not render v0.1.0 version info and v0.1.1 update prompt")

	// 4. Refresh release details and verify manual check button updates UI and backend state.
	currentBody.Store("E2E release notes for v0.1.1 (manual)")
	mcp.mustCallTool(t, "mouse_click", map[string]any{
		"selector": `[data-testid="check-update-btn"]`,
	})

	waitUntil(t, 5*time.Second, func() bool {
		sec, err := mcp.callTool("dom_html", map[string]any{
			"selector": `[data-testid="updater-section"]`,
		})
		return err == nil && strings.Contains(sec, "E2E release notes for v0.1.1 (manual)")
	}, "manual check button did not refresh release info for v0.1.1")

	stateRaw := mcp.mustCallTool(t, "call_bound_method", map[string]any{
		"name": "main.Twsnmp.GetUpdateState",
	})
	var state struct {
		State           string `json:"state"`
		LatestVersion   string `json:"latestVersion"`
		UpdateAvailable bool   `json:"updateAvailable"`
	}
	if err := json.Unmarshal([]byte(stateRaw), &state); err != nil {
		t.Fatalf("failed to unmarshal GetUpdateState: %v", err)
	}
	if state.State != "available" || !state.UpdateAvailable || state.LatestVersion != "v0.1.1" {
		t.Fatalf("unexpected GetUpdateState result: %+v", state)
	}

	verRaw := mcp.mustCallTool(t, "call_bound_method", map[string]any{
		"name": "main.Twsnmp.GetVersion",
	})
	var ver struct {
		Version  string `json:"version"`
		Platform string `json:"platform"`
	}
	if err := json.Unmarshal([]byte(verRaw), &ver); err != nil {
		t.Fatalf("failed to unmarshal GetVersion: %v", err)
	}
	if ver.Version != "v0.1.0" || ver.Platform == "" {
		t.Fatalf("unexpected GetVersion result: %+v", ver)
	}
}

func waitUntil(t *testing.T, timeout time.Duration, cond func() bool, failMsg string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(150 * time.Millisecond)
	}
	t.Fatal(failMsg)
}
