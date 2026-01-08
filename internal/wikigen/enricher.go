package wikigen

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/bnema/turtlectl/internal/wiki"
)

const (
	// GitHubGraphQLAPI is the GitHub GraphQL API endpoint
	GitHubGraphQLAPI = "https://api.github.com/graphql"

	// BatchSize is how many repos to fetch per GraphQL query
	// GitHub has complexity limits, ~100 repos per query is safe
	BatchSize = 50
)

// Enricher fetches metadata from GitHub GraphQL API
type Enricher struct {
	client        *http.Client
	token         string
	authenticated bool
}

// NewEnricher creates a new GitHub enricher
// Requires GITHUB_TOKEN for GraphQL API (no unauthenticated access)
func NewEnricher() *Enricher {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		token = os.Getenv("GH_TOKEN")
	}

	return &Enricher{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		token:         token,
		authenticated: token != "",
	}
}

// IsAuthenticated returns true if using a GitHub token
func (e *Enricher) IsAuthenticated() bool {
	return e.authenticated
}

// ConvertToAddons converts raw addons to WikiAddons without API enrichment
func (e *Enricher) ConvertToAddons(rawAddons []RawAddon) []wiki.WikiAddon {
	addons := make([]wiki.WikiAddon, 0, len(rawAddons))

	for _, raw := range rawAddons {
		addon := wiki.WikiAddon{
			URL:      raw.URL,
			Category: raw.Category,
			Name:     extractNameFromURL(raw.URL),
		}
		addons = append(addons, addon)
	}

	return addons
}

// repoKey creates a unique key for a repo (owner/name)
type repoKey struct {
	Owner string
	Name  string
	Index int // Index in the original addons slice
}

// graphQLResponse represents the GitHub GraphQL API response
type graphQLResponse struct {
	Data   map[string]json.RawMessage `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// repoData represents repository data from GraphQL
type repoData struct {
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	StargazerCount int       `json:"stargazerCount"`
	PushedAt       time.Time `json:"pushedAt"`
	Owner          struct {
		Login string `json:"login"`
	} `json:"owner"`
}

// EnrichAll enriches all addons with GitHub metadata using GraphQL batching
func (e *Enricher) EnrichAll(addons []wiki.WikiAddon, progressFn func(current, total int, name string)) {
	if !e.authenticated {
		fmt.Println("Warning: GITHUB_TOKEN not set, skipping enrichment (GraphQL requires auth)")
		return
	}

	// Build list of GitHub repos to fetch
	var repos []repoKey
	for i, addon := range addons {
		if !IsGitHubURL(addon.URL) {
			continue
		}
		owner, name, ok := ExtractRepoInfo(addon.URL)
		if !ok {
			continue
		}
		repos = append(repos, repoKey{Owner: owner, Name: name, Index: i})
	}

	total := len(repos)
	if total == 0 {
		return
	}

	// Process in batches
	processed := 0
	for i := 0; i < len(repos); i += BatchSize {
		end := i + BatchSize
		if end > len(repos) {
			end = len(repos)
		}
		batch := repos[i:end]

		// Fetch batch
		results, err := e.fetchBatch(batch)
		if err != nil {
			fmt.Printf("\nError fetching batch: %v\n", err)
			continue
		}

		// Apply results to addons
		for _, repo := range batch {
			processed++
			alias := fmt.Sprintf("repo%d", repo.Index)
			if data, ok := results[alias]; ok {
				addons[repo.Index].Description = data.Description
				addons[repo.Index].Stars = data.StargazerCount
				addons[repo.Index].LastCommit = data.PushedAt
				if data.Owner.Login != "" {
					addons[repo.Index].Author = data.Owner.Login
				}
			}

			if progressFn != nil {
				progressFn(processed, total, addons[repo.Index].Name)
			}
		}
	}
}

// fetchBatch fetches multiple repos in a single GraphQL query
func (e *Enricher) fetchBatch(repos []repoKey) (map[string]repoData, error) {
	// Build GraphQL query with aliases
	var queryParts []string
	for _, repo := range repos {
		alias := fmt.Sprintf("repo%d", repo.Index)
		// Escape any special characters in owner/name
		owner := strings.ReplaceAll(repo.Owner, `"`, `\"`)
		name := strings.ReplaceAll(repo.Name, `"`, `\"`)
		queryParts = append(queryParts, fmt.Sprintf(`%s: repository(owner: "%s", name: "%s") {
      name
      description
      stargazerCount
      pushedAt
      owner { login }
    }`, alias, owner, name))
	}

	query := fmt.Sprintf("query { %s }", strings.Join(queryParts, "\n"))

	// Make request
	reqBody, err := json.Marshal(map[string]string{"query": query})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	req, err := http.NewRequest("POST", GitHubGraphQLAPI, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "turtlectl/1.0 (Turtle WoW addon manager)")

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	// Parse response
	var gqlResp graphQLResponse
	if err := json.NewDecoder(resp.Body).Decode(&gqlResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Check for errors (but don't fail - some repos may not exist)
	if len(gqlResp.Errors) > 0 {
		// Log but continue - partial data is fine
		for _, e := range gqlResp.Errors {
			if !strings.Contains(e.Message, "Could not resolve") {
				fmt.Printf("\nGraphQL error: %s\n", e.Message)
			}
		}
	}

	// Parse repo data from response
	results := make(map[string]repoData)
	for alias, rawData := range gqlResp.Data {
		if rawData == nil || string(rawData) == "null" {
			continue
		}
		var data repoData
		if err := json.Unmarshal(rawData, &data); err != nil {
			continue
		}
		results[alias] = data
	}

	return results, nil
}

// extractNameFromURL extracts a reasonable name from a URL
func extractNameFromURL(url string) string {
	_, repo, ok := ExtractRepoInfo(url)
	if ok {
		return repo
	}
	parts := splitURL(url)
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return "unknown"
}

// splitURL splits a URL into path segments
func splitURL(url string) []string {
	url = trimProtocol(url)
	parts := make([]string, 0)
	for _, p := range splitPath(url) {
		if p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}

func trimProtocol(url string) string {
	if len(url) > 8 && url[:8] == "https://" {
		return url[8:]
	}
	if len(url) > 7 && url[:7] == "http://" {
		return url[7:]
	}
	return url
}

func splitPath(s string) []string {
	result := make([]string, 0)
	current := ""
	for _, c := range s {
		if c == '/' {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}
