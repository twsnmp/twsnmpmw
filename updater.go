package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/github"
)

const (
	updateRepository  = "twsnmp/twsnmpmw"
	checksumAssetName = "SHA256SUMS"
	updateStateEvent  = "twsnmp:updater:state"

	defaultCheckTimeout    = 15 * time.Second
	defaultDownloadTimeout = 5 * time.Minute
	progressEmitInterval   = 200 * time.Millisecond
)

// VersionInfo holds build version metadata separated for UI display and SemVer updater comparison.
type VersionInfo struct {
	Version  string `json:"version"`
	Commit   string `json:"commit"`
	Display  string `json:"display"`
	SemVer   string `json:"semVer"`
	Platform string `json:"platform"`
	Arch     string `json:"arch"`
}

// UpdateState represents the current self-update lifecycle snapshot sent to the frontend.
type UpdateState struct {
	Seq             uint64 `json:"seq"`
	State           string `json:"state"`
	CurrentVersion  string `json:"currentVersion"`
	LatestVersion   string `json:"latestVersion"`
	ReleaseName     string `json:"releaseName"`
	ReleaseNotes    string `json:"releaseNotes"`
	ArtifactName    string `json:"artifactName"`
	ArtifactSize    int64  `json:"artifactSize"`
	ProgressWritten int64  `json:"progressWritten"`
	ProgressTotal   int64  `json:"progressTotal"`
	ProgressPercent int    `json:"progressPercent"`
	Error           string `json:"error"`
	UpdateAvailable bool   `json:"updateAvailable"`
	CheckedOnce     bool   `json:"checkedOnce"`
}

type updaterBackend interface {
	Init(cfg updater.Config) error
	Check(ctx context.Context) (*updater.Release, error)
	DownloadAndInstall(ctx context.Context) error
	Restart(ctx context.Context) error
	State() updater.State
}

// updateManager coordinates Wails v3 Updater calls, serializes concurrent operations,
// and publishes strictly ordered UpdateState snapshots to the frontend.
type updateManager struct {
	version    VersionInfo
	app        *application.App
	backend    updaterBackend
	baseURL    string
	httpClient *http.Client

	opMu       sync.Mutex
	mu         sync.RWMutex
	state      UpdateState
	activeOp   string
	lastEmitAt time.Time
	lastPct    int
}

func resolveVersionInfo(rawVersion, rawCommit string) VersionInfo {
	v := strings.TrimSpace(rawVersion)
	v = strings.TrimPrefix(v, "v")
	v = strings.TrimPrefix(v, "V")
	if idx := strings.IndexByte(v, '('); idx > 0 {
		v = strings.TrimSpace(v[:idx])
	}
	if v == "" {
		v = "0.0.0"
	}

	c := strings.TrimSpace(rawCommit)
	if c == "" || c == "unknown" {
		c = readBuildRevision()
	}

	tag := "v" + v
	display := tag
	if c != "" {
		display = fmt.Sprintf("%s (%s)", tag, c)
	}

	return VersionInfo{
		Version:  tag,
		Commit:   c,
		Display:  display,
		SemVer:   v,
		Platform: runtime.GOOS,
		Arch:     runtime.GOARCH,
	}
}

func readBuildRevision() string {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	for _, s := range bi.Settings {
		if s.Key == "vcs.revision" && s.Value != "" {
			if len(s.Value) > 7 {
				return s.Value[:7]
			}
			return s.Value
		}
	}
	return ""
}

func newUpdateManager(app *application.App, version VersionInfo) *updateManager {
	var backend updaterBackend
	if app != nil && app.Updater != nil {
		backend = app.Updater
	}
	return newUpdateManagerWithBackend(app, backend, version, githubBaseURLFromEnv())
}

func newUpdateManagerWithBackend(app *application.App, backend updaterBackend, version VersionInfo, baseURL string) *updateManager {
	return &updateManager{
		version: version,
		app:     app,
		backend: backend,
		baseURL: baseURL,
		state: UpdateState{
			State:          string(updater.StateIdle),
			CurrentVersion: version.Display,
		},
		lastPct: -1,
	}
}

func githubBaseURLFromEnv() string {
	raw := strings.TrimSpace(os.Getenv("TWSNMPMW_GITHUB_BASE_URL"))
	if raw == "" || !isLoopbackURL(raw) {
		return ""
	}
	return raw
}

func isLoopbackURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return false
	}
	host := strings.ToLower(u.Hostname())
	return host == "127.0.0.1" || host == "localhost" || host == "::1"
}

func (m *updateManager) init() error {
	if m.backend == nil {
		m.setStateError("アップデーターが利用できません")
		return updater.ErrNotConfigured
	}

	providerCfg := github.Config{
		Repository:    updateRepository,
		BaseURL:       m.baseURL,
		AssetMatcher:  matchReleaseAsset,
		ChecksumAsset: checksumAssetName,
		HTTPClient:    m.httpClient,
	}
	provider, err := github.New(providerCfg)
	if err != nil {
		m.setStateError(err.Error())
		return err
	}

	cfg := updater.Config{
		CurrentVersion: m.version.SemVer,
		Providers:      []updater.Provider{provider},
		Window:         updater.WindowNone,
	}
	if err := m.backend.Init(cfg); err != nil {
		m.setStateError(err.Error())
		return err
	}

	if m.app != nil {
		m.subscribeEvents()
	}
	return nil
}

func (m *updateManager) subscribeEvents() {
	m.app.Event.On(updater.EventDownloadProgress, func(e *application.CustomEvent) {
		if e == nil {
			return
		}
		var written, total int64
		switch p := e.Data.(type) {
		case updater.Progress:
			written, total = p.Written, p.Total
		case *updater.Progress:
			if p == nil {
				return
			}
			written, total = p.Written, p.Total
		default:
			return
		}
		m.onDownloadProgress(written, total)
	})

	m.app.Event.On(updater.EventVerifying, func(_ *application.CustomEvent) {
		m.advanceApplyPhase(string(updater.StateVerifying), 100)
	})

	m.app.Event.On(updater.EventInstalling, func(_ *application.CustomEvent) {
		m.advanceApplyPhase(string(updater.StateInstalling), 100)
	})
}

func (m *updateManager) startStartupCheck(delay time.Duration) {
	go func() {
		if delay > 0 {
			time.Sleep(delay)
		}
		m.runCheck(context.Background(), true)
	}()
}

func (m *updateManager) State() UpdateState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.state
}

func (m *updateManager) CheckUpdate() UpdateState {
	return m.runCheck(context.Background(), false)
}

func (m *updateManager) runCheck(ctx context.Context, background bool) UpdateState {
	m.opMu.Lock()
	defer m.opMu.Unlock()

	if m.backend == nil {
		return m.State()
	}

	m.mu.Lock()
	if background && m.state.CheckedOnce {
		snap := m.state
		m.mu.Unlock()
		return snap
	}
	prevState := m.state
	m.activeOp = "check"
	if !background || !m.state.UpdateAvailable {
		m.state.State = string(updater.StateChecking)
		m.state.Error = ""
		m.state.ProgressWritten = 0
		m.state.ProgressTotal = 0
		m.state.ProgressPercent = 0
	}
	m.state.Seq++
	checkingSnap := m.state
	m.mu.Unlock()

	if !background {
		m.emitState(checkingSnap)
	}

	checkCtx, cancel := context.WithTimeout(ctx, defaultCheckTimeout)
	defer cancel()

	rel, err := m.backend.Check(checkCtx)

	m.mu.Lock()
	m.activeOp = ""
	if err != nil {
		if background {
			// Preserve clean UI when running offline or behind a closed network.
			log.Printf("background update check skipped: %v", err)
			m.state.State = prevState.State
			if m.state.State == string(updater.StateChecking) || m.state.State == "" {
				m.state.State = string(updater.StateIdle)
			}
			m.state.Error = prevState.Error
			snap := m.state
			m.mu.Unlock()
			return snap
		}
		m.state.CheckedOnce = true
		m.state.State = string(updater.StateError)
		m.state.Error = err.Error()
		m.state.Seq++
		snap := m.state
		m.mu.Unlock()
		m.emitState(snap)
		return snap
	}

	m.state.CheckedOnce = true
	m.state.Error = ""
	if rel == nil {
		m.state.State = string(updater.StateUpToDate)
		m.state.UpdateAvailable = false
		m.state.LatestVersion = ""
		m.state.ReleaseName = ""
		m.state.ReleaseNotes = ""
		m.state.ArtifactName = ""
		m.state.ArtifactSize = 0
	} else {
		m.state.State = string(updater.StateAvailable)
		m.state.UpdateAvailable = true
		m.state.LatestVersion = normalizeVersionTag(rel.Version)
		m.state.ReleaseName = rel.Name
		m.state.ReleaseNotes = rel.Notes
		m.state.ArtifactName = rel.Artifact.Filename
		m.state.ArtifactSize = rel.Artifact.Size
	}
	m.state.Seq++
	snap := m.state
	m.mu.Unlock()

	m.emitState(snap)
	return snap
}

func (m *updateManager) ApplyUpdate() error {
	m.opMu.Lock()
	defer m.opMu.Unlock()

	if m.backend == nil {
		err := updater.ErrNotConfigured
		m.setStateError(err.Error())
		return err
	}

	if m.backend.State() == updater.StateReady {
		return m.restartNow()
	}

	m.mu.Lock()
	m.activeOp = "apply"
	m.state.State = string(updater.StateDownloading)
	m.state.Error = ""
	m.state.ProgressWritten = 0
	m.state.ProgressTotal = m.state.ArtifactSize
	m.state.ProgressPercent = 0
	m.lastPct = -1
	m.lastEmitAt = time.Time{}
	m.state.Seq++
	startSnap := m.state
	m.mu.Unlock()
	m.emitState(startSnap)

	dlCtx, cancel := context.WithTimeout(context.Background(), defaultDownloadTimeout)
	defer cancel()

	if err := m.backend.DownloadAndInstall(dlCtx); err != nil {
		m.mu.Lock()
		m.activeOp = ""
		m.state.State = string(updater.StateError)
		m.state.Error = err.Error()
		m.state.Seq++
		errSnap := m.state
		m.mu.Unlock()
		m.emitState(errSnap)
		return err
	}

	m.mu.Lock()
	m.activeOp = ""
	m.state.State = string(updater.StateReady)
	m.state.ProgressPercent = 100
	if m.state.ProgressTotal > 0 {
		m.state.ProgressWritten = m.state.ProgressTotal
	}
	m.state.Error = ""
	m.state.Seq++
	readySnap := m.state
	m.mu.Unlock()
	m.emitState(readySnap)

	return m.restartNow()
}

func (m *updateManager) restartNow() error {
	if err := m.backend.Restart(context.Background()); err != nil {
		m.setStateError(err.Error())
		return err
	}
	return nil
}

func (m *updateManager) onDownloadProgress(written, total int64) {
	now := time.Now()
	m.mu.Lock()
	if m.activeOp != "apply" || phaseRank(m.state.State) > phaseRank(string(updater.StateDownloading)) {
		m.mu.Unlock()
		return
	}

	pct := 0
	if total > 0 {
		pct = int((written * 100) / total)
		if pct > 100 {
			pct = 100
		}
	}

	shouldEmit := pct != m.lastPct || (total > 0 && written >= total) || now.Sub(m.lastEmitAt) >= progressEmitInterval
	m.state.State = string(updater.StateDownloading)
	m.state.ProgressWritten = written
	m.state.ProgressTotal = total
	m.state.ProgressPercent = pct
	if !shouldEmit {
		m.mu.Unlock()
		return
	}
	m.lastPct = pct
	m.lastEmitAt = now
	m.state.Seq++
	snap := m.state
	m.mu.Unlock()

	m.emitState(snap)
}

func (m *updateManager) advanceApplyPhase(nextState string, pct int) {
	m.mu.Lock()
	if m.activeOp != "apply" || phaseRank(nextState) <= phaseRank(m.state.State) {
		m.mu.Unlock()
		return
	}
	m.state.State = nextState
	if pct >= 0 {
		m.state.ProgressPercent = pct
	}
	m.state.Seq++
	snap := m.state
	m.mu.Unlock()

	m.emitState(snap)
}

func (m *updateManager) setStateError(msg string) {
	m.mu.Lock()
	m.activeOp = ""
	m.state.State = string(updater.StateError)
	m.state.Error = msg
	m.state.Seq++
	snap := m.state
	m.mu.Unlock()

	m.emitState(snap)
}

func (m *updateManager) emitState(snap UpdateState) {
	if m.app != nil && m.app.Event != nil {
		m.app.Event.Emit(updateStateEvent, snap)
	}
}

func normalizeVersionTag(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")
	v = strings.TrimPrefix(v, "V")
	if v == "" {
		return ""
	}
	return "v" + v
}

func phaseRank(state string) int {
	switch updater.State(state) {
	case updater.StateChecking:
		return 1
	case updater.StateUpToDate, updater.StateAvailable:
		return 2
	case updater.StateDownloading:
		return 3
	case updater.StateVerifying:
		return 4
	case updater.StateInstalling:
		return 5
	case updater.StateReady:
		return 6
	default:
		return 0
	}
}

func matchReleaseAsset(req updater.CheckRequest, assets []github.ReleaseAsset) int {
	return matchReleaseAssetForMode(req, assets, isRunningAppImage())
}

func matchReleaseAssetForMode(req updater.CheckRequest, assets []github.ReleaseAsset, inAppImage bool) int {
	plat := strings.ToLower(strings.TrimSpace(req.Platform))
	arch := strings.ToLower(strings.TrimSpace(req.Arch))

	var predicates []func(name string) bool
	switch plat {
	case "darwin":
		predicates = []func(string) bool{
			func(n string) bool { return matchesPlatform(n, "darwin") && matchesArch(n, arch) && strings.HasSuffix(n, ".zip") },
			func(n string) bool { return matchesPlatform(n, "darwin") && strings.Contains(n, "universal") && strings.HasSuffix(n, ".zip") },
			func(n string) bool { return matchesPlatform(n, "darwin") && matchesArch(n, arch) && isTarGz(n) },
		}
	case "windows":
		predicates = []func(string) bool{
			func(n string) bool { return matchesPlatform(n, "windows") && matchesArch(n, arch) && strings.HasSuffix(n, ".zip") },
			func(n string) bool { return matchesPlatform(n, "windows") && matchesArch(n, arch) && strings.HasSuffix(n, ".exe") },
		}
	case "linux":
		if inAppImage {
			predicates = []func(string) bool{
				func(n string) bool { return matchesPlatform(n, "linux") && matchesArch(n, arch) && strings.HasSuffix(n, ".appimage") },
				func(n string) bool { return matchesPlatform(n, "linux") && matchesArch(n, arch) && isTarGz(n) },
			}
		} else {
			predicates = []func(string) bool{
				func(n string) bool { return matchesPlatform(n, "linux") && matchesArch(n, arch) && isTarGz(n) },
				func(n string) bool {
					return matchesPlatform(n, "linux") && matchesArch(n, arch) && !strings.HasSuffix(n, ".appimage") && !strings.HasSuffix(n, ".zip")
				},
			}
		}
	default:
		return github.DefaultAssetMatcher(req, assets)
	}

	for _, match := range predicates {
		for i, a := range assets {
			name := strings.ToLower(a.Name)
			if isIgnoredReleaseAsset(name) {
				continue
			}
			if match(name) {
				return i
			}
		}
	}
	return -1
}

func isRunningAppImage() bool {
	img, mount := os.Getenv("APPIMAGE"), os.Getenv("APPDIR")
	if img == "" || mount == "" {
		return false
	}
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	cleanMount := filepath.Clean(mount) + string(filepath.Separator)
	return strings.HasPrefix(exe, cleanMount)
}

func isTarGz(lowerName string) bool {
	return strings.HasSuffix(lowerName, ".tar.gz") || strings.HasSuffix(lowerName, ".tgz")
}

func matchesPlatform(lowerName, plat string) bool {
	switch plat {
	case "darwin":
		return strings.Contains(lowerName, "darwin") || strings.Contains(lowerName, "macos")
	default:
		return strings.Contains(lowerName, plat)
	}
}

func matchesArch(lowerName, arch string) bool {
	if arch == "" || strings.Contains(lowerName, arch) {
		return true
	}
	if arch == "amd64" {
		return strings.Contains(lowerName, "x86_64") || strings.Contains(lowerName, "x64")
	}
	if arch == "arm64" {
		return strings.Contains(lowerName, "aarch64")
	}
	return false
}

func isIgnoredReleaseAsset(lowerName string) bool {
	ignoredSuffixes := []string{
		".sig", ".asc", ".sha256", ".sha512", ".sums", ".checksum", ".checksums",
		".dmg", ".pkg", ".deb", ".rpm", ".msix", ".msi", ".pkg.tar.zst",
	}
	for _, s := range ignoredSuffixes {
		if strings.HasSuffix(lowerName, s) {
			return true
		}
	}
	if strings.Contains(lowerName, "-installer.") || strings.Contains(lowerName, "_installer.") || lowerName == "installer.exe" {
		return true
	}
	for _, token := range []string{"checksums", "sha256sums", "sha512sums"} {
		if lowerName == token || strings.HasPrefix(lowerName, token+".") {
			return true
		}
	}
	return false
}
