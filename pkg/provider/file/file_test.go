package file

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	"github.com/traefik/traefik/v2/pkg/safe"
)

type ProvideTestCase struct {
	desc                  string
	directoryPaths        []string
	filePath              string
	expectedNumRouter     int
	expectedNumService    int
	expectedNumTLSConf    int
	expectedNumTLSOptions int
}

func TestTLSCertificateContent(t *testing.T) {
	tempDir := t.TempDir()

	fileTLS, err := createTempFile("./fixtures/toml/tls_file.cert", tempDir)
	require.NoError(t, err)

	fileTLSKey, err := createTempFile("./fixtures/toml/tls_file_key.cert", tempDir)
	require.NoError(t, err)

	fileConfig, err := os.CreateTemp(tempDir, "temp*.toml")
	require.NoError(t, err)

	content := `
[[tls.certificates]]
  certFile = "` + fileTLS.Name() + `"
  keyFile = "` + fileTLSKey.Name() + `"

[tls.options.default.clientAuth]
  caFiles = ["` + fileTLS.Name() + `"]

[tls.stores.default.defaultCertificate]
  certFile = "` + fileTLS.Name() + `"
  keyFile = "` + fileTLSKey.Name() + `"

[http.serversTransports.default]
  rootCAs = ["` + fileTLS.Name() + `"]
  [[http.serversTransports.default.certificates]]
    certFile = "` + fileTLS.Name() + `"
    keyFile = "` + fileTLSKey.Name() + `"
`

	_, err = fileConfig.Write([]byte(content))
	require.NoError(t, err)

	provider := &Provider{}
	configuration, err := provider.loadFileConfig(context.Background(), fileConfig.Name(), true)
	require.NoError(t, err)

	require.Equal(t, "CONTENT", configuration.TLS.Certificates[0].Certificate.CertFile.String())
	require.Equal(t, "CONTENTKEY", configuration.TLS.Certificates[0].Certificate.KeyFile.String())

	require.Equal(t, "CONTENT", configuration.TLS.Options["default"].ClientAuth.CAFiles[0].String())

	require.Equal(t, "CONTENT", configuration.TLS.Stores["default"].DefaultCertificate.CertFile.String())
	require.Equal(t, "CONTENTKEY", configuration.TLS.Stores["default"].DefaultCertificate.KeyFile.String())

	require.Equal(t, "CONTENT", configuration.HTTP.ServersTransports["default"].Certificates[0].CertFile.String())
	require.Equal(t, "CONTENTKEY", configuration.HTTP.ServersTransports["default"].Certificates[0].KeyFile.String())
	require.Equal(t, "CONTENT", configuration.HTTP.ServersTransports["default"].RootCAs[0].String())
}

func TestErrorWhenEmptyConfig(t *testing.T) {
	provider := &Provider{}
	configChan := make(chan dynamic.Message)
	errorChan := make(chan struct{})
	go func() {
		err := provider.Provide(configChan, safe.NewPool(context.Background()))
		assert.Error(t, err)
		close(errorChan)
	}()

	timeout := time.After(time.Second)
	select {
	case <-configChan:
		t.Fatal("We should not receive config message")
	case <-timeout:
		t.Fatal("timeout while waiting for config")
	case <-errorChan:
	}
}

func TestProvideWithoutWatch(t *testing.T) {
	for _, test := range getTestCases() {
		t.Run(test.desc+" without watch", func(t *testing.T) {
			provider := createProvider(t, test, false)
			configChan := make(chan dynamic.Message)

			provider.DebugLogGeneratedTemplate = true

			go func() {
				err := provider.Provide(configChan, safe.NewPool(context.Background()))
				assert.NoError(t, err)
			}()

			timeout := time.After(time.Second)
			select {
			case conf := <-configChan:

				require.NotNil(t, conf.Configuration.HTTP)
				numServices := len(conf.Configuration.HTTP.Services) + len(conf.Configuration.TCP.Services) + len(conf.Configuration.UDP.Services)
				numRouters := len(conf.Configuration.HTTP.Routers) + len(conf.Configuration.TCP.Routers) + len(conf.Configuration.UDP.Routers)
				assert.Equal(t, test.expectedNumService, numServices)
				assert.Equal(t, test.expectedNumRouter, numRouters)
				require.NotNil(t, conf.Configuration.TLS)
				assert.Len(t, conf.Configuration.TLS.Certificates, test.expectedNumTLSConf)
				assert.Len(t, conf.Configuration.TLS.Options, test.expectedNumTLSOptions)
			case <-timeout:
				t.Errorf("timeout while waiting for config")
			}
		})
	}
}

func TestProvideWithWatch(t *testing.T) {
	for _, test := range getTestCases() {
		t.Run(test.desc+" with watch", func(t *testing.T) {
			provider := createProvider(t, test, true)
			configChan := make(chan dynamic.Message)

			go func() {
				err := provider.Provide(configChan, safe.NewPool(context.Background()))
				assert.NoError(t, err)
			}()

			timeout := time.After(time.Second)
			select {
			case conf := <-configChan:
				require.NotNil(t, conf.Configuration.HTTP)
				numServices := len(conf.Configuration.HTTP.Services) + len(conf.Configuration.TCP.Services) + len(conf.Configuration.UDP.Services)
				numRouters := len(conf.Configuration.HTTP.Routers) + len(conf.Configuration.TCP.Routers) + len(conf.Configuration.UDP.Routers)
				assert.Equal(t, numServices, 0)
				assert.Equal(t, numRouters, 0)
				require.NotNil(t, conf.Configuration.TLS)
				assert.Len(t, conf.Configuration.TLS.Certificates, 0)
			case <-timeout:
				t.Errorf("timeout while waiting for config")
			}

			if len(test.filePath) > 0 {
				err := copyFile(test.filePath, provider.Filename)
				require.NoError(t, err)
			}

			if len(test.directoryPaths) > 0 {
				for i, filePath := range test.directoryPaths {
					err := copyFile(filePath, filepath.Join(provider.Directory, strconv.Itoa(i)+filepath.Ext(filePath)))
					require.NoError(t, err)
				}
			}

			timeout = time.After(1 * time.Second)
			var numUpdates, numServices, numRouters, numTLSConfs int
			for {
				select {
				case conf := <-configChan:
					numUpdates++
					numServices = len(conf.Configuration.HTTP.Services) + len(conf.Configuration.TCP.Services) + len(conf.Configuration.UDP.Services)
					numRouters = len(conf.Configuration.HTTP.Routers) + len(conf.Configuration.TCP.Routers) + len(conf.Configuration.UDP.Routers)
					numTLSConfs = len(conf.Configuration.TLS.Certificates)
					t.Logf("received update #%d: services %d/%d, routers %d/%d, TLS configs %d/%d", numUpdates, numServices, test.expectedNumService, numRouters, test.expectedNumRouter, numTLSConfs, test.expectedNumTLSConf)

					if numServices == test.expectedNumService && numRouters == test.expectedNumRouter && numTLSConfs == test.expectedNumTLSConf {
						return
					}
				case <-timeout:
					t.Fatal("timeout while waiting for config")
				}
			}
		})
	}
}

func getTestCases() []ProvideTestCase {
	return []ProvideTestCase{
		{
			desc:               "simple file",
			filePath:           "./fixtures/toml/simple_file_01.toml",
			expectedNumRouter:  3,
			expectedNumService: 6,
			expectedNumTLSConf: 5,
		},
		{
			desc:               "simple file with tcp and udp",
			filePath:           "./fixtures/toml/simple_file_02.toml",
			expectedNumRouter:  5,
			expectedNumService: 8,
			expectedNumTLSConf: 5,
		},
		{
			desc:               "simple file yaml",
			filePath:           "./fixtures/yaml/simple_file_01.yml",
			expectedNumRouter:  3,
			expectedNumService: 6,
			expectedNumTLSConf: 5,
		},
		{
			desc:              "template file",
			filePath:          "./fixtures/toml/template_file.toml",
			expectedNumRouter: 20,
		},
		{
			desc:              "template file yaml",
			filePath:          "./fixtures/yaml/template_file.yml",
			expectedNumRouter: 20,
		},
		{
			desc: "simple directory",
			directoryPaths: []string{
				"./fixtures/toml/dir01_file01.toml",
				"./fixtures/toml/dir01_file02.toml",
				"./fixtures/toml/dir01_file03.toml",
			},
			expectedNumRouter:     2,
			expectedNumService:    3,
			expectedNumTLSConf:    4,
			expectedNumTLSOptions: 1,
		},
		{
			desc: "simple directory yaml",
			directoryPaths: []string{
				"./fixtures/yaml/dir01_file01.yml",
				"./fixtures/yaml/dir01_file02.yml",
				"./fixtures/yaml/dir01_file03.yml",
			},
			expectedNumRouter:     2,
			expectedNumService:    3,
			expectedNumTLSConf:    4,
			expectedNumTLSOptions: 1,
		},
		{
			desc: "template in directory",
			directoryPaths: []string{
				"./fixtures/toml/template_in_directory_file01.toml",
				"./fixtures/toml/template_in_directory_file02.toml",
			},
			expectedNumRouter:  20,
			expectedNumService: 20,
		},
		{
			desc: "template in directory yaml",
			directoryPaths: []string{
				"./fixtures/yaml/template_in_directory_file01.yml",
				"./fixtures/yaml/template_in_directory_file02.yml",
			},
			expectedNumRouter:  20,
			expectedNumService: 20,
		},
		{
			desc:               "simple file with empty store yaml",
			filePath:           "./fixtures/yaml/simple_empty_store.yml",
			expectedNumRouter:  0,
			expectedNumService: 0,
			expectedNumTLSConf: 0,
		},
	}
}

func createProvider(t *testing.T, test ProvideTestCase, watch bool) *Provider {
	t.Helper()

	tempDir := t.TempDir()

	provider := &Provider{}
	provider.Watch = true

	if len(test.directoryPaths) > 0 {
		if !watch {
			for _, filePath := range test.directoryPaths {
				var err error
				_, err = createTempFile(filePath, tempDir)
				require.NoError(t, err)
			}
		}
		provider.Directory = tempDir
	}

	if len(test.filePath) > 0 {
		var file *os.File
		if watch {
			var err error
			file, err = os.CreateTemp(tempDir, "temp*"+filepath.Ext(test.filePath))
			require.NoError(t, err)
		} else {
			var err error
			file, err = createTempFile(test.filePath, tempDir)
			require.NoError(t, err)
		}

		provider.Filename = file.Name()
	}

	t.Cleanup(func() {
		os.RemoveAll(tempDir)
	})

	return provider
}

func copyFile(srcPath, dstPath string) error {
	dst, err := os.OpenFile(dstPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o666)
	if err != nil {
		return err
	}
	defer dst.Close()

	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	_, err = io.Copy(dst, src)
	return err
}

func createTempFile(srcPath, tempDir string) (*os.File, error) {
	file, err := os.CreateTemp(tempDir, "temp*"+filepath.Ext(srcPath))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	src, err := os.Open(srcPath)
	if err != nil {
		return nil, err
	}
	defer src.Close()

	_, err = io.Copy(file, src)
	return file, err
}
