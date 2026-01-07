package launcher

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/charmbracelet/log"
)

const (
	// AppImageAPIURL is the API endpoint that returns AppImage metadata and mirror URLs
	AppImageAPIURL = "https://launcher.turtlecraft.gg/api/launcher/TurtleWoW.AppImage"
	// DefaultMirror is the default CDN mirror to use
	DefaultMirror = "bunny"
)

// AppImageInfo represents the API response for AppImage metadata
type AppImageInfo struct {
	Name    string            `json:"name"`
	Version string            `json:"version"`
	Hash    string            `json:"hash"`
	Size    int64             `json:"size"`
	Tags    []string          `json:"tags"`
	Mirrors map[string]string `json:"mirrors"`
}

type Launcher struct {
	log          *log.Logger
	DataDir      string
	CacheDir     string
	GameDir      string
	AppImagePath string
	DesktopDir   string
	IconDir      string
	ScriptPath   string
}

type Preferences struct {
	Language        string `json:"language"`
	LinuxLaunchArgs string `json:"linuxLaunchArgs"`
	Mirror          string `json:"mirror"`
	ClientDir       string `json:"clientDir"`
	SafeDir         string `json:"safeDir"`
}

func New(logger *log.Logger) *Launcher {
	homeDir, _ := os.UserHomeDir()

	dataDir := os.Getenv("XDG_DATA_HOME")
	if dataDir == "" {
		dataDir = filepath.Join(homeDir, ".local", "share")
	}
	dataDir = filepath.Join(dataDir, "turtle-wow")

	cacheDir := os.Getenv("XDG_CACHE_HOME")
	if cacheDir == "" {
		cacheDir = filepath.Join(homeDir, ".cache")
	}
	cacheDir = filepath.Join(cacheDir, "turtle-wow")

	gameDir := os.Getenv("TURTLE_WOW_GAME_DIR")
	if gameDir == "" {
		gameDir = filepath.Join(homeDir, "Games", "turtle-wow")
	}

	desktopDir := filepath.Join(dataDir, "..", "applications")
	iconDir := filepath.Join(dataDir, "..", "icons")

	scriptPath, _ := os.Executable()

	l := &Launcher{
		log:          logger,
		DataDir:      dataDir,
		CacheDir:     cacheDir,
		GameDir:      gameDir,
		AppImagePath: filepath.Join(cacheDir, "TurtleWoW.AppImage"),
		DesktopDir:   desktopDir,
		IconDir:      iconDir,
		ScriptPath:   scriptPath,
	}

	log.Debug("Launcher initialized",
		"data_dir", l.DataDir,
		"cache_dir", l.CacheDir,
		"game_dir", l.GameDir,
		"appimage_path", l.AppImagePath,
	)

	return l
}

// EnsureLauncherDirs creates only the launcher directories (data and cache)
func (l *Launcher) EnsureLauncherDirs() error {
	dirs := []string{l.DataDir, l.CacheDir}

	for _, dir := range dirs {
		log.Debug("Creating directory", "path", dir)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	log.Debug("Launcher directories ready",
		"data", l.DataDir,
		"cache", l.CacheDir,
	)
	return nil
}

// EnsureAllDirs creates all directories including the game directory
func (l *Launcher) EnsureAllDirs() error {
	if err := l.EnsureLauncherDirs(); err != nil {
		return err
	}

	log.Debug("Creating game directory", "path", l.GameDir)
	if err := os.MkdirAll(l.GameDir, 0755); err != nil {
		if os.IsPermission(err) {
			parentDir := filepath.Dir(l.GameDir)
			log.Error("Permission denied creating game directory", "path", l.GameDir)
			log.Warn("Fix with: sudo chown $USER:$USER " + parentDir)
			return fmt.Errorf("permission denied: %w", err)
		}
		return fmt.Errorf("failed to create game directory %s: %w", l.GameDir, err)
	}

	log.Info("Directories ready",
		"data", l.DataDir,
		"cache", l.CacheDir,
		"game", l.GameDir,
	)
	return nil
}

func (l *Launcher) fetchAppImageInfo() (*AppImageInfo, error) {
	log.Debug("Fetching AppImage info from API", "url", AppImageAPIURL)

	resp, err := http.Get(AppImageAPIURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch API: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var info AppImageInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to parse API response: %w", err)
	}

	log.Debug("AppImage info fetched",
		"version", info.Tags,
		"size", formatBytes(info.Size),
		"mirrors", len(info.Mirrors),
	)

	return &info, nil
}

func (l *Launcher) UpdateAppImage() error {
	log.Info("Checking for launcher updates")

	// Get local file size first
	var localSize int64 = 0
	localExists := false
	if info, err := os.Stat(l.AppImagePath); err == nil {
		localSize = info.Size()
		localExists = true
		log.Debug("Local file exists", "size", formatBytes(localSize))
	} else {
		log.Debug("No local AppImage found")
	}

	// Fetch AppImage info from API
	appInfo, err := l.fetchAppImageInfo()
	if err != nil {
		if localExists {
			log.Warn("Failed to check for updates, using existing AppImage", "error", err)
			return nil
		}
		log.Error("Cannot fetch AppImage info", "error", err)
		log.Info("You can manually download from https://turtle-wow.org and place it at:",
			"path", l.AppImagePath,
		)
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	// Compare and download if needed
	if appInfo.Size != localSize {
		log.Info("Downloading latest launcher",
			"remote_size", formatBytes(appInfo.Size),
			"local_size", formatBytes(localSize),
			"version", appInfo.Tags,
		)

		if err := l.downloadAppImage(appInfo); err != nil {
			if localExists {
				log.Warn("Download failed, using existing AppImage", "error", err)
				return nil
			}
			return err
		}

		log.Info("Launcher updated successfully", "version", appInfo.Tags)
	} else {
		log.Info("Launcher is up to date",
			"size", formatBytes(localSize),
			"version", appInfo.Tags,
		)
	}

	return nil
}

func (l *Launcher) downloadAppImage(info *AppImageInfo) error {
	// Get download URL from mirror
	downloadURL, ok := info.Mirrors[DefaultMirror]
	if !ok {
		// Fallback to first available mirror
		for name, url := range info.Mirrors {
			log.Debug("Using fallback mirror", "mirror", name)
			downloadURL = url
			break
		}
	}

	if downloadURL == "" {
		return fmt.Errorf("no download mirrors available")
	}

	log.Debug("Starting download", "url", downloadURL, "mirror", DefaultMirror)

	resp, err := http.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	tmpPath := l.AppImagePath + ".tmp"
	log.Debug("Writing to temporary file", "path", tmpPath)

	out, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	written, err := io.Copy(out, resp.Body)
	_ = out.Close()
	if err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to write file: %w", err)
	}

	log.Debug("Download complete", "bytes_written", written)

	// Move temp file to final location
	if err := os.Rename(tmpPath, l.AppImagePath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to move file: %w", err)
	}

	// Make executable
	if err := os.Chmod(l.AppImagePath, 0755); err != nil {
		return fmt.Errorf("failed to make executable: %w", err)
	}

	log.Debug("AppImage ready", "path", l.AppImagePath)
	return nil
}

func (l *Launcher) CleanConfig() error {
	prefsPath := filepath.Join(l.DataDir, "preferences.json")
	log.Debug("Checking config", "path", prefsPath)

	data, err := os.ReadFile(prefsPath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Debug("No existing preferences file")
			return nil
		}
		return err
	}

	content := string(data)

	// Check for old server URL
	if strings.Contains(content, "launcher.turtle-wow.org") {
		log.Warn("Found old server URL in config, removing")
		if err := os.Remove(prefsPath); err != nil {
			return fmt.Errorf("failed to remove old config: %w", err)
		}
		return nil
	}

	// Check launcher version
	var prefs map[string]interface{}
	if err := json.Unmarshal(data, &prefs); err == nil {
		if version, ok := prefs["launcherVersion"].(string); ok {
			log.Debug("Config launcher version", "version", version)
			if version < "2.3.0" {
				log.Warn("Config from old launcher version, backing up", "version", version)
				backupPath := fmt.Sprintf("%s.bak.%d", l.DataDir, os.Getpid())
				if err := os.Rename(l.DataDir, backupPath); err != nil {
					return fmt.Errorf("failed to backup old config: %w", err)
				}
				log.Info("Old config backed up", "path", backupPath)
				if err := os.MkdirAll(l.DataDir, 0755); err != nil {
					return err
				}
			}
		}
	}

	// Remove problematic migration files
	filesToRemove := []string{
		filepath.Join(l.DataDir, "custom-mpqs.json"),
		filepath.Join(l.DataDir, "custom-dlls.json"),
	}

	for _, f := range filesToRemove {
		if _, err := os.Stat(f); err == nil {
			log.Debug("Removing problematic file", "path", f)
			_ = os.Remove(f)
		}
	}

	log.Debug("Config cleanup complete")
	return nil
}

func (l *Launcher) InitPreferences() error {
	prefsPath := filepath.Join(l.DataDir, "preferences.json")
	log.Debug("Initializing preferences", "path", prefsPath)

	if _, err := os.Stat(prefsPath); os.IsNotExist(err) {
		log.Info("Creating default preferences")

		prefs := Preferences{
			Language:        "en",
			LinuxLaunchArgs: "wine $WoW.exe$",
			Mirror:          "bunny",
			ClientDir:       l.GameDir + "/",
			SafeDir:         l.GameDir + "/",
		}

		data, err := json.MarshalIndent(prefs, "", "    ")
		if err != nil {
			return fmt.Errorf("failed to marshal preferences: %w", err)
		}

		if err := os.WriteFile(prefsPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write preferences: %w", err)
		}

		log.Debug("Preferences created", "content", string(data))
	} else {
		log.Debug("Preferences file exists, updating game directory")

		// Read and update existing preferences
		data, err := os.ReadFile(prefsPath)
		if err != nil {
			return err
		}

		var prefs map[string]interface{}
		if err := json.Unmarshal(data, &prefs); err != nil {
			return err
		}

		prefs["clientDir"] = l.GameDir + "/"
		prefs["safeDir"] = l.GameDir + "/"

		newData, err := json.MarshalIndent(prefs, "", "    ")
		if err != nil {
			return err
		}

		if err := os.WriteFile(prefsPath, newData, 0644); err != nil {
			return err
		}

		log.Debug("Preferences updated")
	}

	return nil
}

func (l *Launcher) Launch(args []string) error {
	log.Info("Launching Turtle WoW",
		"appimage", l.AppImagePath,
		"workdir", l.GameDir,
		"args", args,
	)

	// Change to game directory
	if err := os.Chdir(l.GameDir); err != nil {
		return fmt.Errorf("failed to change to game directory: %w", err)
	}

	log.Debug("Changed to game directory", "path", l.GameDir)

	// Build command args
	cmdArgs := append([]string{l.AppImagePath}, args...)

	log.Debug("Executing AppImage", "command", cmdArgs)

	// Use syscall.Exec to replace current process
	return syscall.Exec(l.AppImagePath, cmdArgs, os.Environ())
}

// ExtractIcon extracts the TurtleWoW.png icon from the AppImage
func (l *Launcher) ExtractIcon() (string, error) {
	iconPath := filepath.Join(l.IconDir, "turtle-wow.png")

	// Check if icon already exists
	if _, err := os.Stat(iconPath); err == nil {
		log.Debug("Icon already exists", "path", iconPath)
		return iconPath, nil
	}

	// Check if AppImage exists
	if _, err := os.Stat(l.AppImagePath); os.IsNotExist(err) {
		return "", fmt.Errorf("AppImage not found at %s", l.AppImagePath)
	}

	log.Debug("Extracting icon from AppImage", "appimage", l.AppImagePath)

	// Create temp directory for extraction
	tmpDir, err := os.MkdirTemp("", "turtle-wow-extract-")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Extract only the icon file using --appimage-extract with pattern
	cmd := exec.Command(l.AppImagePath, "--appimage-extract", "TurtleWoW.png")
	cmd.Dir = tmpDir
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard

	if err := cmd.Run(); err != nil {
		log.Debug("Pattern extraction failed, trying full extraction", "error", err)
		// Fallback: extract everything and find the icon
		cmd = exec.Command(l.AppImagePath, "--appimage-extract")
		cmd.Dir = tmpDir
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("failed to extract AppImage: %w", err)
		}
	}

	// Find the extracted icon
	extractedIcon := filepath.Join(tmpDir, "squashfs-root", "TurtleWoW.png")
	if _, err := os.Stat(extractedIcon); os.IsNotExist(err) {
		return "", fmt.Errorf("icon not found in AppImage")
	}

	// Ensure icon directory exists
	if err := os.MkdirAll(l.IconDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create icon dir: %w", err)
	}

	// Copy icon to final location
	src, err := os.Open(extractedIcon)
	if err != nil {
		return "", fmt.Errorf("failed to open extracted icon: %w", err)
	}
	defer func() { _ = src.Close() }()

	dst, err := os.Create(iconPath)
	if err != nil {
		return "", fmt.Errorf("failed to create icon file: %w", err)
	}
	defer func() { _ = dst.Close() }()

	if _, err := io.Copy(dst, src); err != nil {
		return "", fmt.Errorf("failed to copy icon: %w", err)
	}

	log.Info("Icon extracted from AppImage", "path", iconPath)
	return iconPath, nil
}

func (l *Launcher) InstallDesktop() error {
	log.Info("Installing desktop integration")

	// Create directories
	if err := os.MkdirAll(l.DesktopDir, 0755); err != nil {
		return fmt.Errorf("failed to create desktop dir: %w", err)
	}
	if err := os.MkdirAll(l.IconDir, 0755); err != nil {
		return fmt.Errorf("failed to create icon dir: %w", err)
	}

	// Extract icon from AppImage
	iconPath, err := l.ExtractIcon()
	if err != nil {
		log.Warn("Failed to extract icon from AppImage, using fallback", "error", err)
		// Fallback: download from web
		iconPath = filepath.Join(l.IconDir, "turtle-wow.png")
		if _, statErr := os.Stat(iconPath); os.IsNotExist(statErr) {
			log.Debug("Downloading fallback icon")
			resp, dlErr := http.Get("https://turtle-wow.org/favicon.ico")
			if dlErr == nil {
				defer func() { _ = resp.Body.Close() }()
				if resp.StatusCode == http.StatusOK {
					out, createErr := os.Create(iconPath)
					if createErr == nil {
						_, _ = io.Copy(out, resp.Body)
						_ = out.Close()
						log.Debug("Fallback icon downloaded", "path", iconPath)
					}
				}
			}
		}
	}

	// Create desktop file
	desktopPath := filepath.Join(l.DesktopDir, "turtle-wow.desktop")
	desktopContent := fmt.Sprintf(`[Desktop Entry]
Name=Turtle WoW
Comment=Turtle WoW Launcher (Linux wrapper)
Exec=%s launch
Icon=%s
Terminal=false
Type=Application
Categories=Game;
Keywords=wow;warcraft;mmo;turtle;
`, l.ScriptPath, iconPath)

	log.Debug("Writing desktop file", "path", desktopPath)
	if err := os.WriteFile(desktopPath, []byte(desktopContent), 0644); err != nil {
		return fmt.Errorf("failed to write desktop file: %w", err)
	}

	// Update desktop database
	log.Debug("Updating desktop database")
	_ = exec.Command("update-desktop-database", l.DesktopDir).Run()

	log.Info("Desktop file installed", "path", desktopPath)
	return nil
}

func (l *Launcher) UninstallDesktop() error {
	log.Info("Removing desktop integration")

	desktopPath := filepath.Join(l.DesktopDir, "turtle-wow.desktop")
	iconPath := filepath.Join(l.IconDir, "turtle-wow.png")

	if err := os.Remove(desktopPath); err != nil && !os.IsNotExist(err) {
		log.Warn("Failed to remove desktop file", "error", err)
	} else {
		log.Debug("Removed desktop file", "path", desktopPath)
	}

	if err := os.Remove(iconPath); err != nil && !os.IsNotExist(err) {
		log.Warn("Failed to remove icon", "error", err)
	} else {
		log.Debug("Removed icon", "path", iconPath)
	}

	_ = exec.Command("update-desktop-database", l.DesktopDir).Run()

	log.Info("Desktop integration removed")
	return nil
}

func (l *Launcher) Clean(includeGameFiles bool) error {
	if includeGameFiles {
		log.Warn("Full purge - removing EVERYTHING including game files")
	} else {
		log.Warn("Nuclear clean - removing all data, cache, and config")
	}

	// Remove data directory (preferences, credentials, etc.)
	if err := os.RemoveAll(l.DataDir); err != nil {
		return fmt.Errorf("failed to remove data directory: %w", err)
	}
	log.Debug("Removed data directory", "path", l.DataDir)

	// Remove cache directory (AppImage, WebKit cache, etc.)
	if err := os.RemoveAll(l.CacheDir); err != nil {
		return fmt.Errorf("failed to remove cache directory: %w", err)
	}
	log.Debug("Removed cache directory", "path", l.CacheDir)

	// Remove desktop integration
	desktopFile := filepath.Join(l.DesktopDir, "turtle-wow.desktop")
	if err := os.Remove(desktopFile); err != nil && !os.IsNotExist(err) {
		log.Warn("Failed to remove desktop file", "error", err)
	} else {
		log.Debug("Removed desktop file", "path", desktopFile)
	}

	iconFile := filepath.Join(l.IconDir, "turtle-wow.png")
	if err := os.Remove(iconFile); err != nil && !os.IsNotExist(err) {
		log.Warn("Failed to remove icon", "error", err)
	} else {
		log.Debug("Removed icon", "path", iconFile)
	}

	// Update desktop database
	_ = exec.Command("update-desktop-database", l.DesktopDir).Run()

	// Optionally remove game files
	if includeGameFiles {
		if err := os.RemoveAll(l.GameDir); err != nil {
			if os.IsPermission(err) {
				parentDir := filepath.Dir(l.GameDir)
				log.Error("Permission denied removing game directory",
					"path", l.GameDir,
				)
				log.Warn("Try one of these commands:",
					"fix_parent", "sudo chown $USER:$USER "+parentDir,
					"force_remove", "sudo rm -rf "+l.GameDir,
				)
				return fmt.Errorf("permission denied: %w", err)
			}
			return fmt.Errorf("failed to remove game directory: %w", err)
		}
		log.Debug("Removed game directory", "path", l.GameDir)

		log.Info("Full purge complete",
			"removed_data", l.DataDir,
			"removed_cache", l.CacheDir,
			"removed_game", l.GameDir,
		)
	} else {
		log.Info("Clean complete",
			"removed_data", l.DataDir,
			"removed_cache", l.CacheDir,
		)
		log.Info("Game files preserved", "game_dir", l.GameDir)
	}

	return nil
}

func (l *Launcher) ResetCredentials() error {
	log.Warn("Resetting saved credentials")

	filesToRemove := []string{
		filepath.Join(l.DataDir, "vault.hold"),
		filepath.Join(l.DataDir, "salt.txt"),
	}
	dirsToRemove := []string{
		filepath.Join(l.DataDir, "storage"),
		filepath.Join(l.DataDir, "mediakeys"),
	}

	for _, f := range filesToRemove {
		if _, err := os.Stat(f); err == nil {
			log.Debug("Removing file", "path", f)
			_ = os.Remove(f)
		}
	}

	for _, d := range dirsToRemove {
		if _, err := os.Stat(d); err == nil {
			log.Debug("Removing directory", "path", d)
			_ = os.RemoveAll(d)
		}
	}

	log.Info("Credentials reset")
	return nil
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return strconv.FormatInt(bytes, 10) + " B"
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
