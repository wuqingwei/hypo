// Package app exposes the Hypo services bound to the frontend.
//
// The service layer is intentionally thin: it owns no cleanup logic. It
// locates the engine once, hands typed JSON snapshots to the UI, and turns
// engine failures into messages the dashboard can render.
package app

import (
	"context"
	"errors"
	"sync"

	"github.com/wuqingwei/hypo/internal/engine"
)

// notFoundMessage is what the dashboard shows when no engine tree exists.
const notFoundMessage = "No Mole engine was found. Hypo drives the open-source " +
	"Mole CLI for every operation and never deletes anything itself. Install " +
	"the engine next to Hypo (an \"engine\" folder beside the executable), keep " +
	"a Mole checkout beside the Hypo source tree, or point HYPO_ENGINE_ROOT at " +
	"an engine tree, then retry."

// errNoEngine is returned by every engine call while no engine is installed.
var errNoEngine = errors.New("no Mole engine is installed; see GetEngineInfo")

// EngineInfo describes the engine Hypo is driving, including the friendly
// failure the UI shows when none is installed.
type EngineInfo struct {
	Found   bool   `json:"found"`
	Root    string `json:"root,omitempty"`
	Version string `json:"version,omitempty"`
	Source  string `json:"source,omitempty"`
	Error   string `json:"error,omitempty"`
	// StatusTool reports whether the native status executable is present.
	// When false, the first status call goes through mole.ps1, which tries
	// to download the tool from GitHub Releases or build it with a local Go.
	StatusTool bool `json:"status_tool"`
}

// Service is the Wails service for the M0 dashboard: engine presence, one
// metrics snapshot, recent history.
type Service struct {
	mu     sync.Mutex
	client *engine.Client
	info   EngineInfo
}

// New locates the engine and returns a ready service. A missing engine is not
// an error here: the app must start and show guidance, not crash.
func New() *Service {
	svc := &Service{}
	svc.RefreshEngine()
	return svc
}

// GetEngineInfo returns the current engine presence and location.
func (s *Service) GetEngineInfo() EngineInfo {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.info
}

// RefreshEngine re-runs the locator. The dashboard calls it after the user
// installs an engine or sets HYPO_ENGINE_ROOT.
func (s *Service) RefreshEngine() EngineInfo {
	s.mu.Lock()
	defer s.mu.Unlock()
	install, err := engine.Locate()
	if err != nil {
		s.client = nil
		if errors.Is(err, engine.ErrNotFound) {
			s.info = EngineInfo{Found: false, Error: notFoundMessage}
		} else {
			s.info = EngineInfo{Found: false, Error: err.Error()}
		}
		return s.info
	}
	s.client = engine.NewClient(install)
	s.info = EngineInfo{
		Found:      true,
		Root:       install.Root,
		Version:    install.Version,
		Source:     string(install.Source),
		StatusTool: install.StatusBin != "",
	}
	return s.info
}

// GetStatus returns one metrics snapshot for the dashboard.
func (s *Service) GetStatus() (*engine.StatusSnapshot, error) {
	client, err := s.engineClient()
	if err != nil {
		return nil, err
	}
	return client.Status(context.Background())
}

// GetHistory returns the newest operations-log entries. limit follows the
// engine's rules (1-200); 0 keeps the engine default.
func (s *Service) GetHistory(limit int) (*engine.HistoryReport, error) {
	client, err := s.engineClient()
	if err != nil {
		return nil, err
	}
	return client.History(context.Background(), limit)
}

// GetCleanPreview runs the engine's `clean --dry-run --json`. It deletes
// nothing: dry run is the engine's own guarantee. The records are display
// facts for the cleanup page; execution is a separate, later call that the
// user confirms explicitly.
func (s *Service) GetCleanPreview() (*engine.CleanReport, error) {
	client, err := s.engineClient()
	if err != nil {
		return nil, err
	}
	return client.CleanPreview(context.Background())
}

// GetUninstallList returns the installed applications inventory.
func (s *Service) GetUninstallList() ([]engine.UninstallApp, error) {
	client, err := s.engineClient()
	if err != nil {
		return nil, err
	}
	return client.UninstallList(context.Background())
}

// GetCleanRun runs the engine's `clean --json` (execute mode) after the
// frontend has collected an explicit confirmation. The engine re-enumerates
// and re-runs every guard at execution time; the preview's selection is
// intent, never a list of files to delete.
func (s *Service) GetCleanRun() (*engine.CleanReport, error) {
	client, err := s.engineClient()
	if err != nil {
		return nil, err
	}
	return client.CleanRun(context.Background())
}

// GetUninstallSelect runs the engine's headless uninstall for the given
// selection keys after the frontend has collected an explicit confirmation.
// The engine rereads every registration right before launch and re-runs its
// guards; Hypo adds nothing to the decision.
func (s *Service) GetUninstallSelect(ids []string) (*engine.UninstallRun, error) {
	client, err := s.engineClient()
	if err != nil {
		return nil, err
	}
	return client.UninstallSelect(context.Background(), ids)
}

// GetAnalyze returns the engine's read-only directory scan (or the overview
// set when path is empty). This never deletes anything: the engine's JSON
// mode is read-only by contract.
func (s *Service) GetAnalyze(path string) (*engine.AnalyzeReport, error) {
	client, err := s.engineClient()
	if err != nil {
		return nil, err
	}
	return client.AnalyzeReport(context.Background(), path)
}

// GetWhitelist returns the engine's built-in patterns and the user file's
// lines. Read-only.
func (s *Service) GetWhitelist() (*engine.Whitelist, error) {
	client, err := s.engineClient()
	if err != nil {
		return nil, err
	}
	return client.GetWhitelist(context.Background())
}

// AddWhitelistEntry appends one user pattern; the engine picks it up on its
// next run because it re-reads the file every time.
func (s *Service) AddWhitelistEntry(pattern string) (*engine.Whitelist, error) {
	client, err := s.engineClient()
	if err != nil {
		return nil, err
	}
	return client.AddWhitelistEntry(context.Background(), pattern)
}

// RemoveWhitelistEntry deletes one user pattern line (exact, case-insensitive).
func (s *Service) RemoveWhitelistEntry(pattern string) (*engine.Whitelist, error) {
	client, err := s.engineClient()
	if err != nil {
		return nil, err
	}
	return client.RemoveWhitelistEntry(context.Background(), pattern)
}

func (s *Service) engineClient() (*engine.Client, error) {
	s.mu.Lock()
	client := s.client
	s.mu.Unlock()
	if client == nil {
		return nil, errNoEngine
	}
	return client, nil
}
