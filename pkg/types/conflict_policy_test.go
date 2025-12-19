package types_test

import (
	"testing"

	"github.com/stkevintan/miko/pkg/types"
)

func TestConflictPolicy(t *testing.T) {
	// Test valid policies
	validPolicies := []string{"skip", "overwrite", "rename", "update_tags"}
	for _, policy := range validPolicies {
		t.Run("Valid_"+policy, func(t *testing.T) {
			parsed, err := types.ParseConflictPolicy(policy)
			if err != nil {
				t.Errorf("Expected valid policy %q to parse without error, got: %v", policy, err)
			}
			if parsed.String() != policy {
				t.Errorf("Expected parsed policy to be %q, got %q", policy, parsed.String())
			}
			if !parsed.IsValid() {
				t.Errorf("Expected parsed policy %q to be valid", policy)
			}
		})
	}

	// Test empty string (should default to skip)
	t.Run("Empty_defaults_to_skip", func(t *testing.T) {
		parsed, err := types.ParseConflictPolicy("")
		if err != nil {
			t.Errorf("Expected empty string to parse without error, got: %v", err)
		}
		if parsed != types.ConflictPolicySkip {
			t.Errorf("Expected empty string to default to ConflictPolicySkip, got %q", parsed)
		}
	})

	// Test invalid policy
	t.Run("Invalid_policy", func(t *testing.T) {
		_, err := types.ParseConflictPolicy("invalid")
		if err == nil {
			t.Error("Expected invalid policy to return error")
		}
	})

	// Test enum constants
	t.Run("Enum_constants", func(t *testing.T) {
		if types.ConflictPolicySkip.String() != "skip" {
			t.Errorf("Expected ConflictPolicySkip to be 'skip', got %q", types.ConflictPolicySkip.String())
		}
		if !types.ConflictPolicyOverwrite.IsValid() {
			t.Error("Expected ConflictPolicyOverwrite to be valid")
		}
	})
}
