package updater

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const latestReleaseURL = "https://api.github.com/repos/marvinscham/nextclone/releases/latest"

type Release struct {
	Version string
	URL     string
	Asset   Asset
}

type Asset struct {
	Name string
	URL  string
}

type githubRelease struct {
	TagName string        `json:"tag_name"`
	HTMLURL string        `json:"html_url"`
	Assets  []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func Check(ctx context.Context, current string) (*Release, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, latestReleaseURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "Nextclone")

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("check release: %s", resp.Status)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}
	if !newerVersion(release.TagName, current) {
		return nil, nil
	}

	asset, ok := matchingAsset(release.Assets)
	if !ok {
		return nil, fmt.Errorf("no update asset available for %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	return &Release{Version: release.TagName, URL: release.HTMLURL, Asset: asset}, nil
}

func Install(ctx context.Context, release *Release) error {
	if release == nil {
		return errors.New("no release selected")
	}
	assetPath, err := download(ctx, release.Asset)
	if err != nil {
		return err
	}

	switch runtime.GOOS {
	case "linux":
		if strings.HasSuffix(release.Asset.Name, ".deb") {
			return installDeb(assetPath)
		}
		return installZip(assetPath)
	case "windows":
		return installWindowsZip(assetPath)
	default:
		return fmt.Errorf("self-update is not supported on %s", runtime.GOOS)
	}
}

func matchingAsset(assets []githubAsset) (Asset, bool) {
	suffixes := []string{}
	switch runtime.GOOS {
	case "linux":
		exe, _ := os.Executable()
		if strings.HasPrefix(exe, "/usr/lib/nextclone/") {
			suffixes = []string{"_linux_amd64.deb", "_linux_amd64.zip"}
		} else {
			suffixes = []string{"_linux_amd64.zip", "_linux_amd64.deb"}
		}
	case "windows":
		suffixes = []string{"_windows_amd64.zip"}
	default:
		return Asset{}, false
	}
	if runtime.GOARCH != "amd64" {
		return Asset{}, false
	}
	for _, suffix := range suffixes {
		for _, asset := range assets {
			if strings.HasSuffix(asset.Name, suffix) {
				return Asset{Name: asset.Name, URL: asset.BrowserDownloadURL}, true
			}
		}
	}
	return Asset{}, false
}

func newerVersion(latest, current string) bool {
	latest = strings.TrimPrefix(strings.TrimSpace(latest), "v")
	current = strings.TrimPrefix(strings.TrimSpace(current), "v")
	latestParts := versionParts(latest)
	currentParts := versionParts(current)
	for i := 0; i < len(latestParts); i++ {
		if latestParts[i] > currentParts[i] {
			return true
		}
		if latestParts[i] < currentParts[i] {
			return false
		}
	}
	return false
}

func versionParts(v string) [3]int {
	var parts [3]int
	fields := strings.Split(v, ".")
	for i := 0; i < len(parts) && i < len(fields); i++ {
		field := fields[i]
		if cut := strings.IndexFunc(field, func(r rune) bool { return r < '0' || r > '9' }); cut >= 0 {
			field = field[:cut]
		}
		parts[i], _ = strconv.Atoi(field)
	}
	return parts
}

func download(ctx context.Context, asset Asset) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, asset.URL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Nextclone")

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download update: %s", resp.Status)
	}

	path := filepath.Join(os.TempDir(), asset.Name)
	file, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	if _, err := io.Copy(file, resp.Body); err != nil {
		return "", err
	}
	return path, nil
}

func installDeb(path string) error {
	if _, err := exec.LookPath("pkexec"); err != nil {
		return errors.New("pkexec is required to install the downloaded .deb update")
	}
	cmd := exec.Command("pkexec", "dpkg", "-i", path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("install update: %w\n%s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func installZip(path string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	tmp, err := unzip(path)
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	if err := copyFile(filepath.Join(tmp, "nextclone-linux-amd64"), exe, 0o755); err != nil {
		return err
	}
	rclonePath := os.Getenv("NEXTCLONE_RCLONE_PATH")
	if rclonePath == "" {
		rclonePath = filepath.Join(filepath.Dir(exe), "rclone-linux-amd64")
	}
	if _, err := os.Stat(rclonePath); err == nil {
		return copyFile(filepath.Join(tmp, "rclone-linux-amd64"), rclonePath, 0o755)
	}
	return nil
}

func installWindowsZip(path string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	workDir := filepath.Dir(exe)
	script := filepath.Join(os.TempDir(), "nextclone-update.ps1")
	content := fmt.Sprintf(`$ErrorActionPreference = 'Stop'
$pidToWait = %d
$zip = %q
$targetExe = %q
$targetDir = %q
Wait-Process -Id $pidToWait -ErrorAction SilentlyContinue
$tmp = Join-Path $env:TEMP ('nextclone-update-' + [guid]::NewGuid())
New-Item -ItemType Directory -Force -Path $tmp | Out-Null
Expand-Archive -Path $zip -DestinationPath $tmp -Force
Copy-Item (Join-Path $tmp 'nextclone-windows-amd64.exe') $targetExe -Force
Copy-Item (Join-Path $tmp 'rclone.exe') (Join-Path $targetDir 'rclone.exe') -Force
Start-Process $targetExe
Remove-Item $tmp -Recurse -Force
`, os.Getpid(), path, exe, workDir)
	if err := os.WriteFile(script, []byte(content), 0o600); err != nil {
		return err
	}
	return exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", script).Start()
}

func unzip(path string) (string, error) {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return "", err
	}
	defer reader.Close()

	dir, err := os.MkdirTemp("", "nextclone-update-")
	if err != nil {
		return "", err
	}
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}
		name := filepath.Base(file.Name)
		if name == "." || name == string(filepath.Separator) {
			continue
		}
		in, err := file.Open()
		if err != nil {
			return "", err
		}
		outPath := filepath.Join(dir, name)
		out, err := os.OpenFile(outPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, file.FileInfo().Mode())
		if err != nil {
			in.Close()
			return "", err
		}
		_, copyErr := io.Copy(out, in)
		closeErr := out.Close()
		in.Close()
		if copyErr != nil {
			return "", copyErr
		}
		if closeErr != nil {
			return "", closeErr
		}
	}
	return dir, nil
}

func copyFile(src, dst string, perm os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp := dst + ".nextclone-update"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, perm)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Chmod(tmp, perm); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, dst)
}
