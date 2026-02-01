package application

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ScanCodebaseTree returns a compact directory tree of source files relative to root.
// It skips hidden dirs, vendor, node_modules, and binary files.
// Output is truncated to maxLines to fit AI context windows.
func ScanCodebaseTree(root string, maxLines int) string {
	skipDirs := map[string]bool{
		".git": true, ".roady": true, "vendor": true, "node_modules": true,
		"__pycache__": true, ".idea": true, ".vscode": true, "dist": true, "build": true,
	}

	sourceExts := map[string]bool{
		".go": true, ".ts": true, ".tsx": true, ".js": true, ".jsx": true,
		".py": true, ".rs": true, ".java": true, ".rb": true, ".c": true,
		".h": true, ".cpp": true, ".cs": true, ".swift": true, ".kt": true,
		".yaml": true, ".yml": true, ".json": true, ".toml": true, ".sql": true,
		".proto": true, ".graphql": true, ".md": true,
	}

	var lines []string
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		if rel == "." {
			return nil
		}

		if info.IsDir() {
			if skipDirs[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(info.Name()))
		if !sourceExts[ext] {
			return nil
		}

		lines = append(lines, rel)
		if len(lines) >= maxLines {
			return fmt.Errorf("limit reached")
		}
		return nil
	})

	if len(lines) == 0 {
		return "(no source files found)"
	}

	result := strings.Join(lines, "\n")
	if len(lines) >= maxLines {
		result += fmt.Sprintf("\n... (truncated at %d files)", maxLines)
	}
	return result
}
