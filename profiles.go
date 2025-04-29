package main

import (
	"bytes"
	"errors"
	"fmt"
	"html/template" // Use html/template for safe HTML rendering
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync" // For thread-safe Markdown parser initialization

	// External dependencies
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension" // For extensions like GFM, tables, etc.
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"gopkg.in/yaml.v3" // For parsing YAML front matter
)

// --- Struct Definitions ---

// Profile struct holds the data for a single research profile.
type Profile struct {
	Slug        string        // URL-friendly identifier (e.g., "example-profile")
	Title       string        // Human-readable title (from front matter or filename)
	Markdown    string        // Raw Markdown content (optional, useful for editing later)
	HTMLContent template.HTML // Rendered HTML content, marked safe for templates
}

// FrontMatter struct defines the expected fields in the YAML front matter block.
// `yaml:"title"` maps the struct field Title to the YAML key "title".
type FrontMatter struct {
	Title string `yaml:"title"`
	// Add other fields here if needed, e.g.:
	// Author string `yaml:"author"`
	// Date   string `yaml:"date"`
}

// --- Constants and Globals ---

// frontMatterSeparator defines the standard separator for YAML front matter.
const frontMatterSeparator = "---"

// mdParser is the configured Goldmark Markdown parser instance.
// Use sync.Once to ensure it's initialized safely only once, even with concurrent requests.
var (
	mdParser goldmark.Markdown
	once     sync.Once
)

// initializeMarkdownParser sets up the Goldmark parser with desired extensions and options.
func initializeMarkdownParser() {
	mdParser = goldmark.New(
		// Enable commonly used Markdown extensions
		goldmark.WithExtensions(
			extension.GFM,      // GitHub Flavored Markdown (includes tables, strikethrough, etc.)
			extension.Footnote, // Support for footnotes
			extension.Linkify,  // Autolink URLs
			// Add other extensions as needed: extension.TaskList, extension.Typographer, ...
		),
		// Configure the parser behavior
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(), // Automatically generate IDs for headings (e.g., for linking)
			// parser.WithAttribute(),  // Enable custom attributes like { #myid .myclass } (optional)
		),
		// Configure the HTML renderer behavior
		goldmark.WithRendererOptions(
			html.WithHardWraps(), // Render single newlines in Markdown as <br> tags
			html.WithXHTML(),     // Output XHTML compliant HTML (e.g., self-closing tags <br/>)
			html.WithUnsafe(),    // *** IMPORTANT SECURITY NOTE ***
			// Allows raw HTML blocks within Markdown to be rendered.
			// This is necessary for flexibility but can be a security risk
			// if the Markdown content comes from untrusted sources.
			// If content is user-generated, consider using a sanitizer
			// like bluemonday AFTER rendering to strip potentially
			// malicious HTML/JS: https://github.com/microcosm-cc/bluemonday
		),
	)
	log.Println("Markdown parser initialized.")
}

// getMarkdownParser returns the initialized Goldmark parser instance.
func getMarkdownParser() goldmark.Markdown {
	once.Do(initializeMarkdownParser) // Ensures initialization happens only once
	return mdParser
}

// --- Core Logic Functions ---

// parseMarkdownAndFrontMatter attempts to extract YAML front matter and the main Markdown body
// from the raw byte content of a file.
// It returns the parsed FrontMatter, the remaining Markdown body as a string, and any error encountered
// during YAML parsing (though it tries to recover gracefully).
func parseMarkdownAndFrontMatter(content []byte) (FrontMatter, string, error) {
	var fm FrontMatter
	var markdownBody string
	var yamlParseError error // Store potential YAML error but don't fail immediately

	// Convert []byte separator to string for easier use
	sep := "\n" + frontMatterSeparator + "\n"
	contentStr := string(content)

	// Check if the content starts with the front matter separator.
	if strings.HasPrefix(contentStr, frontMatterSeparator+"\n") {
		// Find the end of the front matter block.
		// Add 1 to index to search *after* the initial '---'.
		endIdx := strings.Index(contentStr[len(frontMatterSeparator)+1:], sep)

		if endIdx != -1 {
			// Adjust endIdx to be relative to the original string
			actualEndIdx := endIdx + len(frontMatterSeparator) + 1

			// Extract the YAML block (between the '---' separators)
			yamlBlock := contentStr[len(frontMatterSeparator)+1 : actualEndIdx]

			// Extract the Markdown body (everything after the second '---')
			markdownBody = strings.TrimSpace(contentStr[actualEndIdx+len(sep):])

			// Attempt to unmarshal the YAML block into the FrontMatter struct.
			err := yaml.Unmarshal([]byte(yamlBlock), &fm)
			if err != nil {
				// Log a warning but don't treat it as a fatal error for the whole file.
				// The content will be treated as if it had no valid front matter.
				log.Printf("WARNING: Failed to parse YAML front matter: %v. Content will be treated as pure markdown.", err)
				yamlParseError = err // Store the error
				// Reset markdownBody to the original content minus the *initial* separator line,
				// as the YAML was invalid. This might not be perfect but is a reasonable fallback.
				markdownBody = strings.TrimSpace(contentStr[len(frontMatterSeparator)+1:])
			}
		} else {
			// Found the starting '---' but not the closing one. Treat everything after
			// the first line as Markdown.
			log.Printf("WARNING: Found starting '---' but no closing separator. Treating content as pure markdown.")
			markdownBody = strings.TrimSpace(contentStr[len(frontMatterSeparator)+1:])
		}
	} else {
		// No front matter separator found at the beginning. Treat the entire content as Markdown.
		markdownBody = contentStr
	}

	// If the Title wasn't successfully parsed from front matter, ensure it's empty.
	// It will be populated from the slug later if necessary.
	if fm.Title == "" {
		// Title remains empty, will use slug as fallback later.
	}

	return fm, markdownBody, yamlParseError // Return the potential YAML error
}

// loadProfiles reads the content directory, identifies Markdown files,
// and loads basic information (Slug, Title) for each profile.
// It's optimized for listing profiles without reading/parsing full content initially.
func loadProfiles() ([]Profile, error) {
	var profiles []Profile

	// Read all entries (files and directories) in the content directory.
	files, err := os.ReadDir(contentDir)
	if err != nil {
		// If the directory doesn't exist (e.g., first run before any uploads),
		// return an empty list and no error.
		if os.IsNotExist(err) {
			log.Printf("INFO: Content directory '%s' does not exist yet. Returning empty profile list.", contentDir)
			return profiles, nil // Return empty slice, not an error
		}
		// For other errors (e.g., permissions), return the error.
		return nil, fmt.Errorf("failed to read content directory '%s': %w", contentDir, err)
	}

	log.Printf("INFO: Scanning directory '%s' for profiles...", contentDir)
	count := 0
	// Iterate over the directory entries.
	for _, file := range files {
		// Skip directories and files that don't end with ".md" (case-insensitive).
		if !file.IsDir() && strings.HasSuffix(strings.ToLower(file.Name()), ".md") {
			count++
			filename := file.Name()
			// Determine the slug by removing the ".md" extension.
			slug := strings.TrimSuffix(filename, filepath.Ext(filename))
			filePath := filepath.Join(contentDir, filename)

			// --- Extract Title (Optimized) ---
			// Read just enough of the file to potentially parse the front matter title.
			// This avoids reading large files entirely just for the list view.
			// We still need to read some content to check for front matter.
			// Reading the whole file is simpler for v0.1, but optimization could be added here.
			content, err := os.ReadFile(filePath)
			if err != nil {
				// Log a warning if a specific file can't be read, but continue processing others.
				log.Printf("WARNING: Failed to read file '%s' for title extraction: %v. Using slug as title.", filename, err)
				// Add the profile with the slug as a fallback title.
				profiles = append(profiles, Profile{Slug: slug, Title: slug})
				continue // Skip to the next file
			}

			// Parse front matter to get the title.
			fm, _, parseErr := parseMarkdownAndFrontMatter(content)
			// parseErr is logged inside parseMarkdownAndFrontMatter if YAML is invalid.

			// Determine the final title. Use front matter title if available, otherwise use the slug.
			title := fm.Title
			if title == "" {
				title = slug // Fallback to using the slug as the title
				if parseErr != nil {
					log.Printf("INFO: Using slug '%s' as title for file '%s' (YAML parse error: %v)", slug, filename, parseErr)
				} else {
					log.Printf("INFO: Using slug '%s' as title for file '%s' (no title in front matter)", slug, filename)
				}
			}

			// Add the profile (with Slug and Title only) to the list.
			profiles = append(profiles, Profile{Slug: slug, Title: title})
		}
	}
	log.Printf("INFO: Found %d profile(s) in '%s'.", count, contentDir)

	return profiles, nil
}

// loadProfileBySlug loads the full content of a single profile identified by its slug.
// It reads the file, parses front matter, renders the Markdown content to HTML,
// and returns the complete Profile struct.
func loadProfileBySlug(slug string) (*Profile, error) {
	// --- Input Validation ---
	// Basic slug validation to prevent directory traversal and invalid characters.
	// Disallow dots, slashes, and backslashes. Ensure it's not empty.
	if slug == "" || strings.ContainsAny(slug, "./\\") {
		log.Printf("ERROR: Attempted to load profile with invalid slug format: '%s'", slug)
		return nil, errors.New("invalid profile identifier") // Generic error for security
	}

	// Construct the expected filename and full path.
	filename := slug + ".md"
	filePath := filepath.Join(contentDir, filename)
	log.Printf("INFO: Attempting to load profile from path: '%s'", filePath)

	// --- File Reading ---
	// Check if the file exists first using os.Stat. This provides a clearer
	// "not found" error compared to relying solely on ReadFile's error.
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		log.Printf("INFO: Profile file not found at path: '%s'", filePath)
		// Return the os.IsNotExist error directly so the handler can check for it.
		return nil, err
	} else if err != nil {
		// Handle other potential errors during Stat (e.g., permission denied).
		log.Printf("ERROR: Failed to stat profile file '%s': %v", filePath, err)
		return nil, fmt.Errorf("failed to access profile file '%s': %w", filename, err)
	}

	// Read the entire file content into memory.
	content, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("ERROR: Failed to read profile file content '%s': %v", filePath, err)
		return nil, fmt.Errorf("failed to read profile file '%s': %w", filename, err)
	}

	// --- Parsing and Rendering ---

	// Parse front matter and separate the Markdown body.
	fm, markdownBody, parseErr := parseMarkdownAndFrontMatter(content)
	// parseErr is logged inside the function if YAML is invalid.

	// Determine the final title (front matter or slug fallback).
	title := fm.Title
	if title == "" {
		title = slug // Use slug if title is missing
		if parseErr != nil {
			log.Printf("INFO: Using slug '%s' as title for file '%s' (YAML parse error: %v)", slug, filename, parseErr)
		} else {
			log.Printf("INFO: Using slug '%s' as title for file '%s' (no title in front matter)", slug, filename)
		}
	}

	// Render the Markdown body to HTML using the configured Goldmark parser.
	var htmlBuffer bytes.Buffer   // Use a buffer to capture the HTML output.
	parser := getMarkdownParser() // Get the initialized parser instance
	if err := parser.Convert([]byte(markdownBody), &htmlBuffer); err != nil {
		log.Printf("ERROR: Failed to render markdown content for '%s': %v", filename, err)
		return nil, fmt.Errorf("failed to render markdown for '%s': %w", filename, err)
	}

	// --- Construct Result ---
	// Create the Profile struct with all the loaded and processed data.
	// Wrap the rendered HTML in template.HTML to mark it as safe for direct
	// inclusion in HTML templates (prevents double-escaping).
	profile := &Profile{
		Slug:        slug,
		Title:       title,
		Markdown:    markdownBody, // Store the raw Markdown body (optional)
		HTMLContent: template.HTML(htmlBuffer.String()),
	}

	log.Printf("INFO: Successfully loaded and rendered profile for slug '%s'.", slug)
	return profile, nil
}
