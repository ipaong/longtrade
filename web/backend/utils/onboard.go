package utils

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var execCommand = exec.Command

func EnsureOnboarded(configPath string) error {
	_, err := os.Stat(configPath)
	if err == nil {
		return nil
	}
	if !os.IsNotExist(err) {
		return fmt.Errorf("stat config: %w", err)
	}

	// Run onboarding non-interactively. The "--yes" flag accepts the terms,
	// skips credential encryption, and skips portfolio prompts. Piping stdin
	// is intentionally avoided: the legal agreement reads stdin via a buffered
	// reader, so any piped answer would be consumed there and the wrong prompt
	// would receive the input.
	cmd := execCommand(FindKhunquantBinary(), "onboard", "--yes")
	cmd.Env = append(os.Environ(), "KHUNQUANT_CONFIG="+configPath)

	output, err := cmd.CombinedOutput()
	if err != nil {
		trimmed := strings.TrimSpace(string(output))
		if trimmed == "" {
			return fmt.Errorf("run onboard: %w", err)
		}
		return fmt.Errorf("run onboard: %w: %s", err, trimmed)
	}

	if _, err := os.Stat(configPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("onboard completed but did not create config %s", configPath)
		}
		return fmt.Errorf("verify config after onboard: %w", err)
	}

	return nil
}
