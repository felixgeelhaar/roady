package billing

// Repository handles the persistence of billing artifacts.
// New code should depend on this interface rather than the composed WorkspaceRepository.
type Repository interface {
	SaveRates(config *RateConfig) error
	LoadRates() (*RateConfig, error)
	SaveTimeEntries(entries []TimeEntry) error
	LoadTimeEntries() ([]TimeEntry, error)
}
