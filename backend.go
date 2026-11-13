package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// backendMode chooses where the ledger lives. Local keeps everything in this
// computer's SQLite database; cloud uses the Stockbook Cloud Supabase project.
// The choice is per installation and can be changed from the sign-in screen.
type backendMode string

const (
	modeLocal backendMode = "local"
	modeCloud backendMode = "cloud"

	backendFileName = "backend.json"
)

type storedBackend struct {
	Mode string `json:"mode"`
}

// BackendInfo describes the active backend for the sign-in screen.
type BackendInfo struct {
	Mode         string `json:"mode"`
	Label        string `json:"label"`
	IsLocal      bool   `json:"isLocal"`
	IsCloud      bool   `json:"isCloud"`
	DataLocation string `json:"dataLocation,omitempty"`
	Error        string `json:"error,omitempty"`
}

var errLocalUnavailable = errors.New("the local database is unavailable")

func backendInfoFor(mode backendMode) BackendInfo {
	info := BackendInfo{Mode: string(mode)}
	switch mode {
	case modeLocal:
		info.Label = "On this computer"
		info.IsLocal = true
	default:
		info.Label = "Stockbook Cloud"
		info.IsCloud = true
	}
	return info
}

func loadBackendMode(dir string) (backendMode, error) {
	path := filepath.Join(dir, backendFileName)
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		// Stockbook 2.0 kept a cloud session in session.json. Prefer cloud for
		// those installs so an upgrade does not appear to lose its workspace.
		// Every other install starts in local mode, keeping Stockbook local
		// first by default.
		if _, sessionErr := os.Stat(filepath.Join(dir, sessionFileName)); sessionErr == nil {
			return modeCloud, nil
		}
		return modeLocal, nil
	}
	if err != nil {
		return modeLocal, fmt.Errorf("read backend settings: %w", err)
	}

	var stored storedBackend
	if err := json.Unmarshal(data, &stored); err != nil {
		return modeLocal, fmt.Errorf("backend.json is not valid JSON: %w", err)
	}
	switch backendMode(strings.TrimSpace(stored.Mode)) {
	case modeLocal:
		return modeLocal, nil
	case modeCloud:
		return modeCloud, nil
	default:
		return modeLocal, fmt.Errorf("backend.json must select %q or %q", modeLocal, modeCloud)
	}
}

func (a *App) saveBackendMode(mode backendMode) error {
	if a.dataDir == "" {
		return errors.New("Stockbook's data folder is unavailable")
	}
	data, err := json.MarshalIndent(storedBackend{Mode: string(mode)}, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(a.dataDir, 0o700); err != nil {
		return err
	}
	path := filepath.Join(a.dataDir, backendFileName)
	temp := path + ".tmp"
	if err := os.WriteFile(temp, data, 0o600); err != nil {
		return err
	}
	if err := os.Rename(temp, path); err != nil {
		_ = os.Remove(temp)
		return err
	}
	return os.Chmod(path, 0o600)
}

func (a *App) currentMode() backendMode {
	a.backendMu.RLock()
	defer a.backendMu.RUnlock()
	return a.mode
}

func (a *App) currentInitErr() error {
	a.backendMu.RLock()
	defer a.backendMu.RUnlock()
	return a.initErr
}

func (a *App) setInitErr(err error) {
	a.backendMu.Lock()
	a.initErr = err
	a.backendMu.Unlock()
}

func (a *App) localDB() (*localStore, error) {
	a.backendMu.RLock()
	defer a.backendMu.RUnlock()
	if a.local == nil {
		return nil, errLocalUnavailable
	}
	return a.local, nil
}

// BackendInfo returns the active backend without exposing any key material.
func (a *App) BackendInfo() BackendInfo {
	info := backendInfoFor(a.currentMode())
	info.DataLocation = a.DataLocation()
	if err := a.currentInitErr(); err != nil {
		info.Error = err.Error()
	}
	return info
}

// openBackend makes sure target has a live client or database. It is safe to
// call for the already-active backend, which gives the boot-error screen a
// retry path after a temporary failure or a corrupt backend.json.
func (a *App) openBackend(target backendMode) error {
	switch target {
	case modeLocal:
		a.backendMu.RLock()
		ready := a.local != nil
		a.backendMu.RUnlock()
		if ready {
			return nil
		}
		store, err := openLocalStore()
		if err != nil {
			return err
		}
		a.backendMu.Lock()
		a.local = store
		a.backendMu.Unlock()
	case modeCloud:
		a.backendMu.RLock()
		ready := a.api != nil
		a.backendMu.RUnlock()
		if ready {
			return nil
		}
		config, err := cloudBackendConfig()
		if err != nil {
			return err
		}
		api, err := newSupabaseAPI(config)
		if err != nil {
			return err
		}
		api.onTokens = a.writeSessionFile
		a.backendMu.Lock()
		a.api = api
		a.backendMu.Unlock()
	}
	return nil
}

// SetBackend switches between the local SQLite ledger and Stockbook Cloud.
// Sessions for both backends are kept separately, so switching back resumes
// the workspace that was already signed in.
func (a *App) SetBackend(mode string) BackendInfo {
	target := backendMode(strings.TrimSpace(mode))
	if target != modeLocal && target != modeCloud {
		info := a.BackendInfo()
		info.Error = "Choose either the local workspace or Stockbook Cloud."
		return info
	}

	if err := a.openBackend(target); err != nil {
		info := a.BackendInfo()
		if target == modeLocal {
			info.Error = "Could not open the local database: " + err.Error()
		} else {
			info.Error = "Could not open Stockbook Cloud: " + err.Error()
		}
		return info
	}

	if err := a.saveBackendMode(target); err != nil {
		info := a.BackendInfo()
		info.Error = "Could not save the backend choice: " + err.Error()
		return info
	}

	a.mu.Lock()
	a.company = nil
	a.mu.Unlock()
	a.backendMu.Lock()
	a.mode = target
	a.initErr = nil
	a.backendMu.Unlock()

	a.restoreSession()
	return a.BackendInfo()
}

// Session reports the signed-in workspace for whichever backend is active.
func (a *App) Session() SessionResult {
	if a.currentMode() == modeLocal {
		if err := a.currentInitErr(); err != nil {
			return SessionResult{Error: err.Error()}
		}
		company := a.currentCompany()
		return SessionResult{Authenticated: company != nil, Company: company}
	}
	return a.cloudSession()
}

// Register creates an account in the active backend.
func (a *App) Register(in RegisterInput) AuthResult {
	if a.currentMode() == modeLocal {
		if err := a.currentInitErr(); err != nil {
			return AuthResult{Error: err.Error()}
		}
		return a.localRegister(in)
	}
	return a.cloudRegister(in)
}

// Login signs in to the active backend.
func (a *App) Login(in LoginInput) AuthResult {
	if a.currentMode() == modeLocal {
		if err := a.currentInitErr(); err != nil {
			return AuthResult{Error: err.Error()}
		}
		return a.localLogin(in)
	}
	return a.cloudLogin(in)
}

// Logout clears the active backend's session.
func (a *App) Logout() error {
	if a.currentMode() == modeLocal {
		a.destroyLocalSession()
		return nil
	}
	return a.cloudLogout()
}

func (a *App) Overview(in RangePageInput) (OverviewResult, error) {
	if a.currentMode() == modeLocal {
		if _, err := a.localDB(); err != nil {
			return OverviewResult{}, err
		}
		return a.localOverview(in)
	}
	return a.cloudOverview(in)
}

func (a *App) Items(in ItemsInput) (ItemsResult, error) {
	if a.currentMode() == modeLocal {
		if _, err := a.localDB(); err != nil {
			return ItemsResult{}, err
		}
		return a.localItems(in)
	}
	return a.cloudItems(in)
}

func (a *App) Inventory(in InventoryInput) (InventoryResult, error) {
	if a.currentMode() == modeLocal {
		if _, err := a.localDB(); err != nil {
			return InventoryResult{}, err
		}
		return a.localInventory(in)
	}
	return a.cloudInventory(in)
}

func (a *App) Purchases(in RangePageInput) (PurchasesResult, error) {
	if a.currentMode() == modeLocal {
		if _, err := a.localDB(); err != nil {
			return PurchasesResult{}, err
		}
		return a.localPurchases(in)
	}
	return a.cloudPurchases(in)
}

func (a *App) Sales(in RangePageInput) (SalesResult, error) {
	if a.currentMode() == modeLocal {
		if _, err := a.localDB(); err != nil {
			return SalesResult{}, err
		}
		return a.localSales(in)
	}
	return a.cloudSales(in)
}

func (a *App) Product(in IDInput) (ProductResult, error) {
	if a.currentMode() == modeLocal {
		if _, err := a.localDB(); err != nil {
			return ProductResult{}, err
		}
		return a.localProductByID(in)
	}
	return a.cloudProduct(in)
}

func (a *App) ProductOptions() ([]ProductOption, error) {
	if a.currentMode() == modeLocal {
		if _, err := a.localDB(); err != nil {
			return nil, err
		}
		return a.localProductOptionsList()
	}
	return a.cloudProductOptions()
}

func (a *App) Categories() ([]string, error) {
	if a.currentMode() == modeLocal {
		if _, err := a.localDB(); err != nil {
			return nil, err
		}
		return a.localCategoriesList()
	}
	return a.cloudCategories()
}

func (a *App) CreateItem(in ItemInput) (MutationResult, error) {
	if a.currentMode() == modeLocal {
		if _, err := a.localDB(); err != nil {
			return MutationResult{}, err
		}
		return a.localCreateItem(in)
	}
	return a.cloudCreateItem(in)
}

func (a *App) UpdateItem(in ItemInput) (MutationResult, error) {
	if a.currentMode() == modeLocal {
		if _, err := a.localDB(); err != nil {
			return MutationResult{}, err
		}
		return a.localUpdateItem(in)
	}
	return a.cloudUpdateItem(in)
}

func (a *App) CreatePurchase(in PurchaseInput) (MutationResult, error) {
	if a.currentMode() == modeLocal {
		if _, err := a.localDB(); err != nil {
			return MutationResult{}, err
		}
		return a.localCreatePurchase(in)
	}
	return a.cloudCreatePurchase(in)
}

func (a *App) DeletePurchase(in IDInput) (MutationResult, error) {
	if a.currentMode() == modeLocal {
		if _, err := a.localDB(); err != nil {
			return MutationResult{}, err
		}
		return a.localDeletePurchase(in)
	}
	return a.cloudDeletePurchase(in)
}

func (a *App) CreateSale(in SaleInput) (MutationResult, error) {
	if a.currentMode() == modeLocal {
		if _, err := a.localDB(); err != nil {
			return MutationResult{}, err
		}
		return a.localCreateSale(in)
	}
	return a.cloudCreateSale(in)
}

func (a *App) DeleteSale(in IDInput) (MutationResult, error) {
	if a.currentMode() == modeLocal {
		if _, err := a.localDB(); err != nil {
			return MutationResult{}, err
		}
		return a.localDeleteSale(in)
	}
	return a.cloudDeleteSale(in)
}
