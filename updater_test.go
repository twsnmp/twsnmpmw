package main

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/github"
)

func TestResolveVersionInfo(t *testing.T) {
	tests := []struct {
		name        string
		rawVersion  string
		rawCommit   string
		wantVersion string
		wantSemVer  string
		wantDisplay string
	}{
		{
			name:        "tag with v prefix and commit",
			rawVersion:  "v1.2.3",
			rawCommit:   "abc1234",
			wantVersion: "v1.2.3",
			wantSemVer:  "1.2.3",
			wantDisplay: "v1.2.3 (abc1234)",
		},
		{
			name:        "tag without v prefix",
			rawVersion:  "2.0.0-rc.1",
			rawCommit:   "deadbee",
			wantVersion: "v2.0.0-rc.1",
			wantSemVer:  "2.0.0-rc.1",
			wantDisplay: "v2.0.0-rc.1 (deadbee)",
		},
		{
			name:        "strips legacy commit suffix if present",
			rawVersion:  "v0.1.0(cafebabe)",
			rawCommit:   "cafebabe",
			wantVersion: "v0.1.0",
			wantSemVer:  "0.1.0",
			wantDisplay: "v0.1.0 (cafebabe)",
		},
		{
			name:        "empty version defaults safely",
			rawVersion:  "",
			rawCommit:   "abc1234",
			wantVersion: "v0.0.0",
			wantSemVer:  "0.0.0",
			wantDisplay: "v0.0.0 (abc1234)",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveVersionInfo(tc.rawVersion, tc.rawCommit)
			if got.Version != tc.wantVersion {
				t.Fatalf("Version = %q, want %q", got.Version, tc.wantVersion)
			}
			if got.SemVer != tc.wantSemVer {
				t.Fatalf("SemVer = %q, want %q", got.SemVer, tc.wantSemVer)
			}
			if got.Display != tc.wantDisplay {
				t.Fatalf("Display = %q, want %q", got.Display, tc.wantDisplay)
			}
		})
	}
}

func TestMatchReleaseAssetForMode(t *testing.T) {
	assets := []github.ReleaseAsset{
		{Name: "SHA256SUMS"},
		{Name: "twsnmpmw-windows-amd64-installer.exe"},
		{Name: "twsnmpmw-windows-amd64.msix"},
		{Name: "twsnmpmw-windows-amd64.zip"},
		{Name: "twsnmpmw-darwin-arm64.dmg"},
		{Name: "twsnmpmw-darwin-arm64.pkg"},
		{Name: "twsnmpmw-darwin-arm64.zip"},
		{Name: "twsnmpmw-linux-amd64.deb"},
		{Name: "twsnmpmw-linux-amd64.rpm"},
		{Name: "twsnmpmw-linux-amd64.AppImage"},
		{Name: "twsnmpmw-linux-amd64.tar.gz"},
	}

	cases := []struct {
		name       string
		req        updater.CheckRequest
		inAppImage bool
		wantName   string
	}{
		{
			name:     "windows picks portable zip over installer and msix",
			req:      updater.CheckRequest{Platform: "windows", Arch: "amd64"},
			wantName: "twsnmpmw-windows-amd64.zip",
		},
		{
			name:     "darwin arm64 picks zip over dmg and pkg",
			req:      updater.CheckRequest{Platform: "darwin", Arch: "arm64"},
			wantName: "twsnmpmw-darwin-arm64.zip",
		},
		{
			name:     "linux normal binary picks tar.gz over deb/rpm/AppImage",
			req:      updater.CheckRequest{Platform: "linux", Arch: "amd64"},
			wantName: "twsnmpmw-linux-amd64.tar.gz",
		},
		{
			name:       "linux inside AppImage prefers AppImage",
			req:        updater.CheckRequest{Platform: "linux", Arch: "amd64"},
			inAppImage: true,
			wantName:   "twsnmpmw-linux-amd64.AppImage",
		},
		{
			name:     "darwin amd64 rejects arm64-only asset",
			req:      updater.CheckRequest{Platform: "darwin", Arch: "amd64"},
			wantName: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			idx := matchReleaseAssetForMode(tc.req, assets, tc.inAppImage)
			if tc.wantName == "" {
				if idx != -1 {
					t.Fatalf("expected -1, got idx=%d (%s)", idx, assets[idx].Name)
				}
				return
			}
			if idx < 0 || idx >= len(assets) {
				t.Fatalf("expected asset %q, got invalid idx=%d", tc.wantName, idx)
			}
			if assets[idx].Name != tc.wantName {
				t.Fatalf("picked %q, want %q", assets[idx].Name, tc.wantName)
			}
		})
	}
}

func TestIsLoopbackURL(t *testing.T) {
	if !isLoopbackURL("http://127.0.0.1:8080") {
		t.Fatal("expected 127.0.0.1 to be allowed")
	}
	if !isLoopbackURL("http://localhost:9000") {
		t.Fatal("expected localhost to be allowed")
	}
	if !isLoopbackURL("https://[::1]:443") {
		t.Fatal("expected ::1 to be allowed")
	}
	if isLoopbackURL("https://example.com") {
		t.Fatal("expected remote host to be rejected")
	}
	if isLoopbackURL("file:///tmp/test") {
		t.Fatal("expected non-http scheme to be rejected")
	}
}

type testUpdaterHost struct {
	mu     sync.Mutex
	events []string
}

func (h *testUpdaterHost) Emit(name string, _ ...any) bool {
	h.mu.Lock()
	h.events = append(h.events, name)
	h.mu.Unlock()
	return true
}

func (h *testUpdaterHost) OnEvent(_ string, _ func(any)) func() { return func() {} }
func (h *testUpdaterHost) OpenWindow(_ updater.WindowOptions) updater.WindowHandle {
	return nil
}
func (h *testUpdaterHost) Quit() {}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestUpdateManagerCheckAndProgressOrdering(t *testing.T) {
	var archiveBuf bytes.Buffer
	zw := zip.NewWriter(&archiveBuf)
	f, err := zw.Create("twsnmpmw")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write([]byte("binary-payload")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	assetBytes := archiveBuf.Bytes()
	sum := sha256.Sum256(assetBytes)

	baseURL := "http://127.0.0.1:18080"
	releaseStatus := http.StatusOK
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/twsnmp/twsnmpmw/releases/latest":
			if releaseStatus != http.StatusOK {
				http.Error(w, "upstream error", releaseStatus)
				return
			}
			assetName := "twsnmpmw-darwin-arm64.zip"
			winName := "twsnmpmw-windows-amd64.zip"
			linuxName := "twsnmpmw-linux-amd64.tar.gz"
			payload := map[string]any{
				"tag_name": "v0.1.1",
				"name":     "Release v0.1.1",
				"body":     "Bug fixes and self-update support",
				"assets": []map[string]any{
					{"id": 1, "name": "SHA256SUMS", "size": 128, "browser_download_url": baseURL + "/download/SHA256SUMS"},
					{"id": 2, "name": winName, "size": len(assetBytes), "browser_download_url": baseURL + "/download/" + winName},
					{"id": 3, "name": linuxName, "size": len(assetBytes), "browser_download_url": baseURL + "/download/" + linuxName},
					{"id": 4, "name": "twsnmpmw-darwin-arm64.dmg", "size": 999, "browser_download_url": baseURL + "/download/dmg"},
					{"id": 5, "name": assetName, "size": len(assetBytes), "browser_download_url": baseURL + "/download/" + assetName},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(payload)
		case "/download/SHA256SUMS":
			fmt.Fprintf(w, "%x  twsnmpmw-windows-amd64.zip\n%x  twsnmpmw-linux-amd64.tar.gz\n%x  twsnmpmw-darwin-arm64.zip\n", sum, sum, sum)
		default:
			_, _ = w.Write(assetBytes)
		}
	})

	httpClient := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if strings.HasPrefix(r.URL.String(), baseURL) {
				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, r)
				resp := rec.Result()
				resp.Request = r
				return resp, nil
			}
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(strings.NewReader("not found")),
				Header:     make(http.Header),
				Request:    r,
			}, nil
		}),
	}

	// Verify background check error stays silent
	releaseStatus = http.StatusInternalServerError
	backend := updater.New(&testUpdaterHost{})
	ver := resolveVersionInfo("v0.1.0", "abc1234")
	mgr := newUpdateManagerWithBackend(nil, backend, ver, baseURL)
	mgr.httpClient = httpClient
	if err := mgr.init(); err != nil {
		t.Fatalf("mgr.init failed: %v", err)
	}

	bgState := mgr.runCheck(context.Background(), true)
	if bgState.State != "idle" || bgState.Error != "" {
		t.Fatalf("expected silent idle state after background error, got %+v", bgState)
	}

	// Verify manual check error is surfaced
	errState := mgr.CheckUpdate()
	if errState.State != "error" || errState.Error == "" {
		t.Fatalf("expected error state on manual check failure, got %+v", errState)
	}

	// Verify successful check detects newer version & matches platform zip/tar.gz
	releaseStatus = http.StatusOK
	availState := mgr.CheckUpdate()
	if availState.State != "available" || !availState.UpdateAvailable || availState.LatestVersion != "v0.1.1" {
		t.Fatalf("expected available v0.1.1, got %+v", availState)
	}
	if availState.ArtifactName == "twsnmpmw-darwin-arm64.dmg" {
		t.Fatalf("must never pick .dmg asset, got %q", availState.ArtifactName)
	}

	// Verify monotonic phase ordering prevents stale progress overwriting verifying
	mgr.mu.Lock()
	mgr.activeOp = "apply"
	mgr.state.State = string(updater.StateDownloading)
	mgr.mu.Unlock()

	mgr.onDownloadProgress(50, 100)
	if st := mgr.State(); st.ProgressPercent != 50 {
		t.Fatalf("expected 50%% progress, got %+v", st)
	}

	mgr.advanceApplyPhase(string(updater.StateVerifying), 100)
	mgr.onDownloadProgress(60, 100) // stale out-of-order progress event
	if st := mgr.State(); st.State != string(updater.StateVerifying) || st.ProgressPercent != 100 {
		t.Fatalf("expected state to remain verifying at 100%%, got %+v", st)
	}
}
