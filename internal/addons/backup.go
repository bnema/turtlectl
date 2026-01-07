package addons

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	// MaxBackupsPerAddon is the maximum number of backups to keep per addon
	MaxBackupsPerAddon = 3
	// BackupTimestampFormat is the format used for backup directory names
	BackupTimestampFormat = "20060102-150405"
)

// BackupManager handles addon backups
type BackupManager struct {
	backupDir string
}

// NewBackupManager creates a new backup manager
func NewBackupManager(dataDir string) *BackupManager {
	return &BackupManager{
		backupDir: filepath.Join(dataDir, "backups"),
	}
}

// CreateBackup creates a backup of an addon directory
func (bm *BackupManager) CreateBackup(addonPath, addonName string) (string, error) {
	// Create backup directory structure
	addonBackupDir := filepath.Join(bm.backupDir, addonName)
	if err := os.MkdirAll(addonBackupDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Create timestamped backup folder
	timestamp := time.Now().Format(BackupTimestampFormat)
	backupPath := filepath.Join(addonBackupDir, timestamp)

	// Copy the addon directory
	if err := copyDir(addonPath, backupPath); err != nil {
		// Cleanup on failure
		_ = os.RemoveAll(backupPath)
		return "", fmt.Errorf("failed to backup addon: %w", err)
	}

	// Cleanup old backups
	if err := bm.cleanupOldBackups(addonName); err != nil {
		// Log but don't fail
		fmt.Printf("Warning: failed to cleanup old backups: %v\n", err)
	}

	return backupPath, nil
}

// RestoreBackup restores an addon from a backup
func (bm *BackupManager) RestoreBackup(addonName string, backupTimestamp string, destPath string) error {
	backupPath := filepath.Join(bm.backupDir, addonName, backupTimestamp)

	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup not found: %s", backupTimestamp)
	}

	// Remove existing addon if present
	if _, err := os.Stat(destPath); err == nil {
		if err := os.RemoveAll(destPath); err != nil {
			return fmt.Errorf("failed to remove existing addon: %w", err)
		}
	}

	// Copy backup to destination
	if err := copyDir(backupPath, destPath); err != nil {
		return fmt.Errorf("failed to restore backup: %w", err)
	}

	return nil
}

// ListBackups lists all available backups for an addon
func (bm *BackupManager) ListBackups(addonName string) ([]string, error) {
	addonBackupDir := filepath.Join(bm.backupDir, addonName)

	entries, err := os.ReadDir(addonBackupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	var backups []string
	for _, entry := range entries {
		if entry.IsDir() {
			backups = append(backups, entry.Name())
		}
	}

	// Sort by timestamp (newest first)
	sort.Sort(sort.Reverse(sort.StringSlice(backups)))

	return backups, nil
}

// GetLatestBackup returns the most recent backup for an addon
func (bm *BackupManager) GetLatestBackup(addonName string) (string, error) {
	backups, err := bm.ListBackups(addonName)
	if err != nil {
		return "", err
	}

	if len(backups) == 0 {
		return "", fmt.Errorf("no backups found for %s", addonName)
	}

	return backups[0], nil
}

// DeleteBackup deletes a specific backup
func (bm *BackupManager) DeleteBackup(addonName, timestamp string) error {
	backupPath := filepath.Join(bm.backupDir, addonName, timestamp)
	return os.RemoveAll(backupPath)
}

// DeleteAllBackups deletes all backups for an addon
func (bm *BackupManager) DeleteAllBackups(addonName string) error {
	addonBackupDir := filepath.Join(bm.backupDir, addonName)
	return os.RemoveAll(addonBackupDir)
}

// cleanupOldBackups removes old backups exceeding MaxBackupsPerAddon
func (bm *BackupManager) cleanupOldBackups(addonName string) error {
	backups, err := bm.ListBackups(addonName)
	if err != nil {
		return err
	}

	if len(backups) <= MaxBackupsPerAddon {
		return nil
	}

	// Remove oldest backups
	for _, backup := range backups[MaxBackupsPerAddon:] {
		if err := bm.DeleteBackup(addonName, backup); err != nil {
			return err
		}
	}

	return nil
}

// BackupSavedVariables creates a backup of SavedVariables for an addon
func (bm *BackupManager) BackupSavedVariables(gameDir, addonName string) (string, error) {
	svDir := filepath.Join(gameDir, "WTF", "Account")

	// Find SavedVariables files matching the addon name
	var svFiles []string
	err := filepath.Walk(svDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue on errors
		}
		if info.IsDir() {
			return nil
		}

		name := info.Name()
		// Match addon name in SavedVariables files
		if strings.HasPrefix(name, addonName) && strings.HasSuffix(name, ".lua") {
			svFiles = append(svFiles, path)
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	if len(svFiles) == 0 {
		return "", nil // No SavedVariables to backup
	}

	// Create backup directory
	timestamp := time.Now().Format(BackupTimestampFormat)
	backupPath := filepath.Join(bm.backupDir, addonName, "savedvariables", timestamp)
	if err := os.MkdirAll(backupPath, 0755); err != nil {
		return "", err
	}

	// Copy SavedVariables files
	for _, svFile := range svFiles {
		destFile := filepath.Join(backupPath, filepath.Base(svFile))
		if err := copyFile(svFile, destFile); err != nil {
			return "", err
		}
	}

	return backupPath, nil
}

// copyDir recursively copies a directory
func copyDir(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// copyFile copies a single file
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = srcFile.Close() }()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer func() { _ = dstFile.Close() }()

	_, err = io.Copy(dstFile, srcFile)
	return err
}
