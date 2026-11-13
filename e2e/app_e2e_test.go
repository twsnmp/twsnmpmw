//go:build e2e

package e2e

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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
