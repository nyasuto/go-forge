package setup

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrintTmuxConfig(t *testing.T) {
	origFind := FindBinaryPath
	defer func() { FindBinaryPath = origFind }()
	FindBinaryPath = func() (string, error) { return "/usr/local/bin/gf-claude-quota", nil }

	var buf bytes.Buffer
	code := Run(&buf, &bytes.Buffer{}, &SetupOptions{Tmux: true})

	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	out := buf.String()
	if !strings.Contains(out, "tmux") {
		t.Errorf("output should contain 'tmux', got %q", out)
	}
	if !strings.Contains(out, "/usr/local/bin/gf-claude-quota") {
		t.Errorf("output should contain binary path, got %q", out)
	}
	if !strings.Contains(out, "--oneline") {
		t.Errorf("output should contain '--oneline', got %q", out)
	}
	if !strings.Contains(out, "status-interval") {
		t.Errorf("output should contain 'status-interval', got %q", out)
	}
}

func TestPrintStarshipConfig(t *testing.T) {
	origFind := FindBinaryPath
	defer func() { FindBinaryPath = origFind }()
	FindBinaryPath = func() (string, error) { return "/usr/local/bin/gf-claude-quota", nil }

	var buf bytes.Buffer
	code := Run(&buf, &bytes.Buffer{}, &SetupOptions{Starship: true})

	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	out := buf.String()
	if !strings.Contains(out, "starship") {
		t.Errorf("output should contain 'starship', got %q", out)
	}
	if !strings.Contains(out, "[custom.claude_quota]") {
		t.Errorf("output should contain '[custom.claude_quota]', got %q", out)
	}
	if !strings.Contains(out, "/usr/local/bin/gf-claude-quota") {
		t.Errorf("output should contain binary path, got %q", out)
	}
}

func TestSetupXbar_NewFile(t *testing.T) {
	tmpDir := t.TempDir()
	pluginFile := filepath.Join(tmpDir, "claude-quota.5m.sh")

	origPath := XbarPluginPath
	origFind := FindBinaryPath
	defer func() {
		XbarPluginPath = origPath
		FindBinaryPath = origFind
	}()
	XbarPluginPath = func() string { return pluginFile }
	FindBinaryPath = func() (string, error) { return "/usr/local/bin/gf-claude-quota", nil }

	var stdout, stderr bytes.Buffer
	code := Run(&stdout, &stderr, &SetupOptions{Xbar: true})

	if code != 0 {
		t.Errorf("exit code = %d, want 0; stderr: %s", code, stderr.String())
	}

	// Verify file was created
	data, err := os.ReadFile(pluginFile)
	if err != nil {
		t.Fatalf("failed to read plugin file: %v", err)
	}
	content := string(data)
	if !strings.HasPrefix(content, "#!/bin/bash\n") {
		t.Error("plugin should start with shebang")
	}
	if !strings.Contains(content, "/usr/local/bin/gf-claude-quota --xbar") {
		t.Errorf("plugin should contain binary path with --xbar, got %q", content)
	}

	// Verify executable permission
	info, _ := os.Stat(pluginFile)
	if info.Mode()&0111 == 0 {
		t.Error("plugin file should be executable")
	}

	if !strings.Contains(stdout.String(), "Plugin installed") {
		t.Errorf("output should contain 'Plugin installed', got %q", stdout.String())
	}
}

func TestSetupXbar_AlreadyInstalled(t *testing.T) {
	tmpDir := t.TempDir()
	pluginFile := filepath.Join(tmpDir, "claude-quota.5m.sh")

	// Pre-create with matching content
	content := "#!/bin/bash\n/usr/local/bin/gf-claude-quota --xbar\n"
	os.WriteFile(pluginFile, []byte(content), 0755)

	origPath := XbarPluginPath
	origFind := FindBinaryPath
	defer func() {
		XbarPluginPath = origPath
		FindBinaryPath = origFind
	}()
	XbarPluginPath = func() string { return pluginFile }
	FindBinaryPath = func() (string, error) { return "/usr/local/bin/gf-claude-quota", nil }

	var stdout, stderr bytes.Buffer
	code := Run(&stdout, &stderr, &SetupOptions{Xbar: true})

	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "already installed") {
		t.Errorf("output should contain 'already installed', got %q", stdout.String())
	}
}

func TestSetupXbar_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	pluginFile := filepath.Join(tmpDir, "claude-quota.5m.sh")

	origPath := XbarPluginPath
	origFind := FindBinaryPath
	defer func() {
		XbarPluginPath = origPath
		FindBinaryPath = origFind
	}()
	XbarPluginPath = func() string { return pluginFile }
	FindBinaryPath = func() (string, error) { return "/usr/local/bin/gf-claude-quota", nil }

	var stdout, stderr bytes.Buffer
	code := Run(&stdout, &stderr, &SetupOptions{Xbar: true, DryRun: true})

	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}

	out := stdout.String()
	if !strings.Contains(out, "Dry run") {
		t.Errorf("output should contain 'Dry run', got %q", out)
	}
	if !strings.Contains(out, "#!/bin/bash") {
		t.Errorf("output should contain script content, got %q", out)
	}

	// File should NOT be created
	if _, err := os.Stat(pluginFile); !os.IsNotExist(err) {
		t.Error("plugin file should not exist after dry run")
	}
}

func TestSetupXbar_NestedDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	pluginFile := filepath.Join(tmpDir, "deep", "nested", "claude-quota.5m.sh")

	origPath := XbarPluginPath
	origFind := FindBinaryPath
	defer func() {
		XbarPluginPath = origPath
		FindBinaryPath = origFind
	}()
	XbarPluginPath = func() string { return pluginFile }
	FindBinaryPath = func() (string, error) { return "/usr/local/bin/gf-claude-quota", nil }

	var stdout, stderr bytes.Buffer
	code := Run(&stdout, &stderr, &SetupOptions{Xbar: true})

	if code != 0 {
		t.Errorf("exit code = %d, want 0; stderr: %s", code, stderr.String())
	}

	if _, err := os.Stat(pluginFile); os.IsNotExist(err) {
		t.Error("plugin file should exist in nested directory")
	}
}

func TestSetupXbar_NoHomeDir(t *testing.T) {
	origPath := XbarPluginPath
	origFind := FindBinaryPath
	defer func() {
		XbarPluginPath = origPath
		FindBinaryPath = origFind
	}()
	XbarPluginPath = func() string { return "" }
	FindBinaryPath = func() (string, error) { return "/usr/local/bin/gf-claude-quota", nil }

	var stdout, stderr bytes.Buffer
	code := Run(&stdout, &stderr, &SetupOptions{Xbar: true})

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "home directory") {
		t.Errorf("stderr should contain 'home directory', got %q", stderr.String())
	}
}

func TestSetupStatusLine_NewFile(t *testing.T) {
	tmpDir := t.TempDir()
	settingsFile := filepath.Join(tmpDir, "settings.json")

	origPath := SettingsPath
	origFind := FindBinaryPath
	defer func() {
		SettingsPath = origPath
		FindBinaryPath = origFind
	}()
	SettingsPath = func() string { return settingsFile }
	FindBinaryPath = func() (string, error) { return "/usr/local/bin/gf-claude-quota", nil }

	var stdout, stderr bytes.Buffer
	code := Run(&stdout, &stderr, &SetupOptions{})

	if code != 0 {
		t.Errorf("exit code = %d, want 0; stderr: %s", code, stderr.String())
	}

	data, err := os.ReadFile(settingsFile)
	if err != nil {
		t.Fatalf("failed to read settings: %v", err)
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatalf("failed to parse settings: %v", err)
	}

	sl, ok := settings["statusLine"]
	if !ok {
		t.Fatal("statusLine not found in settings")
	}
	if sl != "/usr/local/bin/gf-claude-quota --statusline" {
		t.Errorf("statusLine = %q, want '/usr/local/bin/gf-claude-quota --statusline'", sl)
	}

	if !strings.Contains(stdout.String(), "statusLine configured") {
		t.Errorf("output should contain 'statusLine configured', got %q", stdout.String())
	}
}

func TestSetupStatusLine_ExistingSettings(t *testing.T) {
	tmpDir := t.TempDir()
	settingsFile := filepath.Join(tmpDir, "settings.json")

	existing := map[string]interface{}{
		"theme": "dark",
		"model": "claude-sonnet",
	}
	data, _ := json.MarshalIndent(existing, "", "  ")
	os.WriteFile(settingsFile, data, 0600)

	origPath := SettingsPath
	origFind := FindBinaryPath
	defer func() {
		SettingsPath = origPath
		FindBinaryPath = origFind
	}()
	SettingsPath = func() string { return settingsFile }
	FindBinaryPath = func() (string, error) { return "/usr/local/bin/gf-claude-quota", nil }

	var stdout, stderr bytes.Buffer
	code := Run(&stdout, &stderr, &SetupOptions{})

	if code != 0 {
		t.Errorf("exit code = %d, want 0; stderr: %s", code, stderr.String())
	}

	// Check backup was created
	if !strings.Contains(stdout.String(), "Backup created") {
		t.Errorf("output should contain 'Backup created', got %q", stdout.String())
	}

	// Verify backup file exists
	entries, _ := os.ReadDir(tmpDir)
	backupFound := false
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "settings.json.backup.") {
			backupFound = true
			// Verify backup content
			backupData, _ := os.ReadFile(filepath.Join(tmpDir, e.Name()))
			if string(backupData) != string(data) {
				t.Error("backup content doesn't match original")
			}
		}
	}
	if !backupFound {
		t.Error("backup file not found")
	}

	// Verify new settings
	newData, _ := os.ReadFile(settingsFile)
	var newSettings map[string]interface{}
	json.Unmarshal(newData, &newSettings)

	if newSettings["theme"] != "dark" {
		t.Error("existing 'theme' setting was lost")
	}
	if newSettings["model"] != "claude-sonnet" {
		t.Error("existing 'model' setting was lost")
	}
	if newSettings["statusLine"] != "/usr/local/bin/gf-claude-quota --statusline" {
		t.Errorf("statusLine = %v, want expected value", newSettings["statusLine"])
	}
}

func TestSetupStatusLine_AlreadyConfigured(t *testing.T) {
	tmpDir := t.TempDir()
	settingsFile := filepath.Join(tmpDir, "settings.json")

	existing := map[string]interface{}{
		"statusLine": "/usr/local/bin/gf-claude-quota --statusline",
	}
	data, _ := json.MarshalIndent(existing, "", "  ")
	os.WriteFile(settingsFile, data, 0600)

	origPath := SettingsPath
	origFind := FindBinaryPath
	defer func() {
		SettingsPath = origPath
		FindBinaryPath = origFind
	}()
	SettingsPath = func() string { return settingsFile }
	FindBinaryPath = func() (string, error) { return "/usr/local/bin/gf-claude-quota", nil }

	var stdout, stderr bytes.Buffer
	code := Run(&stdout, &stderr, &SetupOptions{})

	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "already configured") {
		t.Errorf("output should contain 'already configured', got %q", stdout.String())
	}
}

func TestSetupStatusLine_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	settingsFile := filepath.Join(tmpDir, "settings.json")

	origPath := SettingsPath
	origFind := FindBinaryPath
	defer func() {
		SettingsPath = origPath
		FindBinaryPath = origFind
	}()
	SettingsPath = func() string { return settingsFile }
	FindBinaryPath = func() (string, error) { return "/usr/local/bin/gf-claude-quota", nil }

	var stdout, stderr bytes.Buffer
	code := Run(&stdout, &stderr, &SetupOptions{DryRun: true})

	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}

	out := stdout.String()
	if !strings.Contains(out, "Dry run") {
		t.Errorf("output should contain 'Dry run', got %q", out)
	}
	if !strings.Contains(out, "statusLine") {
		t.Errorf("output should contain 'statusLine', got %q", out)
	}

	// File should NOT be created
	if _, err := os.Stat(settingsFile); !os.IsNotExist(err) {
		t.Error("settings file should not exist after dry run")
	}
}

func TestSetupStatusLine_DryRunWithExisting(t *testing.T) {
	tmpDir := t.TempDir()
	settingsFile := filepath.Join(tmpDir, "settings.json")

	existing := map[string]interface{}{
		"theme": "dark",
	}
	data, _ := json.MarshalIndent(existing, "", "  ")
	os.WriteFile(settingsFile, data, 0600)

	origPath := SettingsPath
	origFind := FindBinaryPath
	defer func() {
		SettingsPath = origPath
		FindBinaryPath = origFind
	}()
	SettingsPath = func() string { return settingsFile }
	FindBinaryPath = func() (string, error) { return "/usr/local/bin/gf-claude-quota", nil }

	var stdout, stderr bytes.Buffer
	code := Run(&stdout, &stderr, &SetupOptions{DryRun: true})

	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}

	// Original file should be unchanged
	currentData, _ := os.ReadFile(settingsFile)
	if string(currentData) != string(data) {
		t.Error("dry run modified the settings file")
	}
}

func TestSetupStatusLine_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	settingsFile := filepath.Join(tmpDir, "settings.json")
	os.WriteFile(settingsFile, []byte("{invalid json"), 0600)

	origPath := SettingsPath
	origFind := FindBinaryPath
	defer func() {
		SettingsPath = origPath
		FindBinaryPath = origFind
	}()
	SettingsPath = func() string { return settingsFile }
	FindBinaryPath = func() (string, error) { return "/usr/local/bin/gf-claude-quota", nil }

	var stdout, stderr bytes.Buffer
	code := Run(&stdout, &stderr, &SetupOptions{})

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "failed to parse") {
		t.Errorf("stderr should contain 'failed to parse', got %q", stderr.String())
	}
}

func TestSetupStatusLine_NoHomeDir(t *testing.T) {
	origPath := SettingsPath
	origFind := FindBinaryPath
	defer func() {
		SettingsPath = origPath
		FindBinaryPath = origFind
	}()
	SettingsPath = func() string { return "" }
	FindBinaryPath = func() (string, error) { return "/usr/local/bin/gf-claude-quota", nil }

	var stdout, stderr bytes.Buffer
	code := Run(&stdout, &stderr, &SetupOptions{})

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "home directory") {
		t.Errorf("stderr should contain 'home directory', got %q", stderr.String())
	}
}

func TestSetupStatusLine_OverwriteExistingStatusLine(t *testing.T) {
	tmpDir := t.TempDir()
	settingsFile := filepath.Join(tmpDir, "settings.json")

	existing := map[string]interface{}{
		"statusLine": "some-other-command --flag",
	}
	data, _ := json.MarshalIndent(existing, "", "  ")
	os.WriteFile(settingsFile, data, 0600)

	origPath := SettingsPath
	origFind := FindBinaryPath
	defer func() {
		SettingsPath = origPath
		FindBinaryPath = origFind
	}()
	SettingsPath = func() string { return settingsFile }
	FindBinaryPath = func() (string, error) { return "/usr/local/bin/gf-claude-quota", nil }

	var stdout, stderr bytes.Buffer
	code := Run(&stdout, &stderr, &SetupOptions{})

	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}

	// Should have created backup
	if !strings.Contains(stdout.String(), "Backup created") {
		t.Errorf("output should contain 'Backup created', got %q", stdout.String())
	}

	// Verify overwritten
	newData, _ := os.ReadFile(settingsFile)
	var newSettings map[string]interface{}
	json.Unmarshal(newData, &newSettings)

	if newSettings["statusLine"] != "/usr/local/bin/gf-claude-quota --statusline" {
		t.Errorf("statusLine = %v, want expected value", newSettings["statusLine"])
	}
}

func TestSetupStatusLine_NestedDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	settingsFile := filepath.Join(tmpDir, "deep", "nested", "settings.json")

	origPath := SettingsPath
	origFind := FindBinaryPath
	defer func() {
		SettingsPath = origPath
		FindBinaryPath = origFind
	}()
	SettingsPath = func() string { return settingsFile }
	FindBinaryPath = func() (string, error) { return "/usr/local/bin/gf-claude-quota", nil }

	var stdout, stderr bytes.Buffer
	code := Run(&stdout, &stderr, &SetupOptions{})

	if code != 0 {
		t.Errorf("exit code = %d, want 0; stderr: %s", code, stderr.String())
	}

	// Verify file was created in nested directory
	if _, err := os.Stat(settingsFile); os.IsNotExist(err) {
		t.Error("settings file should exist")
	}
}
