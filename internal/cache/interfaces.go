package cache

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/likexian/whois"
)

// HTTPClient interface for making HTTP requests.
type HTTPClient interface {
	Get(ctx context.Context, url string) (*http.Response, error)
	Do(req *http.Request) (*http.Response, error)
}

// WhoisClient interface for whois lookups.
type WhoisClient interface {
	Lookup(ip string) (string, error)
}

// Logger interface for logging.
type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

// FileSystem interface for file operations.
type FileSystem interface {
	MkdirAll(path string, perm uint32) error
	OpenFile(name string, flag int, perm uint32) (File, error)
}

// File interface for file operations.
type File interface {
	io.ReadWriteCloser
}

// Default implementations

// defaultHTTPClient implements HTTPClient using the standard http.Client.
type defaultHTTPClient struct {
	client *http.Client
}

// NewDefaultHTTPClient creates a new default HTTP client with the specified timeout.
func NewDefaultHTTPClient(timeout time.Duration) HTTPClient {
	return &defaultHTTPClient{
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *defaultHTTPClient) Get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute HTTP request: %w", err)
	}
	return resp, nil
}

func (c *defaultHTTPClient) Do(req *http.Request) (*http.Response, error) {
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute HTTP request: %w", err)
	}
	return resp, nil
}

// defaultWhoisClient implements WhoisClient using the likexian/whois package.
type defaultWhoisClient struct {
	timeout time.Duration
}

// NewDefaultWhoisClient creates a new default whois client with the specified timeout.
func NewDefaultWhoisClient(timeout time.Duration) WhoisClient {
	return &defaultWhoisClient{
		timeout: timeout,
	}
}

func (c *defaultWhoisClient) Lookup(ip string) (string, error) {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	// Use a channel to handle the timeout
	resultCh := make(chan string, 1)
	errCh := make(chan error, 1)

	go func() {
		result, err := whois.Whois(ip)
		if err != nil {
			errCh <- fmt.Errorf("whois lookup failed: %w", err)
			return
		}
		resultCh <- result
	}()

	// Wait for result or timeout
	select {
	case result := <-resultCh:
		return result, nil
	case err := <-errCh:
		return "", err
	case <-ctx.Done():
		return "", fmt.Errorf("whois lookup timed out after %v", c.timeout)
	}
}

// defaultLogger implements Logger using the standard log package.
type defaultLogger struct {
	enabled bool
}

// NewDefaultLogger creates a new default logger with the specified enabled state.
func NewDefaultLogger(enabled bool) Logger {
	return &defaultLogger{
		enabled: enabled,
	}
}

func (l *defaultLogger) Debug(msg string, args ...interface{}) {
	if l.enabled {
		log.Printf("[DEBUG] "+msg, args...)
	}
}

func (l *defaultLogger) Info(msg string, args ...interface{}) {
	if l.enabled {
		log.Printf("[INFO] "+msg, args...)
	}
}

func (l *defaultLogger) Warn(msg string, args ...interface{}) {
	if l.enabled {
		log.Printf("[WARN] "+msg, args...)
	}
}

func (l *defaultLogger) Error(msg string, args ...interface{}) {
	if l.enabled {
		log.Printf("[ERROR] "+msg, args...)
	}
}

// defaultFileSystem implements FileSystem using the standard os package.
type defaultFileSystem struct{}

// NewDefaultFileSystem creates a new default file system implementation.
func NewDefaultFileSystem() FileSystem {
	return &defaultFileSystem{}
}

func (fs *defaultFileSystem) MkdirAll(path string, perm uint32) error {
	err := os.MkdirAll(path, os.FileMode(perm))
	if err != nil {
		return fmt.Errorf("failed to create directory %s: %w", path, err)
	}
	return nil
}

func (fs *defaultFileSystem) OpenFile(name string, flag int, perm uint32) (File, error) {
	// Validate the path to prevent potential file inclusion via variable
	if !filepath.IsAbs(name) {
		// For relative paths, ensure they don't contain path traversal
		cleanPath := filepath.Clean(name)
		if cleanPath != name {
			return nil, os.ErrInvalid
		}
	}
	file, err := os.OpenFile(name, flag, os.FileMode(perm))
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", name, err)
	}
	return file, nil
}
