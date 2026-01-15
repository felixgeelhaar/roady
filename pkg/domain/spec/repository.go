package spec

// Repository handles persistence of product specifications.
type Repository interface {
	Save(spec *ProductSpec) error
	Load() (*ProductSpec, error)
	SaveLock(spec *ProductSpec) error
	LoadLock() (*ProductSpec, error)
}
