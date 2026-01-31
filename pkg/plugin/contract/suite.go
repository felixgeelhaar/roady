package contract

import (
	"fmt"

	domainPlugin "github.com/felixgeelhaar/roady/pkg/domain/plugin"
	infraPlugin "github.com/felixgeelhaar/roady/pkg/plugin"
)

// ContractSuite runs all contract assertions against a plugin binary.
type ContractSuite struct {
	loader *infraPlugin.Loader
}

// NewContractSuite creates a new contract suite.
func NewContractSuite() *ContractSuite {
	return &ContractSuite{
		loader: infraPlugin.NewLoader(),
	}
}

// SuiteResult aggregates results from running the full contract suite.
type SuiteResult struct {
	Results []Result
	Passed  int
	Failed  int
}

// RunWithSyncer runs the contract suite against an already-loaded syncer instance.
func (s *ContractSuite) RunWithSyncer(syncer domainPlugin.Syncer) *SuiteResult {
	assertions := []func(domainPlugin.Syncer) Result{
		AssertInitSuccess,
		AssertInitWithBadConfig,
		AssertSyncEmptyPlan,
		AssertSyncWithTasks,
		AssertPushValidTask,
		AssertPushInvalidTask,
	}

	sr := &SuiteResult{}
	for _, assert := range assertions {
		result := assert(syncer)
		sr.Results = append(sr.Results, result)
		if result.Passed {
			sr.Passed++
		} else {
			sr.Failed++
		}
	}
	return sr
}

// RunBinary loads a plugin binary and runs the full contract suite.
func (s *ContractSuite) RunBinary(path string) (*SuiteResult, error) {
	defer s.loader.Cleanup()

	syncer, err := s.loader.Load(path)
	if err != nil {
		return nil, fmt.Errorf("load plugin: %w", err)
	}

	return s.RunWithSyncer(syncer), nil
}
