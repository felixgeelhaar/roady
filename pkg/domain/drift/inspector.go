package drift

// CodeInspector defines the interface for inspecting the actual codebase.
type CodeInspector interface {
	// FileExists checks if a file exists at the given path.
	FileExists(path string) (bool, error)
	// FileNotEmpty checks if the file has content (size > 0).
	FileNotEmpty(path string) (bool, error)
	// GitStatus returns the git status of the file ("clean", "modified", "untracked", "ignored", "missing", or "error").
	GitStatus(path string) (string, error)
}
