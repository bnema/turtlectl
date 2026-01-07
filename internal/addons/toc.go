package addons

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// wowColorCodeRegex matches WoW color escape sequences like |cffRRGGBB and |r
var wowColorCodeRegex = regexp.MustCompile(`\|c[0-9a-fA-F]{8}|\|r`)

// TOCInfo contains parsed information from a .toc file
type TOCInfo struct {
	Title     string
	Version   string
	Author    string
	Notes     string
	Interface string
}

// stripWoWColorCodes removes WoW color escape sequences from a string
func stripWoWColorCodes(s string) string {
	return wowColorCodeRegex.ReplaceAllString(s, "")
}

// ParseTOC parses a .toc file and extracts metadata
func ParseTOC(tocPath string) (*TOCInfo, error) {
	file, err := os.Open(tocPath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	info := &TOCInfo{}
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// TOC metadata lines start with ##
		if !strings.HasPrefix(line, "##") {
			continue
		}

		// Remove ## prefix and trim
		line = strings.TrimPrefix(line, "##")
		line = strings.TrimSpace(line)

		// Split on first colon
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch strings.ToLower(key) {
		case "title":
			info.Title = stripWoWColorCodes(value)
		case "version":
			info.Version = value
		case "author":
			info.Author = value
		case "notes":
			info.Notes = stripWoWColorCodes(value)
		case "interface":
			info.Interface = value
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return info, nil
}

// FindTOCFile finds the .toc file in an addon directory
// Returns the path to the .toc file and the expected addon name
// It first checks the root directory, then checks immediate subdirectories
// (for multi-addon repos where the .toc is in a subfolder)
func FindTOCFile(addonDir string) (tocPath string, addonName string, err error) {
	entries, err := os.ReadDir(addonDir)
	if err != nil {
		return "", "", err
	}

	// First, check the root directory for a .toc file
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(strings.ToLower(entry.Name()), ".toc") {
			tocPath = filepath.Join(addonDir, entry.Name())
			// Addon name is the .toc filename without extension
			addonName = strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
			return tocPath, addonName, nil
		}
	}

	// If not found, check immediate subdirectories (for multi-addon repos)
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		subDir := filepath.Join(addonDir, entry.Name())
		subEntries, err := os.ReadDir(subDir)
		if err != nil {
			continue
		}
		for _, subEntry := range subEntries {
			if subEntry.IsDir() {
				continue
			}
			if strings.HasSuffix(strings.ToLower(subEntry.Name()), ".toc") {
				tocPath = filepath.Join(subDir, subEntry.Name())
				addonName = strings.TrimSuffix(subEntry.Name(), filepath.Ext(subEntry.Name()))
				return tocPath, addonName, nil
			}
		}
	}

	return "", "", os.ErrNotExist
}

// GetAddonNameFromTOC extracts the expected addon name from a .toc file
func GetAddonNameFromTOC(addonDir string) (string, error) {
	_, name, err := FindTOCFile(addonDir)
	return name, err
}
