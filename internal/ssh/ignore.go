package ssh

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// IgnoreFile is the project file that adjusts what gets uploaded.
const IgnoreFile = ".shipignore"

// IgnoreRules is a deliberately small subset of .gitignore: one glob per line,
// '#' comments, and '!' to upload something Ship would otherwise skip. A pattern
// with a slash matches the path from the project root; without one it matches a
// name at any depth. "**" is not supported.
type IgnoreRules struct {
	Exclude []string
	Include []string
}

// LoadIgnore reads .shipignore from dir. A missing file means no rules.
func LoadIgnore(dir string) (IgnoreRules, error) {
	var rules IgnoreRules
	data, err := os.ReadFile(filepath.Join(dir, IgnoreFile))
	if os.IsNotExist(err) {
		return rules, nil
	}
	if err != nil {
		return rules, fmt.Errorf("reading %s: %w", IgnoreFile, err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		include := strings.HasPrefix(line, "!")
		pattern := strings.Trim(strings.TrimPrefix(line, "!"), "/")
		if _, err := path.Match(pattern, ""); err != nil || pattern == "" || strings.Contains(pattern, "**") {
			return rules, fmt.Errorf("%s: unsupported pattern %q", IgnoreFile, line)
		}
		if include {
			rules.Include = append(rules.Include, pattern)
		} else {
			rules.Exclude = append(rules.Exclude, pattern)
		}
	}
	return rules, nil
}

func matchAny(patterns []string, rel string) bool {
	for _, p := range patterns {
		target := rel
		if !strings.Contains(p, "/") {
			target = path.Base(rel)
		}
		if ok, _ := path.Match(p, target); ok {
			return true
		}
	}
	return false
}
