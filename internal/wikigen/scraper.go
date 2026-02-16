package wikigen

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// WikiURL is the Turtle WoW addon wiki page
const WikiURL = "https://turtle-wow.fandom.com/wiki/Addons"

// WikiAPIURL is the MediaWiki API endpoint.
const WikiAPIURL = "https://turtle-wow.fandom.com/api.php"

// Scraper handles fetching and parsing the wiki page
type Scraper struct {
	client      *http.Client
	timeout     time.Duration
	endpointURL string
}

// NewScraper creates a new wiki scraper
func NewScraper() *Scraper {
	return &Scraper{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		timeout:     30 * time.Second,
		endpointURL: WikiAPIURL,
	}
}

// ScrapeResult contains the results of a wiki scrape
type ScrapeResult struct {
	Addons []RawAddon
	ETag   string
}

type mediaWikiSection struct {
	Index string `json:"index"`
	Line  string `json:"line"`
}

type mediaWikiSectionsResponse struct {
	Parse struct {
		Sections []mediaWikiSection `json:"sections"`
	} `json:"parse"`
}

type mediaWikiExternalLinksResponse struct {
	Parse struct {
		ExternalLinks []string `json:"externallinks"`
	} `json:"parse"`
}

// RawAddon represents a minimal addon entry scraped from wiki (before enrichment)
type RawAddon struct {
	URL      string // GitHub/GitLab URL
	Category string // Letter section (A-Z)
}

// gitURLPattern matches GitHub and GitLab repository URLs
var gitURLPattern = regexp.MustCompile(`^https?://(github\.com|gitlab\.com)/[^/]+/[^/]+/?$`)

// Scrape fetches the wiki page and extracts addon URLs
// If etag is provided, it will be sent as If-None-Match header
// Returns nil, nil if page hasn't changed (304 Not Modified)
func (s *Scraper) Scrape(etag string) (*ScrapeResult, error) {
	req, err := http.NewRequest("GET", s.buildParseURL("sections", ""), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "turtlectl/1.0 (Turtle WoW addon manager)")
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch wiki page: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Handle 304 Not Modified
	if resp.StatusCode == http.StatusNotModified {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Read body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var parsed mediaWikiSectionsResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("failed to decode MediaWiki API response: %w", err)
	}

	if len(parsed.Parse.Sections) == 0 {
		return nil, fmt.Errorf("MediaWiki API response missing parse.sections")
	}

	addons, err := s.fetchAddonsBySection(parsed.Parse.Sections)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch addon links: %w", err)
	}

	return &ScrapeResult{
		Addons: addons,
		ETag:   resp.Header.Get("ETag"),
	}, nil
}

func (s *Scraper) fetchAddonsBySection(sections []mediaWikiSection) ([]RawAddon, error) {
	var addons []RawAddon
	seen := make(map[string]bool) // Deduplicate URLs

	for _, section := range sections {
		category, ok := sectionToCategory(section.Line)
		if !ok {
			continue
		}

		externalLinks, err := s.fetchSectionExternalLinks(section.Index)
		if err != nil {
			return nil, err
		}

		for _, href := range externalLinks {
			repoURL := normalizeGitURL(href)
			if repoURL == "" || seen[repoURL] {
				continue
			}

			seen[repoURL] = true
			addons = append(addons, RawAddon{
				URL:      repoURL,
				Category: category,
			})
		}
	}

	return addons, nil
}

func (s *Scraper) fetchSectionExternalLinks(sectionIndex string) ([]string, error) {
	req, err := http.NewRequest("GET", s.buildParseURL("externallinks", sectionIndex), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "turtlectl/1.0 (Turtle WoW addon manager)")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch section %s links: %w", sectionIndex, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code for section %s: %d", sectionIndex, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read section %s response body: %w", sectionIndex, err)
	}

	var parsed mediaWikiExternalLinksResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("failed to decode section %s response: %w", sectionIndex, err)
	}

	return parsed.Parse.ExternalLinks, nil
}

func (s *Scraper) buildParseURL(prop, section string) string {
	values := url.Values{}
	values.Set("action", "parse")
	values.Set("page", "Addons")
	values.Set("prop", prop)
	values.Set("format", "json")
	values.Set("formatversion", "2")
	if section != "" {
		values.Set("section", section)
	}

	return s.endpointURL + "?" + values.Encode()
}

func sectionToCategory(line string) (string, bool) {
	line = strings.TrimSpace(line)
	if len(line) != 1 {
		return "", false
	}

	if line[0] < 'A' || line[0] > 'Z' {
		return "", false
	}

	return line, true
}

// normalizeGitURL validates and normalizes a Git repository URL
// Returns empty string if not a valid GitHub/GitLab URL
func normalizeGitURL(href string) string {
	// Skip anchors, relative URLs, etc.
	if !strings.HasPrefix(href, "http") {
		return ""
	}

	// Remove trailing slashes and .git suffix for consistency
	url := strings.TrimSuffix(href, "/")
	url = strings.TrimSuffix(url, ".git")

	// Remove any query parameters or fragments
	if idx := strings.Index(url, "?"); idx != -1 {
		url = url[:idx]
	}
	if idx := strings.Index(url, "#"); idx != -1 {
		url = url[:idx]
	}

	// Validate it's a GitHub/GitLab repo URL
	if !gitURLPattern.MatchString(url) {
		return ""
	}

	// Skip pages that aren't repos (like github.com/topics/...)
	parts := strings.Split(url, "/")
	if len(parts) >= 4 {
		owner := parts[3]
		// Skip common non-repo paths
		if owner == "topics" || owner == "explore" || owner == "settings" ||
			owner == "notifications" || owner == "login" || owner == "signup" {
			return ""
		}
	}

	return url
}

// ExtractRepoInfo extracts owner and repo name from a GitHub/GitLab URL
func ExtractRepoInfo(url string) (owner, repo string, ok bool) {
	// Remove protocol and host
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")

	// Split by /
	parts := strings.Split(url, "/")
	if len(parts) < 3 {
		return "", "", false
	}

	// parts[0] = github.com or gitlab.com
	// parts[1] = owner
	// parts[2] = repo
	owner = parts[1]
	repo = strings.TrimSuffix(parts[2], ".git")

	if owner == "" || repo == "" {
		return "", "", false
	}

	return owner, repo, true
}

// IsGitHubURL returns true if the URL is a GitHub repository
func IsGitHubURL(url string) bool {
	return strings.Contains(url, "github.com")
}

// IsGitLabURL returns true if the URL is a GitLab repository
func IsGitLabURL(url string) bool {
	return strings.Contains(url, "gitlab.com")
}
