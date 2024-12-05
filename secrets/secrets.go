package secrets

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

func FetchSecrets(ctx context.Context, secretsConfig map[string]string) (map[string]string, error) {
	secrets := make(map[string]string)
	client := &http.Client{}
	vaultURL := "https://vault.doubleverify.io/v1/%s"
	token := os.Getenv("VAULT_TOKEN")

	if token == "" {
		return nil, fmt.Errorf("VAULT_TOKEN environment variable is not set")
	}

	for name, path := range secretsConfig {
		req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf(vaultURL, path), nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request for secret %s: %w", name, err)
		}
		req.Header.Add("X-Vault-Token", token)

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch secret %s: %w", name, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := ioutil.ReadAll(resp.Body)
			return nil, fmt.Errorf("failed to fetch secret %s: %s - %s", name, resp.Status, string(body))
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response body for secret %s: %w", name, err)
		}

		var secretData struct {
			Data struct {
				Data map[string]string `json:"data"`
			} `json:"data"`
		}
		err = json.Unmarshal(body, &secretData)
		if err != nil {
			return nil, fmt.Errorf("failed to parse response for secret %s: %w", name, err)
		}

		secretValue, ok := secretData.Data.Data[name]
		if !ok {
			return nil, fmt.Errorf("secret %s not found in response", name)
		}
		secrets[name] = fmt.Sprintf("%s", secretValue)
	}

	return secrets, nil
}
