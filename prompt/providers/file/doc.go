// Package file provides a filesystem-based PromptManager that loads prompt
// templates from a directory of JSON files. Each JSON file represents a single
// template version with name, version, content, and variables fields.
//
// # FileManager
//
// FileManager implements prompt.PromptManager by loading templates from JSON
// files in a directory. Files must have a .json extension and contain a valid
// prompt.Template structure. All files are parsed on creation.
//
// Templates are organized by name, with multiple versions supported per name.
// When retrieving a template without specifying a version, the latest version
// (lexicographically highest) is returned.
//
// # Template File Format
//
// Each JSON file should contain:
//
//	{
//	    "name": "greeting",
//	    "version": "1.0.0",
//	    "content": "Hello, {{.name}}!",
//	    "variables": {"name": "World"}
//	}
//
// # Usage
//
//	mgr, err := file.NewFileManager("/path/to/prompts")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Get a specific template version
//	tmpl, err := mgr.Get("greeting", "1.0.0")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Render the latest version with variables
//	msgs, err := mgr.Render("greeting", map[string]any{"name": "Alice"})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// List all available templates
//	infos := mgr.List()
package file
