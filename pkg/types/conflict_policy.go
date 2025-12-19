package types

import "fmt"

// ConflictPolicy defines how to handle existing files during download
type ConflictPolicy string

const (
	// ConflictPolicySkip skips download if file already exists
	ConflictPolicySkip ConflictPolicy = "skip"
	// ConflictPolicyOverwrite overwrites existing files
	ConflictPolicyOverwrite ConflictPolicy = "overwrite"
	// ConflictPolicyRename creates a new file with a numeric suffix
	ConflictPolicyRename ConflictPolicy = "rename"
	// ConflictPolicyUpdateTags skips download but updates metadata tags
	ConflictPolicyUpdateTags ConflictPolicy = "update_tags"
)

// String returns the string representation of ConflictPolicy
func (cp ConflictPolicy) String() string {
	return string(cp)
}

// IsValid checks if the ConflictPolicy is valid
func (cp ConflictPolicy) IsValid() bool {
	switch cp {
	case ConflictPolicySkip, ConflictPolicyOverwrite, ConflictPolicyRename, ConflictPolicyUpdateTags:
		return true
	default:
		return false
	}
}

// ValidConflictPolicies returns a slice of all valid conflict policies
func ValidConflictPolicies() []ConflictPolicy {
	return []ConflictPolicy{
		ConflictPolicySkip,
		ConflictPolicyOverwrite,
		ConflictPolicyRename,
		ConflictPolicyUpdateTags,
	}
}

// ParseConflictPolicy parses a string into a ConflictPolicy with validation
func ParseConflictPolicy(s string) (ConflictPolicy, error) {
	if s == "" {
		return ConflictPolicySkip, nil // default value
	}

	policy := ConflictPolicy(s)
	if !policy.IsValid() {
		return "", fmt.Errorf("invalid conflict policy: %q, valid values are: %v", s, ValidConflictPolicies())
	}

	return policy, nil
}
