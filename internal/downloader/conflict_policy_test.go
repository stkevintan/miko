package downloader

import "testing"

func TestConflictPolicy(t *testing.T) {
	// Test valid policies
	validPolicies := []string{"skip", "overwrite", "rename", "update_tags"}
	for _, policy := range validPolicies {
		t.Run("Valid_"+policy, func(t *testing.T) {
			parsed, err := ParseConflictPolicy(policy)
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
		parsed, err := ParseConflictPolicy("")
		if err != nil {
			t.Errorf("Expected empty string to parse without error, got: %v", err)
		}
		if parsed != ConflictPolicySkip {
			t.Errorf("Expected empty string to default to ConflictPolicySkip, got %q", parsed)
		}
	})

	// Test invalid policy
	t.Run("Invalid_policy", func(t *testing.T) {
		_, err := ParseConflictPolicy("invalid")
		if err == nil {
			t.Error("Expected invalid policy to return error")
		}
	})

	// Test enum constants
	t.Run("Enum_constants", func(t *testing.T) {
		if ConflictPolicySkip.String() != "skip" {
			t.Errorf("Expected ConflictPolicySkip to be 'skip', got %q", ConflictPolicySkip.String())
		}
		if !ConflictPolicyOverwrite.IsValid() {
			t.Error("Expected ConflictPolicyOverwrite to be valid")
		}
	})
}
