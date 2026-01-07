package wikigen

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// WikiURL is the Turtle WoW addon wiki page
const WikiURL = "https://turtle-wow.fandom.com/wiki/Addons"

// Scraper handles fetching and parsing the wiki page
type Scraper struct {
	client  *http.Client
	timeout time.Duration
}

// NewScraper creates a new wiki scraper
func NewScraper() *Scraper {
	return &Scraper{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		timeout: 30 * time.Second,
	}
}

// ScrapeResult contains the results of a wiki scrape
type ScrapeResult struct {
	Addons []RawAddon
	ETag   string
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
	req, err := http.NewRequest("GET", WikiURL, nil)
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

	// Parse HTML and extract addon URLs
	addons, err := s.parseHTML(string(body))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	return &ScrapeResult{
		Addons: addons,
		ETag:   resp.Header.Get("ETag"),
	}, nil
}

// parseHTML extracts addon URLs from the wiki HTML
func (s *Scraper) parseHTML(htmlContent string) ([]RawAddon, error) {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil, err
	}

	var addons []RawAddon
	seen := make(map[string]bool) // Deduplicate URLs
	currentCategory := ""

	// Walk the DOM tree
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		// Track current category (A, B, C, etc.)
		// Wiki uses h2 or h3 with span id="A", id="B", etc.
		if n.Type == html.ElementNode && (n.Data == "h2" || n.Data == "h3") {
			category := extractCategoryFromHeading(n)
			if category != "" {
				currentCategory = category
			}
		}

		// Look for anchor tags
		if n.Type == html.ElementNode && n.Data == "a" {
			href := getAttr(n, "href")
			if href != "" {
				// Normalize URL
				url := normalizeGitURL(href)
				if url != "" && !seen[url] {
					seen[url] = true
					addons = append(addons, RawAddon{
						URL:      url,
						Category: currentCategory,
					})
				}
			}
		}

		// Recurse into children
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}

	walk(doc)

	return addons, nil
}

// extractCategoryFromHeading extracts category letter from heading elements
// Looks for patterns like <h2><span id="A">A</span></h2>
func extractCategoryFromHeading(n *html.Node) string {
	// Look for span with single letter id
	var findSpan func(*html.Node) string
	findSpan = func(node *html.Node) string {
		if node.Type == html.ElementNode && node.Data == "span" {
			id := getAttr(node, "id")
			// Check if id is a single uppercase letter
			if len(id) == 1 && id[0] >= 'A' && id[0] <= 'Z' {
				return id
			}
			// Also check class="mw-headline" and look at text content
			if getAttr(node, "class") == "mw-headline" {
				text := getTextContent(node)
				if len(text) == 1 && text[0] >= 'A' && text[0] <= 'Z' {
					return text
				}
			}
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			if result := findSpan(c); result != "" {
				return result
			}
		}
		return ""
	}

	return findSpan(n)
}

// getAttr gets an attribute value from an HTML node
func getAttr(n *html.Node, key string) string {
	for _, attr := range n.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}
	return ""
}

// getTextContent extracts text content from a node
func getTextContent(n *html.Node) string {
	if n.Type == html.TextNode {
		return strings.TrimSpace(n.Data)
	}
	var text string
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		text += getTextContent(c)
	}
	return strings.TrimSpace(text)
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
