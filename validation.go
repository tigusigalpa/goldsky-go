package goldsky

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	subgraphNameRe    = regexp.MustCompile(`^[a-zA-Z][\w-]*$`)
	subgraphVersionRe = regexp.MustCompile(`^[a-zA-Z0-9][\w+.-]*$`)
)

func validateSubgraphTarget(name, version string) error {
	if !subgraphNameRe.MatchString(name) {
		return fmt.Errorf("invalid subgraph name %q: must start with a letter and contain only letters, numbers, underscores, and hyphens", name)
	}
	if version != "" && !subgraphVersionRe.MatchString(version) {
		return fmt.Errorf("invalid subgraph version or tag %q", version)
	}
	return nil
}

func validateResourceName(kind, name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("goldsky: %s name is required", kind)
	}
	return nil
}
