package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const maxExportSize = 64 << 20

// ExportData asks for a destination and writes a JSON copy of the signed-in
// workspace. The file contains only inventory data: items, purchases, and
// sales. It never contains account details, email addresses, or passwords.
func (a *App) ExportData() (ExportResult, error) {
	if _, err := a.requireCompany(); err != nil {
		return ExportResult{}, err
	}
	if a.ctx == nil {
		return ExportResult{Error: "The app window is not ready yet. Try again in a moment."}, nil
	}

	a.opMu.Lock()
	defer a.opMu.Unlock()

	path, err := wailsruntime.SaveFileDialog(a.ctx, wailsruntime.SaveDialogOptions{
		Title:                "Export Stockbook data",
		DefaultFilename:      "stockbook-data-" + time.Now().Format("2006-01-02-150405") + ".json",
		CanCreateDirectories: true,
		Filters: []wailsruntime.FileFilter{
			{DisplayName: "Stockbook data (*.json)", Pattern: "*.json"},
		},
	})
	if err != nil {
		return ExportResult{}, err
	}
	if strings.TrimSpace(path) == "" {
		return ExportResult{Canceled: true}, nil
	}
	if !strings.HasSuffix(strings.ToLower(path), ".json") {
		path += ".json"
	}

	size, err := a.exportToFile(path)
	if err != nil {
		return ExportResult{Error: friendlyExportError(err)}, nil
	}
	return ExportResult{Path: path, Size: size}, nil
}

func friendlyExportError(err error) string {
	if errors.Is(err, os.ErrPermission) {
		return "Stockbook could not write to that folder. Choose another location."
	}
	return "Could not export your data: " + err.Error()
}

// exportToFile exports the workspace and writes it atomically to target.
func (a *App) exportToFile(target string) (int64, error) {
	clean, err := a.exportData()
	if err != nil {
		return 0, err
	}
	data, err := buildDataFile(clean)
	if err != nil {
		return 0, err
	}

	dir := filepath.Dir(target)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return 0, err
	}

	temp, err := os.CreateTemp(dir, ".stockbook-export-*.json")
	if err != nil {
		return 0, err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)

	if err := temp.Chmod(0o600); err != nil {
		temp.Close()
		return 0, err
	}
	if _, err := temp.Write(data); err != nil {
		temp.Close()
		return 0, err
	}
	if err := temp.Close(); err != nil {
		return 0, err
	}
	if err := replaceFile(tempPath, target); err != nil {
		return 0, err
	}

	info, err := os.Stat(target)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// replaceFile moves source over target. Windows will not rename over an
// existing file, so the destination is removed after the source is ready.
func replaceFile(source, target string) error {
	if runtime.GOOS == "windows" {
		if err := os.Remove(target); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return os.Rename(source, target)
}

// exportData reads the active backend into the shared inventory shape.
func (a *App) exportData() (*cleanData, error) {
	if a.currentMode() == modeLocal {
		return a.exportLocalData()
	}

	var raw json.RawMessage
	if err := a.api.rpc(a.context(), "stockbook_export", map[string]any{}, &raw); err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return nil, errors.New("the backend returned an empty export")
	}
	// Validate and re-serialise so even an older backend response cannot leak
	// account fields or malformed values into the file.
	return parseDataFile(raw)
}

// ImportData asks for a Stockbook JSON export and replaces the signed-in
// workspace with it. The current workspace is snapshotted first, so a bad
// import never destroys the only copy. The account and session stay intact.
func (a *App) ImportData() (ImportResult, error) {
	if _, err := a.requireCompany(); err != nil {
		return ImportResult{}, err
	}
	if a.ctx == nil {
		return ImportResult{Error: "The app window is not ready yet. Try again in a moment."}, nil
	}

	a.opMu.Lock()
	defer a.opMu.Unlock()

	path, err := wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Choose a Stockbook data file",
		Filters: []wailsruntime.FileFilter{
			{DisplayName: "Stockbook data (*.json)", Pattern: "*.json"},
		},
	})
	if err != nil {
		return ImportResult{}, err
	}
	if strings.TrimSpace(path) == "" {
		return ImportResult{Canceled: true}, nil
	}

	safety, err := a.importFromFile(path)
	if err != nil {
		return ImportResult{Error: friendlyImportError(err)}, nil
	}
	return ImportResult{Imported: true, SafetyBackup: safety}, nil
}

func friendlyImportError(err error) string {
	switch {
	case errors.Is(err, os.ErrPermission):
		return "Stockbook could not read that file. Choose another location."
	case strings.Contains(err.Error(), "not a Stockbook data file"):
		return err.Error()
	default:
		return "Could not import the data: " + err.Error()
	}
}

// importFromFile validates source, snapshots the current workspace, and
// imports the file. It returns the safety snapshot path.
func (a *App) importFromFile(source string) (string, error) {
	info, err := os.Stat(source)
	if err != nil {
		return "", err
	}
	if info.Size() > maxExportSize {
		return "", invalidData("the file is too large")
	}

	data, err := os.ReadFile(source)
	if err != nil {
		return "", err
	}

	clean, err := parseDataFile(data)
	if err != nil {
		return "", err
	}

	safety, err := a.writeSafetySnapshot()
	if err != nil {
		return "", err
	}

	if err := a.importData(clean); err != nil {
		return "", err
	}
	return safety, nil
}

// importData replaces the active backend's inventory with validated records.
func (a *App) importData(clean *cleanData) error {
	if a.currentMode() == modeLocal {
		return a.importLocalData(clean)
	}

	payload, err := buildDataFile(clean)
	if err != nil {
		return err
	}
	var outcome rpcOutcome
	if err := a.api.rpc(a.context(), "stockbook_import",
		map[string]any{"p_data": json.RawMessage(payload)}, &outcome); err != nil {
		return err
	}
	if !outcome.OK {
		message := outcome.Error
		if message == "" {
			message = "the data could not be imported"
		}
		return errors.New(message)
	}
	return nil
}

// writeSafetySnapshot keeps a pre-import copy outside the normal export
// location so a mistaken import can always be undone.
func (a *App) writeSafetySnapshot() (string, error) {
	dir := filepath.Join(a.dataDir, "backups")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	target := filepath.Join(dir, "import-safety-"+time.Now().Format("20060102-150405.000")+".json")
	if _, err := a.exportToFile(target); err != nil {
		return "", err
	}
	return target, nil
}
