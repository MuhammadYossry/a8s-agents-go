package hub

import (
	"fmt"
	"strconv"
	"strings"
)

type Version struct {
	Major int
	Minor int
	Patch int
}

func ParseVersion(v string) (*Version, error) {
	// Remove 'v' prefix if present
	v = strings.TrimPrefix(v, "v")

	// Split version parts
	parts := strings.Split(v, ".")

	// Handle cases like "1.0" by adding .0
	if len(parts) == 2 {
		parts = append(parts, "0")
	}

	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid version format: %s", v)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid major version: %s", parts[0])
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid minor version: %s", parts[1])
	}

	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return nil, fmt.Errorf("invalid patch version: %s", parts[2])
	}

	return &Version{Major: major, Minor: minor, Patch: patch}, nil
}

func (v Version) String() string {
	return fmt.Sprintf("v%d.%d.%d", v.Major, v.Minor, v.Patch)
}

func (v Version) Compare(other Version) int {
	if v.Major != other.Major {
		return v.Major - other.Major
	}
	if v.Minor != other.Minor {
		return v.Minor - other.Minor
	}
	return v.Patch - other.Patch
}
