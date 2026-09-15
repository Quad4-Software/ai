// SPDX-License-Identifier: 0BSD
package kaneo

import (
	"fmt"
	"net/url"
	"os/exec"
	"strings"
)

// Secret storage. The API key lives in the OS keyring, never in the
// config file. Two backends are supported through their CLIs so the
// server stays dependency-free:
//
//   - secret-tool (libsecret / Secret Service): GNOME Keyring, KWallet
//   - pass (the standard unix password store)
//
// Resolution order in LoadConfig: KANEO_API_KEY env, then the keyring,
// then pass. A plaintext apiKey found in config.json is migrated into
// the first available backend and stripped from the file.
//
// The lookups shell out with argv forms only. The key travels on stdin
// for stores and stdout for lookups, never in argv or tool output.

var lookPath = exec.LookPath

// keyAccount scopes the stored secret to the API instance so several
// Kaneo servers can coexist under one login keyring.
func keyAccount(apiURL string) string {
	u, err := url.Parse(apiURL)
	if err != nil || u.Host == "" {
		return apiURL
	}
	return u.Host
}

// runSecretTool invokes secret-tool. stdin is piped for store calls.
var runSecretTool = func(args []string, stdin string) (string, error) {
	// #nosec G204 -- fixed binary, argv built from fixed attribute names
	// plus the configured API host; the key travels on stdin only
	cmd := exec.Command("secret-tool", args...)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("secret-tool: %w", err)
	}
	return string(out), nil
}

// runPass invokes pass(1) with the same discipline.
var runPass = func(args []string, stdin string) (string, error) {
	// #nosec G204 -- fixed binary, argv built from fixed strings plus
	// the configured API host; the key travels on stdin only
	cmd := exec.Command("pass", args...)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("pass: %w", err)
	}
	return string(out), nil
}

// Backend names reported by auth_status and setup.
const (
	BackendEnv     = "env"
	BackendKeyring = "secret-service"
	BackendPass    = "pass"
	BackendNone    = ""
)

// DetectBackend returns the name of the first usable secret backend.
// The env backend is not returned here; it is a source, not a store.
// A failed store in setup surfaces the real error to the user.
func DetectBackend() string {
	if _, err := lookPath("secret-tool"); err == nil {
		return BackendKeyring
	}
	if _, err := lookPath("pass"); err == nil {
		return BackendPass
	}
	return BackendNone
}

// StoreKey writes the key into the named backend.
func StoreKey(backend, apiURL, key string) error {
	acct := keyAccount(apiURL)
	switch backend {
	case BackendKeyring:
		label := fmt.Sprintf("Kaneo API key (%s)", acct)
		_, err := runSecretTool([]string{
			"store", "--label=" + label, "service", "kaneo", "account", acct,
		}, key)
		return err
	case BackendPass:
		_, err := runPass([]string{"insert", "-m", "-f", "kaneo/" + acct}, key)
		return err
	default:
		return fmt.Errorf("no secret backend available; install libsecret (secret-tool) or pass")
	}
}

// LoadKey reads the key from the first backend that has it.
func LoadKey(apiURL string) (key, source string) {
	acct := keyAccount(apiURL)
	if _, err := lookPath("secret-tool"); err == nil {
		if out, err := runSecretTool([]string{"lookup", "service", "kaneo", "account", acct}, ""); err == nil {
			if k := strings.TrimSpace(out); k != "" {
				return k, BackendKeyring
			}
		}
	}
	if _, err := lookPath("pass"); err == nil {
		if out, err := runPass([]string{"show", "kaneo/" + acct}, ""); err == nil {
			if k := strings.TrimSpace(out); k != "" {
				return k, BackendPass
			}
		}
	}
	return "", BackendNone
}
