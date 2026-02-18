package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	// Find all subdirectories in routes/
	routesDir := "routes"
	var routePackages []string

	err := filepath.Walk(routesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the routes directory itself and hidden directories
		if path == routesDir || strings.HasPrefix(filepath.Base(path), ".") {
			return nil
		}

		// If it's a directory and contains .go files, it's a route package
		if info.IsDir() {
			goFiles, _ := filepath.Glob(filepath.Join(path, "*.go"))
			if len(goFiles) > 0 {
				// Convert path to package import path
				relPath, _ := filepath.Rel(routesDir, path)
				importPath := fmt.Sprintf("github.com/xihale/rsshub-go/routes/%s", relPath)
				routePackages = append(routePackages, importPath)
			}
		}

		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error walking routes directory: %v\n", err)
		os.Exit(1)
	}

	// Read cmd/server.go
	serverGo, err := os.ReadFile("cmd/server.go")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading cmd/server.go: %v\n", err)
		os.Exit(1)
	}

	// Find the marker and replace route imports
	marker := "\t// Import route packages to trigger init() registration"
	content := string(serverGo)

	markerIdx := strings.Index(content, marker)
	if markerIdx == -1 {
		fmt.Fprintf(os.Stderr, "Error: Marker comment not found in cmd/server.go\n")
		os.Exit(1)
	}

	// Find the end of the route imports (look for closing paren of import block)
	// We need to find the next ')' that's at the start of a line after the marker
	restOfContent := content[markerIdx:]
	lines := strings.Split(restOfContent, "\n")

	importEndIdx := markerIdx
	for i, line := range lines {
		if i == 0 {
			// Start after the marker line
			importEndIdx += len(line) + 1 // +1 for the newline
			continue
		}

		trimmed := strings.TrimSpace(line)
		if trimmed == ")" {
			// Found the closing paren - stop here (we'll keep this line)
			break
		}

		// Add this line's length to skip over it
		importEndIdx += len(line) + 1
	}

	// Build new import block
	var buf bytes.Buffer
	buf.WriteString("\t// Import route packages to trigger init() registration\n")
	for _, pkg := range routePackages {
		buf.WriteString(fmt.Sprintf("\t_ \"%s\"\n", pkg))
	}

	// Replace old imports with new ones (keep the closing paren line)
	newContent := content[:markerIdx] + buf.String() + content[importEndIdx:]

	// Write back to cmd/server.go
	if err := os.WriteFile("cmd/server.go", []byte(newContent), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing cmd/server.go: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generated %d route imports in cmd/server.go\n", len(routePackages))
}
