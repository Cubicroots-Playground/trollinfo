package angelapi

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
)

// Config holds the configuration for an angel service.
type Config struct {
	BaseURL string
	APIKey  string
}

// ParseFromEnvironment parses the config from the environment.
func (c *Config) ParseFromEnvironment() {
	baseURL := os.Getenv("TROLLINFO_API_BASE_URL")
	apiKey := os.Getenv("TROLLINFO_API_KEY")
	c.BaseURL = baseURL
	c.APIKey = apiKey
}

type service struct {
	config *Config
}

// New assembles a new angel service.
func New(config *Config) Service {
	return &service{
		config: config,
	}
}

func (service *service) makeRequest(method string, urlPath string, body io.Reader, parseResponseTo interface{}) error {
	url := service.config.BaseURL + urlPath
	slog.Info("making request", "method", method, "url", url)

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return err
	}

	req.Header.Set("x-api-key", service.config.APIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 ||
		resp.StatusCode > 299 {
		rawBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(rawBody))
	}

	err = json.NewDecoder(resp.Body).Decode(parseResponseTo)
	if err != nil {
		return err
	}

	return nil
}
