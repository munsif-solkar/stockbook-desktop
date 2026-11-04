package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	maxReleaseBody     = 2 << 20
	maxChecksumSize    = 1 << 20
	maxUpdateDownload  = 250 << 20
	updateHTTPTimeout  = 45 * time.Second
	checksumAssetName  = "SHA256SUMS.txt"
	linuxUpdateAsset   = "Stockbook-x86_64.AppImage"
	windowsUpdateAsset = "stockbook.exe"
)

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

type githubRelease struct {
	TagName     string        `json:"tag_name"`
	Name        string        `json:"name"`
	HTMLURL     string        `json:"html_url"`
	Body        string        `json:"body"`
	PublishedAt string        `json:"published_at"`
	Assets      []githubAsset `json:"assets"`
}

var updateClient = &http.Client{Timeout: updateHTTPTimeout}

// CheckForUpdates compares the running version with the latest published
// GitHub release. It never changes anything on disk.
func (a *App) CheckForUpdates() (UpdateInfo, error) {
	info := UpdateInfo{CurrentVersion: appVersion}
	release, err := fetchLatestRelease()
	if err != nil {
		info.Error = friendlyUpdateFetchError(err)
		return info, nil
	}

	info.LatestVersion = strings.TrimPrefix(strings.TrimSpace(release.TagName), "v")
	info.ReleaseName = strings.TrimSpace(release.Name)
	info.ReleaseURL = release.HTMLURL
	info.PublishedAt = release.PublishedAt
	info.Notes = truncateText(strings.TrimSpace(release.Body), 4000)

	newer, err := isNewerVersion(release.TagName, appVersion)
	if err != nil {
		info.Error = "Could not compare the available version. Download the release manually."
		return info, nil
	}
	info.UpdateAvailable = newer
	if !newer {
		return info, nil
	}

	asset, err := selectUpdateAsset(release.Assets)
	if err != nil {
		info.Error = err.Error()
		return info, nil
	}
	info.AssetName = asset.Name
	info.AssetSize = asset.Size
	return info, nil
}

// ApplyUpdate downloads the latest release, verifies its published checksum,
// replaces the running executable, and restarts the app.
func (a *App) ApplyUpdate() (UpdateApplyResult, error) {
	if _, err := a.requireCompany(); err != nil {
		return UpdateApplyResult{}, err
	}
	if a.ctx == nil {
		return UpdateApplyResult{Error: "The app window is not ready yet. Try again in a moment."}, nil
	}

	a.opMu.Lock()
	defer a.opMu.Unlock()

	release, err := fetchLatestRelease()
	if err != nil {
		return UpdateApplyResult{Error: friendlyUpdateFetchError(err)}, nil
	}
	newer, err := isNewerVersion(release.TagName, appVersion)
	if err != nil {
		return UpdateApplyResult{Error: "Could not compare versions. Download the release manually."}, nil
	}
	if !newer {
		return UpdateApplyResult{Error: "Stockbook is already on its latest version."}, nil
	}
	asset, err := selectUpdateAsset(release.Assets)
	if err != nil {
		return UpdateApplyResult{Error: err.Error()}, nil
	}
	if !allowedDownloadURL(asset.BrowserDownloadURL) {
		return UpdateApplyResult{Error: "The release download address was not recognised. Download the update manually from GitHub."}, nil
	}

	target, err := updateTargetPath()
	if err != nil {
		return UpdateApplyResult{Error: err.Error()}, nil
	}
	dir := filepath.Dir(target)
	temp, err := os.CreateTemp(dir, ".stockbook-update-*.tmp")
	if err != nil {
		return UpdateApplyResult{Error: "Stockbook cannot write an update beside its executable. Move the app to a folder you can write to, or download the update manually."}, nil
	}
	tempPath := temp.Name()
	if err := temp.Close(); err != nil {
		_ = os.Remove(tempPath)
		return UpdateApplyResult{Error: "Could not prepare the update download."}, nil
	}
	defer os.Remove(tempPath)

	if err := downloadFile(asset.BrowserDownloadURL, tempPath, maxUpdateDownload); err != nil {
		return UpdateApplyResult{Error: friendlyUpdateDownloadError(err)}, nil
	}
	info, err := os.Stat(tempPath)
	if err != nil {
		return UpdateApplyResult{Error: "Could not read the downloaded update."}, nil
	}
	if asset.Size > 0 && info.Size() != asset.Size {
		return UpdateApplyResult{Error: "The downloaded update was incomplete. Try again."}, nil
	}

	expected, err := releaseChecksum(release.Assets, asset.Name)
	if err != nil {
		return UpdateApplyResult{Error: err.Error()}, nil
	}
	actual, err := fileSHA256(tempPath)
	if err != nil {
		return UpdateApplyResult{Error: "Could not verify the downloaded update."}, nil
	}
	if !strings.EqualFold(expected, actual) {
		return UpdateApplyResult{Error: "The downloaded update failed its checksum and was discarded. Try again or download it manually from GitHub."}, nil
	}
	if err := os.Chmod(tempPath, 0o755); err != nil && runtime.GOOS != "windows" {
		return UpdateApplyResult{Error: "Could not make the downloaded update executable."}, nil
	}

	if err := installUpdate(tempPath, target); err != nil {
		return UpdateApplyResult{Error: err.Error()}, nil
	}

	cmd := exec.Command(target)
	cmd.Dir = filepath.Dir(target)
	if err := cmd.Start(); err != nil {
		return UpdateApplyResult{
			Updated: true,
			Error:   "The update was installed but Stockbook could not restart automatically. Close and open it again.",
		}, nil
	}

	// Give the new process a moment to start before this one exits.
	go func() {
		time.Sleep(1500 * time.Millisecond)
		wailsruntime.Quit(a.ctx)
	}()

	return UpdateApplyResult{Updated: true, Restarting: true}, nil
}

func fetchLatestRelease() (*githubRelease, error) {
	request, err := http.NewRequest(http.MethodGet, releasesAPI, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "Stockbook-Desktop/"+appVersion)

	response, err := updateClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return nil, errors.New("no release found")
	case http.StatusForbidden, http.StatusTooManyRequests:
		return nil, errors.New("rate limited")
	default:
		return nil, fmt.Errorf("github returned %s", response.Status)
	}

	var release githubRelease
	decoder := json.NewDecoder(io.LimitReader(response.Body, maxReleaseBody))
	if err := decoder.Decode(&release); err != nil {
		return nil, err
	}
	if strings.TrimSpace(release.TagName) == "" {
		return nil, errors.New("release has no version tag")
	}
	return &release, nil
}

func selectUpdateAsset(assets []githubAsset) (githubAsset, error) {
	preferred := windowsUpdateAsset
	suffix := ".exe"
	if runtime.GOOS == "linux" {
		preferred = linuxUpdateAsset
		suffix = ".appimage"
	} else if runtime.GOOS != "windows" {
		return githubAsset{}, errors.New("Automatic updates are only available for the Linux and Windows releases. Download the new version from GitHub.")
	}

	for _, asset := range assets {
		if strings.EqualFold(asset.Name, preferred) {
			return asset, nil
		}
	}
	for _, asset := range assets {
		if strings.HasSuffix(strings.ToLower(asset.Name), suffix) {
			return asset, nil
		}
	}
	return githubAsset{}, errors.New("The latest release does not contain a download for this platform. Download it manually from GitHub.")
}

func releaseChecksum(assets []githubAsset, assetName string) (string, error) {
	var checksumAsset *githubAsset
	for i := range assets {
		if strings.EqualFold(assets[i].Name, checksumAssetName) {
			checksumAsset = &assets[i]
			break
		}
	}
	if checksumAsset == nil {
		return "", errors.New("The latest release does not publish checksums, so the automatic update was not installed. Download it manually from GitHub.")
	}
	if !allowedDownloadURL(checksumAsset.BrowserDownloadURL) {
		return "", errors.New("The checksum download address was not recognised.")
	}

	data, err := downloadBytes(checksumAsset.BrowserDownloadURL, maxChecksumSize)
	if err != nil {
		return "", errors.New("Could not download the update checksums. Try again.")
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		name := strings.TrimPrefix(fields[1], "*")
		if strings.EqualFold(filepath.Base(name), assetName) {
			return fields[0], nil
		}
	}
	return "", errors.New("The latest release has no checksum for this platform's download. Download it manually from GitHub.")
}

func allowedDownloadURL(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "https" {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	return host == "github.com" ||
		strings.HasSuffix(host, ".github.com") ||
		host == "githubusercontent.com" ||
		strings.HasSuffix(host, ".githubusercontent.com")
}

func downloadBytes(source string, limit int64) ([]byte, error) {
	if !allowedDownloadURL(source) {
		return nil, errors.New("untrusted download address")
	}
	response, err := updateClient.Get(source)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download returned %s", response.Status)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, errors.New("download is too large")
	}
	return data, nil
}

func downloadFile(source, target string, limit int64) error {
	if !allowedDownloadURL(source) {
		return errors.New("untrusted download address")
	}
	response, err := updateClient.Get(source)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned %s", response.Status)
	}
	if response.ContentLength > limit {
		return errors.New("download is too large")
	}

	file, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	written, err := io.Copy(file, io.LimitReader(response.Body, limit+1))
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if written > limit {
		return errors.New("download is too large")
	}
	return nil
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func updateTargetPath() (string, error) {
	if runtime.GOOS == "linux" {
		// AppImages run from a temporary mount, so os.Executable points inside
		// the mount. APPIMAGE names the real file the user launched.
		if appImage := strings.TrimSpace(os.Getenv("APPIMAGE")); appImage != "" {
			return filepath.Clean(appImage), nil
		}
	}
	executable, err := os.Executable()
	if err != nil {
		return "", errors.New("Could not locate the Stockbook executable.")
	}
	if resolved, err := filepath.EvalSymlinks(executable); err == nil {
		executable = resolved
	}
	if runtime.GOOS == "windows" {
		executable = strings.TrimSuffix(executable, ".old")
	}
	return executable, nil
}

func installUpdate(source, target string) error {
	if runtime.GOOS == "windows" {
		// Windows cannot overwrite a running executable, but it can rename it.
		// The old copy is deleted on the next launch.
		old := target + ".old"
		_ = os.Remove(old)
		if err := os.Rename(target, old); err != nil {
			return errors.New("Windows would not let Stockbook replace its running executable. Move the app to a folder you can write to, or download the update manually from GitHub.")
		}
		if err := os.Rename(source, target); err != nil {
			_ = os.Rename(old, target)
			return errors.New("Could not move the update into place. Download it manually from GitHub.")
		}
		return nil
	}

	if err := os.Rename(source, target); err != nil {
		return errors.New("Could not replace the Stockbook executable. Move the app to a folder you can write to, or download the update manually from GitHub.")
	}
	return nil
}

func friendlyUpdateFetchError(err error) string {
	switch {
	case err == nil:
		return ""
	case strings.Contains(err.Error(), "rate limited"):
		return "GitHub is limiting update checks right now. Try again later or download the release manually from GitHub."
	case strings.Contains(err.Error(), "no release"):
		return "No published release was found on GitHub."
	default:
		return "Could not reach GitHub. Check your internet connection and try again."
	}
}

func friendlyUpdateDownloadError(err error) string {
	if errors.Is(err, os.ErrPermission) {
		return "Stockbook could not write the update beside its executable. Move the app to a folder you can write to, or download the update manually."
	}
	return "Could not download the update. Check your internet connection and try again."
}

func truncateText(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit]) + "…"
}
