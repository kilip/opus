// White-box testing is used to access package-level unexported variables like rootCmd and package-level function Execute.
package main

import (
	"io"
	"testing"
)

func TestRootExecute(t *testing.T) {
	// Set CLI args to --help to verify parsing without starting the actual Fiber server
	RootCmd().SetArgs([]string{"--help"})
	RootCmd().SetOut(io.Discard)
	RootCmd().SetErr(io.Discard)
	t.Cleanup(func() {
		RootCmd().SetArgs(nil)
		RootCmd().SetOut(nil)
		RootCmd().SetErr(nil)
	})
	err := Execute()
	if err != nil {
		t.Errorf("Expected no error from root command execution, got %v", err)
	}
}
