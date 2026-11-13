package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const maxBackupSize = 64 << 20

// BackupDatabase asks for a destination and writes a JSON snapshot of the
// signed-in workspace from Supabase. The file contains every item, purchase,
// and sale for that workspace.
func (a *App) BackupDatabase() (BackupResult, error) {
	if _, err := a.requireCompany(); err != nil {
		return BackupResult{}, err
	}
	if a.ctx == nil {
		return BackupResult{Error: "The app window is not ready yet. Try again in a moment."}, nil
	}

	a.opMu.Lock()
	defer a.opMu.Unlock()

	path, err := wailsruntime.SaveFileDialog(a.ctx, wailsruntime.SaveDialogOptions{
		Title:                "Save Stockbook backup",
		DefaultFilename:      "stockbook-backup-" + time.Now().Format("2006-01-02-150405") + ".json",
		CanCreateDirectories: true,
		Filters: []wailsruntime.FileFilter{
			{DisplayName: "Stockbook backup (*.json)", Pattern: "*.json"},
		},
	})
	if err != nil {
		return BackupResult{}, err
	}
	if strings.TrimSpace(path) == "" {
		return BackupResult{Canceled: true}, nil
	}
	if !strings.HasSuffix(strings.ToLower(path), ".json") {
		path += ".json"
	}

	size, err := a.backupToFile(path)
	if err != nil {
		return BackupResult{Error: friendlyBackupError(err)}, nil
	}
	return BackupResult{Path: path, Size: size}, nil
}

func friendlyBackupError(err error) string {
	if errors.Is(err, os.ErrPermission) {
		return "Stockbook could not write to that folder. Choose another location."
	}
	return "Could not create the backup: " + err.Error()
}

// backupToFile exports the workspace and writes it atomically to target.
func (a *App) backupToFile(target string) (int64, error) {
	var raw json.RawMessage
	if err := a.api.rpc(a.context(), "stockbook_export", map[string]any{}, &raw); err != nil {
		return 0, err
	}
	if len(raw) == 0 {
		return 0, errors.New("the backend returned an empty backup")
	}

	var pretty bytes.Buffer
	if err := json.Indent(&pretty, raw, "", "  "); err != nil {
		return 0, err
	}
	pretty.WriteByte('\n')

	dir := filepath.Dir(target)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return 0, err
	}

	temp, err := os.CreateTemp(dir, ".stockbook-backup-*.json")
	if err != nil {
		return 0, err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)

	if err := temp.Chmod(0o600); err != nil {
		temp.Close()
		return 0, err
	}
	if _, err := temp.Write(pretty.Bytes()); err != nil {
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

type backupEnvelope struct {
	Format    string          `json:"format"`
	Version   int             `json:"version"`
	Products  json.RawMessage `json:"products"`
	Purchases json.RawMessage `json:"purchases"`
	Sales     json.RawMessage `json:"sales"`
}

func isJSONArray(raw json.RawMessage) bool {
	trimmed := bytes.TrimSpace(raw)
	return len(trimmed) > 1 && trimmed[0] == '['
}

// RestoreDatabase asks for a Stockbook backup and replaces the signed-in
// workspace in Supabase with it. The current workspace is snapshotted first,
// so a bad restore never destroys the only copy.
func (a *App) RestoreDatabase() (RestoreResult, error) {
	if _, err := a.requireCompany(); err != nil {
		return RestoreResult{}, err
	}
	if a.ctx == nil {
		return RestoreResult{Error: "The app window is not ready yet. Try again in a moment."}, nil
	}

	a.opMu.Lock()
	defer a.opMu.Unlock()

	path, err := wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Choose a Stockbook backup",
		Filters: []wailsruntime.FileFilter{
			{DisplayName: "Stockbook backup (*.json)", Pattern: "*.json"},
		},
	})
	if err != nil {
		return RestoreResult{}, err
	}
	if strings.TrimSpace(path) == "" {
		return RestoreResult{Canceled: true}, nil
	}

	safety, err := a.restoreFromFile(path)
	if err != nil {
		return RestoreResult{Error: friendlyRestoreError(err)}, nil
	}
	return RestoreResult{Restored: true, SafetyBackup: safety}, nil
}

func friendlyRestoreError(err error) string {
	switch {
	case errors.Is(err, os.ErrPermission):
		return "Stockbook could not read that file. Choose another location."
	case strings.Contains(err.Error(), "not a Stockbook backup"):
		return err.Error()
	default:
		return "Could not restore the backup: " + err.Error()
	}
}

// restoreFromFile validates source, snapshots the current workspace, imports
// the backup, and signs out. It returns the safety snapshot path.
func (a *App) restoreFromFile(source string) (string, error) {
	info, err := os.Stat(source)
	if err != nil {
		return "", err
	}
	if info.Size() > maxBackupSize {
		return "", errors.New("not a Stockbook backup: the file is too large")
	}

	data, err := os.ReadFile(source)
	if err != nil {
		return "", err
	}

	var envelope backupEnvelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		return "", errors.New("not a Stockbook backup: the file is not valid JSON")
	}
	if envelope.Format != "stockbook-backup" ||
		!isJSONArray(envelope.Products) ||
		!isJSONArray(envelope.Purchases) ||
		!isJSONArray(envelope.Sales) {
		return "", errors.New("not a Stockbook backup: expected products, purchases, and sales lists")
	}

	safety, err := a.writeSafetySnapshot()
	if err != nil {
		return "", err
	}

	var outcome rpcOutcome
	if err := a.api.rpc(a.context(), "stockbook_import",
		map[string]any{"p_data": json.RawMessage(data)}, &outcome); err != nil {
		return "", err
	}
	if !outcome.OK {
		message := outcome.Error
		if message == "" {
			message = "the backup could not be imported"
		}
		return "", errors.New(message)
	}

	// Restoring replaces the signed-in workspace, so force a fresh sign-in.
	a.destroySession()
	return safety, nil
}

// writeSafetySnapshot keeps a pre-restore copy outside the normal backup
// location so a mistaken restore can always be undone.
func (a *App) writeSafetySnapshot() (string, error) {
	dir := filepath.Join(a.dataDir, "backups")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	target := filepath.Join(dir, "restore-safety-"+time.Now().Format("20060102-150405.000")+".json")
	if _, err := a.backupToFile(target); err != nil {
		return "", err
	}
	return target, nil
}
