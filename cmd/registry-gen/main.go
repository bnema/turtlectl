// registry-gen is a standalone tool for generating the addon registry.
// It scrapes the Turtle WoW wiki and enriches addons with GitHub metadata.
// This tool is used by CI to keep the registry up to date.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/bnema/turtlectl/internal/wiki"
	"github.com/bnema/turtlectl/internal/wikigen"
)

func main() {
	outputPath := flag.String("output", "data/addons.json", "Output path for the registry JSON")
	flag.Parse()

	if err := run(*outputPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(outputPath string) error {
	fmt.Println("=== Addon Registry Generator ===")
	fmt.Println()

	// Load existing registry to preserve added_at dates and revision
	existing := loadExistingRegistry(outputPath)
	fmt.Printf("Loaded %d existing addons from registry (revision %d)\n", len(existing.Addons), existing.Revision)

	// Scrape wiki
	fmt.Println("Scraping Turtle WoW wiki...")
	scraper := wikigen.NewScraper()
	result, err := scraper.Scrape("")
	if err != nil {
		return fmt.Errorf("failed to scrape wiki: %w", err)
	}
	fmt.Printf("Found %d addon URLs\n", len(result.Addons))

	// Convert to WikiAddons
	enricher := wikigen.NewEnricher()
	addons := enricher.ConvertToAddons(result.Addons)

	// Merge with existing data (preserve added_at, update other fields)
	now := time.Now().UTC()
	newCount := 0
	for i := range addons {
		if existingAddon, ok := existing.Addons[addons[i].URL]; ok {
			// Preserve added_at from existing
			addons[i].AddedAt = existingAddon.AddedAt
		} else {
			// New addon
			addons[i].AddedAt = now
			newCount++
		}
	}
	fmt.Printf("New addons: %d\n", newCount)

	// Enrich with GitHub metadata using GraphQL
	fmt.Println()
	fmt.Println("Enriching addons with GitHub metadata (GraphQL)...")
	if enricher.IsAuthenticated() {
		fmt.Println("Using GitHub GraphQL API (batched queries)")
	} else {
		fmt.Println("ERROR: GITHUB_TOKEN required for GraphQL API")
		fmt.Println("Set GITHUB_TOKEN environment variable")
		return fmt.Errorf("GITHUB_TOKEN not set")
	}
	fmt.Println()

	startTime := time.Now()
	lastPrint := time.Now()
	enricher.EnrichAll(addons, func(current, total int, name string) {
		// Print progress every 50 addons or every 2 seconds
		if current%50 == 0 || time.Since(lastPrint) > 2*time.Second || current == total {
			elapsed := time.Since(startTime)
			rate := float64(current) / elapsed.Seconds()
			remaining := time.Duration(float64(total-current)/rate) * time.Second
			fmt.Printf("[%d/%d] %.1f/sec, ~%s remaining\n", current, total, rate, remaining.Round(time.Second))
			lastPrint = time.Now()
		}
	})
	fmt.Println()

	// Sort alphabetically
	sort.Slice(addons, func(i, j int) bool {
		return addons[i].Name < addons[j].Name
	})

	// Create registry data (increment revision)
	newRevision := existing.Revision + 1
	registry := wiki.RegistryData{
		Version:     wiki.RegistryVersion,
		Revision:    newRevision,
		GeneratedAt: now,
		SourceURL:   wikigen.WikiURL,
		AddonCount:  len(addons),
		Addons:      addons,
	}

	// Write output
	fmt.Printf("Writing registry to %s...\n", outputPath)
	data, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registry: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write registry: %w", err)
	}

	// Summary
	fmt.Println()
	fmt.Println("=== Summary ===")
	fmt.Printf("Revision:     %d\n", newRevision)
	fmt.Printf("Total addons: %d\n", len(addons))
	fmt.Printf("New addons:   %d\n", newCount)
	fmt.Printf("Generated:    %s\n", now.Format(time.RFC3339))
	fmt.Printf("Output:       %s\n", outputPath)

	return nil
}

// existingRegistry holds data from the previous registry
type existingRegistry struct {
	Addons   map[string]wiki.WikiAddon
	Revision int
}

// loadExistingRegistry loads the existing registry to preserve added_at dates and revision
func loadExistingRegistry(path string) existingRegistry {
	result := existingRegistry{
		Addons:   make(map[string]wiki.WikiAddon),
		Revision: 0,
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return result
	}

	var registry wiki.RegistryData
	if err := json.Unmarshal(data, &registry); err != nil {
		return result
	}

	result.Revision = registry.Revision
	for _, addon := range registry.Addons {
		result.Addons[addon.URL] = addon
	}

	return result
}
