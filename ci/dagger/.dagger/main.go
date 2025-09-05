// Integration test runner for the repository
//
// This module provides functions to run integration tests inside containers
// using the Dagger platform for consistent, reproducible test execution.

package main

import (
	"context"
	"dagger/integration-tests/internal/dagger"
)

type IntegrationTests struct{}

// Run integration tests in a Ubuntu container
func (m *IntegrationTests) Test(ctx context.Context, source *dagger.Directory) (string, error) {
	return m.testContainer(source).
		WithExec([]string{"bash", "test/containers/test-validator/run-integration-tests.sh"}).
		Stdout(ctx)
}

// Returns a container configured for running integration tests
func (m *IntegrationTests) TestContainer(source *dagger.Directory) *dagger.Container {
	return m.testContainer(source)
}

// Internal helper to create the test container
func (m *IntegrationTests) testContainer(source *dagger.Directory) *dagger.Container {
	return dag.Container().
		From("golang:1.25").
		WithExec([]string{"apt-get", "update"}).
		WithExec([]string{"apt-get", "install", "-y", "git", "ffmpeg"}).
		WithDirectory("/work", source).
		WithWorkdir("/work")
}
