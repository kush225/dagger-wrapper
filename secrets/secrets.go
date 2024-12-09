package secrets

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

func SplitPathKey(input string) (string, string, error) {
	lastSlashIndex := strings.LastIndex(input, "/")
	if lastSlashIndex == -1 {
		return "", "", fmt.Errorf("invalid input string, no '/' found")
	}

	path := input[:lastSlashIndex]
	key := input[lastSlashIndex+1:]

	return path, key, nil
}

func FetchSecrets(ctx context.Context, secretsConfig map[string]string) (map[string]string, error) {
	secrets := make(map[string]string)
	client := &http.Client{}
	vaultURL := "https://vault.doubleverify.io/v1/%s"
	token := os.Getenv("VAULT_TOKEN")

	if token == "" {
		return nil, fmt.Errorf("VAULT_TOKEN environment variable is not set")
	}
	secrets["VAULT_TOKEN"] = token

	for name, path := range secretsConfig {

		key_path, key, err := SplitPathKey(path)
		if err != nil {
			return nil, err
		}

		req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf(vaultURL, key_path), nil)
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

		secretValue, ok := secretData.Data.Data[key]
		if !ok {
			return nil, fmt.Errorf("secret %s not found in response", key)
		}
		secrets[name] = fmt.Sprintf("%s", secretValue)
	}
	

	return secrets, nil
}
