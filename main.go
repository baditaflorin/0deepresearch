package main

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

// Global variables (consider dependency injection for larger apps)
var (
	templates   *template.Template // Parsed HTML templates
	contentDir  = "content"        // Directory for Markdown profile files
	templateDir = "templates"      // Directory for HTML template files
	staticDir   = "static"         // Directory for static assets (CSS, JS)
)

func main() {
	// --- Pre-computation and Setup ---

	// Ensure the content directory exists, create if not
	if _, err := os.Stat(contentDir); os.IsNotExist(err) {
		log.Printf("Content directory '%s' not found, creating it.", contentDir)
		err = os.Mkdir(contentDir, 0755) // Use standard directory permissions
		if err != nil {
			log.Fatalf("FATAL: Failed to create content directory '%s': %v", contentDir, err)
		}
	}

	// Parse all HTML templates from the templates directory on startup.
	// Using ParseGlob is convenient but panics if templates are invalid.
	// It's generally better to handle this error gracefully.
	var err error
	templates, err = template.ParseGlob(filepath.Join(templateDir, "*.html"))
	if err != nil {
		log.Fatalf("FATAL: Error parsing HTML templates in '%s': %v", templateDir, err)
	}
	log.Println("HTML templates parsed successfully.")

	// --- Server Setup ---

	// Create a new ServeMux (HTTP request router)
	mux := http.NewServeMux()

	// Setup file server for static assets (CSS, JS, images)
	// http.Dir specifies the root directory for the file server.
	// http.StripPrefix removes the "/static/" prefix from the request path
	// so the file server looks for files relative to the 'static' directory.
	// Example: Request to /static/style.css -> File server looks for static/style.css
	fs := http.FileServer(http.Dir(staticDir))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))
	log.Printf("Serving static files from '/%s/' directory.", staticDir)

	// Register application route handlers
	mux.HandleFunc("/", handleHome)                    // Root path for listing profiles
	mux.HandleFunc("/profile/", handleProfile)         // Path for viewing a single profile (expects /profile/slug-name)
	mux.HandleFunc("/admin", handleAdmin)              // Path for the admin view (GET)
	mux.HandleFunc("/admin/upload", handleAdminUpload) // Path for handling profile uploads (POST)
	log.Println("Registered application routes.")

	// --- Start Server ---

	port := ":8080" // Port to listen on
	log.Printf("Starting HTTP server on http://localhost%s", port)

	// Start the HTTP server. ListenAndServe blocks until the server stops.
	// It logs errors internally if it fails to start (e.g., port already in use).
	err = http.ListenAndServe(port, mux)
	if err != nil {
		// This error usually occurs if the server fails *after* starting successfully.
		log.Fatalf("FATAL: HTTP server ListenAndServe error: %v", err)
	}
}
