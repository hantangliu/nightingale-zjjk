package pconf

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// EnsureOpenGeminiDatabases creates databases referenced by OpenGemini remote
// write URLs. Other Prometheus-compatible writers are left untouched.
func EnsureOpenGeminiDatabases(pushgw Pushgw) error {
	for _, writer := range pushgw.Writers {
		database, endpoint, ok, err := openGeminiDatabase(writer.Url)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}

		if err := createOpenGeminiDatabase(endpoint, database, writer); err != nil {
			return err
		}
	}
	return nil
}

func openGeminiDatabase(rawURL string) (string, string, bool, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return "", "", false, fmt.Errorf("invalid writer URL %q: %w", rawURL, err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", "", false, fmt.Errorf("writer URL %q must include scheme and host", rawURL)
	}
	if parsed.Path != "/api/v1/write" {
		return "", "", false, nil
	}

	database := parsed.Query().Get("db")
	if database == "" {
		return "", "", false, fmt.Errorf("OpenGemini writer URL %q must include the db query parameter", rawURL)
	}
	if strings.ContainsAny(database, "\"'\r\n") {
		return "", "", false, fmt.Errorf("invalid OpenGemini database name %q", database)
	}

	parsed.Path = "/query"
	query := parsed.Query()
	query.Del("q")
	query.Set("db", database)
	parsed.RawQuery = query.Encode()
	return database, parsed.String(), true, nil
}

func createOpenGeminiDatabase(endpoint, database string, writer WriterOptions) error {
	queryURL, err := url.Parse(endpoint)
	if err != nil {
		return fmt.Errorf("invalid OpenGemini query URL %q: %w", endpoint, err)
	}
	tlsConfig, err := writer.ClientConfig.TLSConfig()
	if err != nil {
		return fmt.Errorf("create OpenGemini TLS config for %q: %w", database, err)
	}
	client := &http.Client{
		Timeout:   time.Duration(writer.Timeout) * time.Millisecond,
		Transport: &http.Transport{TLSClientConfig: tlsConfig},
	}
	form := url.Values{"q": []string{fmt.Sprintf(`CREATE DATABASE "%s"`, database)}}
	request, err := http.NewRequest(http.MethodPost, queryURL.String(), strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("create OpenGemini database %q: %w", database, err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if writer.BasicAuthUser != "" {
		request.SetBasicAuth(writer.BasicAuthUser, writer.BasicAuthPass)
	}

	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("create OpenGemini database %q: %w", database, err)
	}
	defer response.Body.Close()
	body, readErr := io.ReadAll(io.LimitReader(response.Body, 64*1024))
	if readErr != nil {
		return fmt.Errorf("read OpenGemini database response for %q: %w", database, readErr)
	}
	if response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusMultipleChoices {
		return nil
	}

	// OpenGemini returns an error when the database already exists. That is a
	// successful outcome for startup initialization.
	if strings.Contains(strings.ToLower(string(body)), "already exist") {
		return nil
	}
	return fmt.Errorf("create OpenGemini database %q: status %s, response: %s",
		database, response.Status, strings.TrimSpace(string(body)))
}
