package main

import (
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// --- Helper Functions ---

// renderTemplate executes the named template with the given data.
// It centralizes error handling for template execution.
func renderTemplate(w http.ResponseWriter, tmplName string, data interface{}) {
	// Execute the template associated with the given name.
	err := templates.ExecuteTemplate(w, tmplName, data)
	if err != nil {
		// Log the detailed error on the server side.
		log.Printf("ERROR: Failed to execute template '%s': %v", tmplName, err)
		// Provide a generic error message to the client.
		// Avoid exposing internal details in the error response.
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// --- Route Handlers ---

// handleHome handles requests to the root path ("/")
// Method: GET
// Action: Displays a list of available profiles.
func handleHome(w http.ResponseWriter, r *http.Request) {
	// Ensure this handler only responds to the exact root path.
	if r.URL.Path != "/" {
		http.NotFound(w, r) // Return 404 for paths like "/something-else"
		return
	}
	// Ensure only GET requests are handled.
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	log.Printf("Handler: handleHome (GET %s)", r.URL.Path)

	// Load basic profile info (slug, title) for listing.
	profiles, err := loadProfiles()
	if err != nil {
		log.Printf("ERROR: Failed to load profiles for home page: %v", err)
		http.Error(w, "Could not load profiles", http.StatusInternalServerError)
		return
	}

	// Prepare data for the template.
	templateData := map[string]interface{}{
		"PageTitle": "Available Profiles",
		"Profiles":  profiles,
	}

	// Render the "list.html" template using the base "layout.html".
	renderTemplate(w, "layout.html", templateData)
}

// handleProfile handles requests to view a single profile ("/profile/{slug}")
// Method: GET
// Action: Displays the content of a specific profile identified by its slug.
func handleProfile(w http.ResponseWriter, r *http.Request) {
	// Ensure only GET requests are handled.
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract the profile slug from the URL path.
	// Example: /profile/example-profile -> "example-profile"
	// TrimPrefix removes the fixed part. TrimSuffix handles optional trailing slashes.
	slug := strings.TrimPrefix(r.URL.Path, "/profile/")
	slug = strings.TrimSuffix(slug, "/")

	// Basic validation: Ensure slug is not empty after trimming.
	if slug == "" {
		http.NotFound(w, r) // No slug provided
		return
	}

	log.Printf("Handler: handleProfile (GET %s, Slug: %s)", r.URL.Path, slug)

	// Load the full profile data using the extracted slug.
	profile, err := loadProfileBySlug(slug)
	if err != nil {
		// Check if the error is specifically "file not found".
		if os.IsNotExist(err) {
			log.Printf("INFO: Profile not found for slug '%s'", slug)
			http.NotFound(w, r) // Return 404 if profile doesn't exist.
		} else {
			// Log other errors (e.g., permission issues, markdown parsing errors).
			log.Printf("ERROR: Failed to load profile with slug '%s': %v", slug, err)
			http.Error(w, "Could not load profile", http.StatusInternalServerError)
		}
		return
	}

	// Prepare data for the template.
	templateData := map[string]interface{}{
		"PageTitle": profile.Title, // Use profile title for the page title
		"Profile":   profile,
	}

	// Render the "profile.html" template using the base "layout.html".
	renderTemplate(w, "layout.html", templateData)
}

// handleAdmin handles requests to the admin page ("/admin")
// Method: GET
// Action: Displays a list of existing profiles and an upload form.
func handleAdmin(w http.ResponseWriter, r *http.Request) {
	// Ensure only GET requests are handled.
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	// Ensure this handler only responds to the exact admin path.
	if r.URL.Path != "/admin" {
		http.NotFound(w, r)
		return
	}

	log.Printf("Handler: handleAdmin (GET %s)", r.URL.Path)

	// Load basic profile info for listing on the admin page.
	profiles, err := loadProfiles()
	if err != nil {
		log.Printf("ERROR: Failed to load profiles for admin page: %v", err)
		// Don't fail the request; render the page with an empty list or error message.
		profiles = []Profile{} // Ensure profiles is not nil for the template
		// Optionally, add an error message to templateData
	}

	// Prepare data for the template.
	templateData := map[string]interface{}{
		"PageTitle": "Admin Console",
		"Profiles":  profiles,
		// Add "ErrorMessage": "Could not load profiles" if err != nil (optional)
	}

	// Render the "admin.html" template using the base "layout.html".
	renderTemplate(w, "layout.html", templateData)
}

// handleAdminUpload handles profile uploads from the admin page.
// Method: POST
// Action: Processes the uploaded Markdown file and saves it to the content directory.
func handleAdminUpload(w http.ResponseWriter, r *http.Request) {
	// Ensure only POST requests are handled.
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	// Ensure this handler only responds to the exact upload path.
	if r.URL.Path != "/admin/upload" {
		http.NotFound(w, r)
		return
	}

	log.Printf("Handler: handleAdminUpload (POST %s)", r.URL.Path)

	// --- File Upload Processing ---

	// Set a maximum upload size (e.g., 10 MB) to prevent abuse.
	// ParseMultipartForm parses the multipart form data from the request.
	// The argument specifies the maximum bytes of memory to use for storing
	// file parts and form values. Larger files are stored in temporary files on disk.
	maxUploadSize := int64(10 * 1024 * 1024) // 10 MB
	err := r.ParseMultipartForm(maxUploadSize)
	if err != nil {
		log.Printf("ERROR: Failed parsing multipart form: %v", err)
		// Check if the error is due to the request body being too large.
		if err.Error() == "http: request body too large" {
			http.Error(w, fmt.Sprintf("File upload failed: File exceeds maximum size limit of %d MB", maxUploadSize/(1024*1024)), http.StatusBadRequest)
		} else {
			http.Error(w, "File upload failed: Error processing form data", http.StatusBadRequest)
		}
		return
	}

	// Retrieve the file from the form data using the input field's name ("profileFile").
	// FormFile returns the first file for the given key, the file header, and an error.
	file, handler, err := r.FormFile("profileFile")
	if err != nil {
		// Handle cases where the file field is missing or there's an error reading it.
		if err == http.ErrMissingFile {
			log.Println("WARNING: No file uploaded in 'profileFile' field.")
			http.Error(w, "File upload failed: No file was provided", http.StatusBadRequest)
		} else {
			log.Printf("ERROR: Failed retrieving uploaded file: %v", err)
			http.Error(w, "File upload failed: Could not retrieve file from form", http.StatusBadRequest)
		}
		return
	}
	// IMPORTANT: Always close the uploaded file handle when done.
	defer func(file multipart.File) {
		err := file.Close()
		if err != nil {
			log.Printf("ERROR: Failed to close uploaded file handle: %v", err)
		}
	}(file)

	log.Printf("INFO: Received upload: Filename=[%s], Size=[%d bytes], MIME Header=[%v]",
		handler.Filename, handler.Size, handler.Header)

	// --- Filename Sanitization and Validation ---

	// 1. Basic Security: Get only the filename part to prevent path traversal attacks.
	//    filepath.Base extracts the last element of the path.
	originalFilename := filepath.Base(handler.Filename)

	// 2. File Type Validation: Ensure the file has a ".md" extension (case-insensitive).
	if !strings.HasSuffix(strings.ToLower(originalFilename), ".md") {
		log.Printf("WARNING: Upload rejected. Invalid file type: '%s'. Only '.md' allowed.", originalFilename)
		http.Error(w, "Invalid file type: Only Markdown (.md) files are allowed.", http.StatusBadRequest)
		return
	}

	// 3. Create a Safe Filename/Slug:
	//    - Remove the ".md" extension.
	//    - Convert to lowercase.
	//    - Replace spaces and non-alphanumeric characters with hyphens.
	//    - Remove leading/trailing hyphens and collapse multiple hyphens.
	safeSlug := strings.TrimSuffix(strings.ToLower(originalFilename), ".md")
	// Regex to match any character that is NOT a lowercase letter, number, or hyphen
	reg := regexp.MustCompile("[^a-z0-9-]+")
	safeSlug = reg.ReplaceAllString(safeSlug, "-") // Replace unwanted chars with hyphen
	// Regex to collapse multiple consecutive hyphens into one
	reg = regexp.MustCompile("-+")
	safeSlug = reg.ReplaceAllString(safeSlug, "-")
	safeSlug = strings.Trim(safeSlug, "-") // Remove leading/trailing hyphens

	// Handle edge case where the name becomes empty after sanitization.
	if safeSlug == "" {
		safeSlug = "untitled-profile" // Provide a default name
	}
	finalFilename := safeSlug + ".md" // Add the extension back

	// Construct the full destination path within the designated content directory.
	dstPath := filepath.Join(contentDir, finalFilename)
	log.Printf("INFO: Sanitized filename: '%s', Destination path: '%s'", finalFilename, dstPath)

	// --- Check for Existing File ---
	// Prevent overwriting existing files by default for v0.1.
	if _, err := os.Stat(dstPath); err == nil {
		// File exists (no error means Stat succeeded)
		log.Printf("WARNING: Upload rejected. File '%s' already exists.", finalFilename)
		http.Error(w, fmt.Sprintf("File '%s' already exists. Upload cancelled.", finalFilename), http.StatusConflict) // 409 Conflict
		return
	} else if !os.IsNotExist(err) {
		// An error occurred during Stat, but it wasn't "file not found".
		// This could indicate a permissions issue or other problem.
		log.Printf("ERROR: Could not check for existing file '%s': %v", dstPath, err)
		http.Error(w, "Could not process upload due to a server error.", http.StatusInternalServerError)
		return
	}

	// --- Save the Uploaded File ---

	// Create the destination file on the server's filesystem.
	// os.Create creates the file or truncates it if it already exists (though we checked above).
	dst, err := os.Create(dstPath)
	if err != nil {
		log.Printf("ERROR: Failed to create destination file '%s': %v", dstPath, err)
		http.Error(w, "Could not save uploaded file", http.StatusInternalServerError)
		return
	}
	// Ensure the destination file is closed, even if errors occur during copy.
	defer func(dst *os.File) {
		err := dst.Close()
		if err != nil {
			log.Printf("ERROR: Failed to close destination file handle '%s': %v", dstPath, err)
		}
	}(dst)

	// Copy the content from the uploaded file (source) to the destination file.
	// io.Copy efficiently handles transferring the data.
	bytesCopied, err := io.Copy(dst, file)
	if err != nil {
		log.Printf("ERROR: Failed copying uploaded file content to '%s': %v", dstPath, err)
		// Attempt to remove the partially written file to avoid corruption.
		removeErr := os.Remove(dstPath)
		if removeErr != nil {
			log.Printf("ERROR: Failed to remove partially written file '%s' after copy error: %v", dstPath, removeErr)
		}
		http.Error(w, "Could not copy file content", http.StatusInternalServerError)
		return
	}

	log.Printf("SUCCESS: Uploaded '%s' (%d bytes) saved as '%s'", originalFilename, bytesCopied, finalFilename)

	// --- Redirect After Success ---
	// Redirect the user back to the admin page after a successful upload.
	// http.StatusSeeOther (303) is appropriate for POST-redirect-GET pattern.
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}
