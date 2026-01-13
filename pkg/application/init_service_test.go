package application_test

import (
	"errors"
	"testing"

	"github.com/felixgeelhaar/roady/pkg/application"
)

func TestInitService_Detailed(t *testing.T) {
	// 1. Success
	repo := &MockRepo{Initialized: false}
	audit := application.NewAuditService(repo)
	service := application.NewInitService(repo, audit)

	if err := service.InitializeProject("test"); err != nil {
		t.Fatal(err)
	}
	if !repo.Initialized {
		t.Error("Should be initialized")
	}

	// 2. Already initialized
	if err := service.InitializeProject("test"); err == nil {
		t.Error("Expected error for already initialized repo")
	}

	// 3. Save Errors
	repo.Initialized = false
	repo.SaveError = errors.New("save error")
	if err := service.InitializeProject("test"); err == nil {
		t.Error("Expected error on save failure")
	}
}