package wikigen

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestScrapeFetchesAddonsViaMediaWikiAPI(t *testing.T) {
	t.Helper()

	s := NewScraper()
	s.client = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if !strings.Contains(req.URL.String(), "/api.php") {
				t.Fatalf("expected API endpoint, got %s", req.URL.String())
			}

			if req.Header.Get("User-Agent") == "" {
				t.Fatal("expected User-Agent header")
			}

			if strings.Contains(req.URL.RawQuery, "prop=sections") {
				return jsonResponse(`{"parse":{"sections":[{"line":"Featured Addons","index":"13"},{"line":"A","index":"15"},{"line":"B","index":"16"}]}}`, `W/"abc123"`), nil
			}

			if strings.Contains(req.URL.RawQuery, "prop=externallinks") && strings.Contains(req.URL.RawQuery, "section=15") {
				return jsonResponse(`{"parse":{"externallinks":["https://github.com/foo/bar/","https://github.com/foo/bar","https://example.com/not-repo","https://gitlab.com/acme/baz"]}}`, ""), nil
			}

			if strings.Contains(req.URL.RawQuery, "prop=externallinks") && strings.Contains(req.URL.RawQuery, "section=16") {
				return jsonResponse(`{"parse":{"externallinks":["https://github.com/owner/repo"]}}`, ""), nil
			}

			t.Fatalf("unexpected request: %s", req.URL.String())
			return nil, nil
		}),
	}

	result, err := s.Scrape("")
	if err != nil {
		t.Fatalf("Scrape() returned error: %v", err)
	}

	if result == nil {
		t.Fatal("Scrape() returned nil result")
	}

	if result.ETag != `W/"abc123"` {
		t.Fatalf("unexpected ETag: got %q", result.ETag)
	}

	if len(result.Addons) != 3 {
		t.Fatalf("expected 3 addons, got %d", len(result.Addons))
	}

	if result.Addons[0].URL != "https://github.com/foo/bar" {
		t.Fatalf("unexpected URL: got %q", result.Addons[0].URL)
	}

	if result.Addons[0].Category != "A" {
		t.Fatalf("unexpected category: got %q", result.Addons[0].Category)
	}

	if result.Addons[1].URL != "https://gitlab.com/acme/baz" {
		t.Fatalf("unexpected URL: got %q", result.Addons[1].URL)
	}

	if result.Addons[1].Category != "A" {
		t.Fatalf("unexpected category: got %q", result.Addons[1].Category)
	}

	if result.Addons[2].URL != "https://github.com/owner/repo" {
		t.Fatalf("unexpected URL: got %q", result.Addons[2].URL)
	}

	if result.Addons[2].Category != "B" {
		t.Fatalf("unexpected category: got %q", result.Addons[2].Category)
	}
}

func jsonResponse(body, etag string) *http.Response {
	header := make(http.Header)
	if etag != "" {
		header.Set("ETag", etag)
	}

	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     header,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
