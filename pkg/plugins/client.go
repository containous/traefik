package plugins

import (
	zipa "archive/zip"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/mod/module"
	"golang.org/x/mod/zip"
	"gopkg.in/yaml.v3"
)

const (
	sourcesFolder  = "sources"
	archivesFolder = "archives"
	stateFilename  = "state.json"
	goPathSrc      = "src"
	pluginManifest = ".traefik.yml"
)

const pilotURL = "https://plugin.pilot.traefik.io/public/"

const (
	hashHeader  = "X-Plugin-Hash"
	tokenHeader = "X-Token"
)

// ClientOptions the options of a Traefik Pilot client.
type ClientOptions struct {
	Output string
	Token  string
}

// Client a Traefik Pilot client.
type Client struct {
	HTTPClient *http.Client
	baseURL    *url.URL

	token     string
	archives  string
	stateFile string
	goPath    string
	sources   string
}

// NewClient creates a new Traefik Pilot client.
func NewClient(opts ClientOptions) (*Client, error) {
	baseURL, err := url.Parse(pilotURL)
	if err != nil {
		return nil, err
	}

	sourcesRootPath := filepath.Join(filepath.FromSlash(opts.Output), sourcesFolder)
	err = resetDirectory(sourcesRootPath)
	if err != nil {
		return nil, err
	}

	goPath, err := os.MkdirTemp(sourcesRootPath, "gop-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create GoPath: %w", err)
	}

	archivesPath := filepath.Join(filepath.FromSlash(opts.Output), archivesFolder)
	err = os.MkdirAll(archivesPath, 0o755)
	if err != nil {
		return nil, fmt.Errorf("failed to create archives directory %s: %w", archivesPath, err)
	}

	return &Client{
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
		baseURL:    baseURL,

		archives:  archivesPath,
		stateFile: filepath.Join(archivesPath, stateFilename),

		goPath:  goPath,
		sources: filepath.Join(goPath, goPathSrc),

		token: opts.Token,
	}, nil
}

// GoPath gets the plugins GoPath.
func (c *Client) GoPath() string {
	return c.goPath
}

// ReadManifest reads a plugin manifest.
func (c *Client) ReadManifest(moduleName string) (*Manifest, error) {
	return ReadManifest(c.goPath, moduleName)
}

// ReadManifest reads a plugin manifest.
func ReadManifest(goPath, moduleName string) (*Manifest, error) {
	p := filepath.Join(goPath, goPathSrc, filepath.FromSlash(moduleName), pluginManifest)

	file, err := os.Open(p)
	if err != nil {
		return nil, fmt.Errorf("failed to open the plugin manifest %s: %w", p, err)
	}

	defer func() { _ = file.Close() }()

	m := &Manifest{}
	err = yaml.NewDecoder(file).Decode(m)
	if err != nil {
		return nil, fmt.Errorf("failed to decode the plugin manifest %s: %w", p, err)
	}

	return m, nil
}

// Download downloads a plugin archive.
func (c *Client) Download(ctx context.Context, pName, pVersion string) (string, error) {
	filename := c.buildArchivePath(pName, pVersion)

	var hash string
	_, err := os.Stat(filename)
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("failed to read archive %s: %w", filename, err)
	}

	if err == nil {
		hash, err = computeHash(filename)
		if err != nil {
			return "", fmt.Errorf("failed to compute hash: %w", err)
		}
	}

	endpoint, err := c.baseURL.Parse(path.Join(c.baseURL.Path, "download", pName, pVersion))
	if err != nil {
		return "", fmt.Errorf("failed to parse endpoint URL: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	if hash != "" {
		req.Header.Set(hashHeader, hash)
	}

	if c.token != "" {
		req.Header.Set(tokenHeader, c.token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call service: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusOK {
		err = os.MkdirAll(filepath.Dir(filename), 0o755)
		if err != nil {
			return "", fmt.Errorf("failed to create directory: %w", err)
		}

		var file *os.File
		file, err = os.Create(filename)
		if err != nil {
			return "", fmt.Errorf("failed to create file %q: %w", filename, err)
		}

		defer func() { _ = file.Close() }()

		_, err = io.Copy(file, resp.Body)
		if err != nil {
			return "", fmt.Errorf("failed to write response: %w", err)
		}

		hash, err = computeHash(filename)
		if err != nil {
			return "", fmt.Errorf("failed to compute hash: %w", err)
		}

		return hash, nil
	}

	if resp.StatusCode == http.StatusNotModified {
		// noop
		return hash, nil
	}

	data, _ := io.ReadAll(resp.Body)
	return "", fmt.Errorf("error: %d: %s", resp.StatusCode, string(data))
}

// Check checks the plugin archive integrity.
func (c *Client) Check(ctx context.Context, pName, pVersion, hash string) error {
	endpoint, err := c.baseURL.Parse(path.Join(c.baseURL.Path, "validate", pName, pVersion))
	if err != nil {
		return fmt.Errorf("failed to parse endpoint URL: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if hash != "" {
		req.Header.Set(hashHeader, hash)
	}

	if c.token != "" {
		req.Header.Set(tokenHeader, c.token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call service: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusOK {
		return nil
	}

	return fmt.Errorf("plugin integrity check failed")
}

// Unzip unzip a plugin archive.
func (c *Client) Unzip(pName, pVersion string) error {
	err := c.unzipModule(pName, pVersion)
	if err == nil {
		return nil
	}

	return c.unzipArchive(pName, pVersion)
}

func (c *Client) unzipModule(pName, pVersion string) error {
	src := c.buildArchivePath(pName, pVersion)
	dest := filepath.Join(c.sources, filepath.FromSlash(pName))

	return zip.Unzip(dest, module.Version{Path: pName, Version: pVersion}, src)
}

func (c *Client) unzipArchive(pName, pVersion string) error {
	zipPath := c.buildArchivePath(pName, pVersion)

	archive, err := zipa.OpenReader(zipPath)
	if err != nil {
		return err
	}

	defer func() { _ = archive.Close() }()

	dest := filepath.Join(c.sources, filepath.FromSlash(pName))

	for _, f := range archive.File {
		err = unzipFile(f, dest)
		if err != nil {
			return err
		}
	}

	return nil
}

func unzipFile(f *zipa.File, dest string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}

	defer func() { _ = rc.Close() }()

	pathParts := strings.SplitN(f.Name, "/", 2)
	p := filepath.Join(dest, pathParts[1])

	if f.FileInfo().IsDir() {
		return os.MkdirAll(p, f.Mode())
	}

	err = os.MkdirAll(filepath.Dir(p), 0o750)
	if err != nil {
		return err
	}

	elt, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}

	defer func() { _ = elt.Close() }()

	_, err = io.Copy(elt, rc)
	if err != nil {
		return err
	}

	return nil
}

// CleanArchives cleans plugins archives.
func (c *Client) CleanArchives(plugins map[string]Descriptor) error {
	if _, err := os.Stat(c.stateFile); os.IsNotExist(err) {
		return nil
	}

	stateFile, err := os.Open(c.stateFile)
	if err != nil {
		return fmt.Errorf("failed to open state file %s: %w", c.stateFile, err)
	}

	previous := make(map[string]string)
	err = json.NewDecoder(stateFile).Decode(&previous)
	if err != nil {
		return fmt.Errorf("failed to decode state file %s: %w", c.stateFile, err)
	}

	for pName, pVersion := range previous {
		for _, desc := range plugins {
			if desc.ModuleName == pName && desc.Version != pVersion {
				archivePath := c.buildArchivePath(pName, pVersion)
				if err = os.RemoveAll(archivePath); err != nil {
					return fmt.Errorf("failed to remove archive %s: %w", archivePath, err)
				}
			}
		}
	}

	return nil
}

// WriteState writes the plugins state files.
func (c *Client) WriteState(plugins map[string]Descriptor) error {
	m := make(map[string]string)

	for _, descriptor := range plugins {
		m[descriptor.ModuleName] = descriptor.Version
	}

	mp, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(c.stateFile, mp, 0o600)
}

// ResetAll resets all plugins related directories.
func (c *Client) ResetAll() error {
	if c.goPath == "" {
		return errors.New("goPath is empty")
	}

	err := resetDirectory(filepath.Join(c.goPath, ".."))
	if err != nil {
		return err
	}

	return resetDirectory(c.archives)
}

func (c *Client) buildArchivePath(pName, pVersion string) string {
	return filepath.Join(c.archives, filepath.FromSlash(pName), pVersion+".zip")
}

func resetDirectory(dir string) error {
	dirPath, err := filepath.Abs(dir)
	if err != nil {
		return err
	}

	currentPath, err := os.Getwd()
	if err != nil {
		return err
	}

	if strings.HasPrefix(currentPath, dirPath) {
		return fmt.Errorf("cannot be deleted: the directory path %s is the parent of the current path %s", dirPath, currentPath)
	}

	err = os.RemoveAll(dir)
	if err != nil {
		return err
	}

	return os.MkdirAll(dir, 0o755)
}

func computeHash(filepath string) (string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return "", err
	}

	hash := sha256.New()

	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	sum := hash.Sum(nil)

	return fmt.Sprintf("%x", sum), nil
}
