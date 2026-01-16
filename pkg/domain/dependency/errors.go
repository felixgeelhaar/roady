package dependency

import "errors"

// Dependency domain errors.
var (
	// ErrCyclicDependency indicates a cycle was detected in the dependency graph.
	ErrCyclicDependency = errors.New("cyclic dependency detected")
	// ErrDependencyNotFound indicates a dependency was not found.
	ErrDependencyNotFound = errors.New("dependency not found")
	// ErrInvalidDependencyType indicates an invalid dependency type.
	ErrInvalidDependencyType = errors.New("invalid dependency type")
	// ErrRepoNotReachable indicates a repository path is not accessible.
	ErrRepoNotReachable = errors.New("repository not reachable")
	// ErrSelfDependency indicates a repo cannot depend on itself.
	ErrSelfDependency = errors.New("repository cannot depend on itself")
)
