package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const defaultReleaseBaseURL = "https://kernus.app"

var ErrRestartScheduled = errors.New("update installed; restart scheduled")

type Client struct {
	baseURL string
	client  *http.Client
}

func NewClient(baseURL string) *Client {
	if v := strings.TrimSpace(baseURL); v == "" {
		baseURL = strings.TrimSpace(os.Getenv("KERNUS_RELEASE_BASE_URL"))
	}
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultReleaseBaseURL
	}
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) MaybeSelfUpdate(ctx context.Context, currentVersion string) (string, error) {
	currentVersion = strings.TrimSpace(currentVersion)
	if currentVersion == "" || strings.EqualFold(currentVersion, "dev") {
		return "", nil
	}
	if v := strings.TrimSpace(os.Getenv("KERNUS_DISABLE_AUTO_UPDATE")); v != "" && v != "0" && !strings.EqualFold(v, "false") {
		return "", nil
	}
	if _, ok := parseVersion(currentVersion); !ok {
		return "", nil
	}

	latestVersion, err := c.fetchLatestVersion(ctx)
	if err != nil {
		return "", err
	}
	if compareVersions(latestVersion, currentVersion) <= 0 {
		return "", nil
	}
	if err := c.applyUpdate(ctx); err != nil {
		return latestVersion, err
	}
	return latestVersion, nil
}

func (c *Client) fetchLatestVersion(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/agent/tag", nil)
	if err != nil {
		return "", err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("latest version endpoint returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	version := strings.TrimSpace(string(body))
	if version == "" {
		return "", fmt.Errorf("latest version endpoint returned empty response")
	}
	return version, nil
}

func (c *Client) applyUpdate(ctx context.Context) error {
	assetName, err := assetNameFor(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return err
	}

	checksums, err := c.fetchChecksums(ctx)
	if err != nil {
		return err
	}
	expectedChecksum, ok := checksums[assetName]
	if !ok {
		return fmt.Errorf("checksum for %s not found", assetName)
	}

	binary, err := c.downloadBinary(ctx)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(binary)
	if hex.EncodeToString(sum[:]) != strings.ToLower(expectedChecksum) {
		return fmt.Errorf("checksum mismatch for %s", assetName)
	}

	exePath, err := os.Executable()
	if err != nil {
		return err
	}
	exePath, err = filepath.Abs(exePath)
	if err != nil {
		return err
	}

	dir := filepath.Dir(exePath)
	tempPath := filepath.Join(dir, filepath.Base(exePath)+".download")
	if runtime.GOOS == "windows" {
		tempPath = exePath + ".new"
	}

	if err := os.WriteFile(tempPath, binary, 0o755); err != nil {
		return fmt.Errorf("write updated binary: %w", err)
	}

	if runtime.GOOS == "windows" {
		return c.scheduleWindowsRestart(exePath, tempPath)
	}
	return replaceAndRestart(exePath, tempPath)
}

func (c *Client) fetchChecksums(ctx context.Context) (map[string]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/agent/checksums", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("checksums endpoint returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	checksums := make(map[string]string)
	for _, line := range strings.Split(string(body), "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) < 2 {
			continue
		}
		checksums[fields[len(fields)-1]] = strings.ToLower(fields[0])
	}
	return checksums, nil
}

func (c *Client) downloadBinary(ctx context.Context) ([]byte, error) {
	q := url.Values{}
	q.Set("os", runtime.GOOS)
	q.Set("arch", runtime.GOARCH)
	downloadURL := c.baseURL + "/api/agent/download?" + q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download endpoint returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, nil
}

func assetNameFor(goos, goarch string) (string, error) {
	switch goos {
	case "windows":
		if goarch == "amd64" {
			return "kernus-windows-amd64.exe", nil
		}
	case "linux", "darwin":
		if goarch == "amd64" || goarch == "arm64" {
			return fmt.Sprintf("kernus-%s-%s", goos, goarch), nil
		}
	}
	return "", fmt.Errorf("unsupported auto-update target %s/%s", goos, goarch)
}

func replaceAndRestart(exePath, tempPath string) error {
	backupPath := exePath + ".bak"
	_ = os.Remove(backupPath)
	if err := os.Rename(exePath, backupPath); err != nil {
		return fmt.Errorf("move current binary aside: %w", err)
	}
	if err := os.Rename(tempPath, exePath); err != nil {
		_ = os.Rename(backupPath, exePath)
		return fmt.Errorf("install updated binary: %w", err)
	}
	_ = os.Remove(backupPath)
	cmd := exec.Command(exePath, os.Args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("restart updated binary: %w", err)
	}
	return ErrRestartScheduled
}

func (c *Client) scheduleWindowsRestart(exePath, newPath string) error {
	backupPath := exePath + ".old"
	scriptPath := exePath + ".update.ps1"
	script := windowsUpdateScript(exePath, newPath, backupPath, os.Args[1:])
	if err := os.WriteFile(scriptPath, []byte(script), 0o600); err != nil {
		return fmt.Errorf("write update helper script: %w", err)
	}

	cmd := exec.Command("powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", scriptPath)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start update helper: %w", err)
	}
	return ErrRestartScheduled
}

func windowsUpdateScript(exePath, newPath, backupPath string, args []string) string {
	quotedArgs := make([]string, 0, len(args))
	for _, arg := range args {
		quotedArgs = append(quotedArgs, "'"+strings.ReplaceAll(arg, "'", "''")+"'")
	}
	argList := "@()"
	if len(quotedArgs) > 0 {
		argList = "@(" + strings.Join(quotedArgs, ", ") + ")"
	}

	return fmt.Sprintf(`$ErrorActionPreference = 'SilentlyContinue'
$exe = '%s'
$new = '%s'
$bak = '%s'
for ($i = 0; $i -lt 20; $i++) {
  Start-Sleep -Milliseconds 500
  try {
    if (Test-Path $bak) { Remove-Item -Force $bak }
    Move-Item -Force $exe $bak
    Move-Item -Force $new $exe
    Start-Process -FilePath $exe -ArgumentList %s
    Start-Sleep -Milliseconds 500
    if (Test-Path $bak) { Remove-Item -Force $bak }
    Remove-Item -Force $PSCommandPath
    exit 0
  } catch {}
}
exit 1
`, escapePowerShell(exePath), escapePowerShell(newPath), escapePowerShell(backupPath), argList)
}

func escapePowerShell(v string) string {
	return strings.ReplaceAll(v, "'", "''")
}

func compareVersions(left, right string) int {
	lv, lok := parseVersion(left)
	rv, rok := parseVersion(right)
	if !lok || !rok {
		return strings.Compare(left, right)
	}
	for i := 0; i < len(lv); i++ {
		if lv[i] < rv[i] {
			return -1
		}
		if lv[i] > rv[i] {
			return 1
		}
	}
	return 0
}

func parseVersion(v string) ([3]int, bool) {
	var out [3]int
	v = strings.TrimSpace(strings.TrimPrefix(v, "v"))
	if v == "" {
		return out, false
	}
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return out, false
	}
	for i, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil {
			return out, false
		}
		out[i] = n
	}
	return out, true
}
