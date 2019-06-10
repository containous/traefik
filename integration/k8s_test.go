package integration

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/containous/traefik/integration/try"
	"github.com/containous/traefik/pkg/api"
	"github.com/go-check/check"
	"github.com/pmezard/go-difflib/difflib"
	checker "github.com/vdemeester/shakers"
)

var updateExpected = flag.Bool("update_expected", false, "Update expected files in testdata")

// K8sSuite
type K8sSuite struct{ BaseSuite }

func (s *K8sSuite) SetUpSuite(c *check.C) {
	s.createComposeProject(c, "k8s")
	s.composeProject.Start(c)

	abs, err := filepath.Abs("./fixtures/k8s/kubeconfig.yaml")
	c.Assert(err, checker.IsNil)

	err = try.Do(60*time.Second, try.DoCondition(func() error {
		_, err := os.Stat(abs)
		return err
	}))
	c.Assert(err, checker.IsNil)

	err = os.Setenv("KUBECONFIG", abs)
	c.Assert(err, checker.IsNil)
}

func (s *K8sSuite) TearDownSuite(c *check.C) {
	s.composeProject.Stop(c)

	err := os.Remove("./fixtures/k8s/kubeconfig.yaml")
	if err != nil {
		c.Log(err)
	}
	err = os.Remove("./fixtures/k8s/coredns.yaml")
	if err != nil {
		c.Log(err)
	}
	err = os.Remove("./fixtures/k8s/traefik.yaml")
	if err != nil {
		c.Log(err)
	}
}

func (s *K8sSuite) TestIngressConfiguration(c *check.C) {
	cmd, display := s.traefikCmd(withConfigFile("fixtures/k8s_default.toml"))
	defer display(c)

	err := cmd.Start()
	c.Assert(err, checker.IsNil)
	defer cmd.Process.Kill()

	testConfiguration(c, "testdata/rawdata-ingress.json")
}

func (s *K8sSuite) TestCRDConfiguration(c *check.C) {
	cmd, display := s.traefikCmd(withConfigFile("fixtures/k8s_crd.toml"))
	defer display(c)

	err := cmd.Start()
	c.Assert(err, checker.IsNil)
	defer cmd.Process.Kill()

	testConfiguration(c, "testdata/rawdata-crd.json")
}

func testConfiguration(c *check.C, path string) {
	expectedJSON := filepath.FromSlash(path)

	if *updateExpected {
		fi, err := os.Create(expectedJSON)
		c.Assert(err, checker.IsNil)
		err = fi.Close()
		c.Assert(err, checker.IsNil)
	}

	var buf bytes.Buffer
	err := try.GetRequest("http://127.0.0.1:8080/api/rawdata", 20*time.Second, try.StatusCodeIs(http.StatusOK), matchesConfig(expectedJSON, &buf))

	if !*updateExpected {
		if err != nil {
			c.Error(err)
		}
		return
	}

	if err != nil {
		c.Logf("In file update mode, got expected error: %v", err)
	}

	var rtRepr api.RunTimeRepresentation
	err = json.Unmarshal(buf.Bytes(), &rtRepr)
	c.Assert(err, checker.IsNil)

	newJSON, err := json.MarshalIndent(rtRepr, "", "\t")
	c.Assert(err, checker.IsNil)

	err = ioutil.WriteFile(expectedJSON, newJSON, 0644)
	c.Assert(err, checker.IsNil)
	c.Errorf("We do not want a passing test in file update mode")
}

func matchesConfig(wantConfig string, buf *bytes.Buffer) try.ResponseCondition {
	return func(res *http.Response) error {
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body: %s", err)
		}

		if err := res.Body.Close(); err != nil {
			return err
		}

		var obtained api.RunTimeRepresentation
		err = json.Unmarshal(body, &obtained)
		if err != nil {
			return err
		}

		if buf != nil {
			buf.Reset()
			if _, err := io.Copy(buf, bytes.NewReader(body)); err != nil {
				return err
			}
		}

		got, err := json.MarshalIndent(obtained, "", "\t")
		if err != nil {
			return err
		}

		expected, err := ioutil.ReadFile(wantConfig)
		if err != nil {
			return err
		}

		// The pods IPs are dynamic, so we cannot predict them,
		// which is why we have to ignore them in the comparison.
		rxURL := regexp.MustCompile(`"(url|address)":\s+(".*")`)
		sanitizedExpected := rxURL.ReplaceAll(expected, []byte(`"$1": "XXXX"`))
		sanitizedGot := rxURL.ReplaceAll(got, []byte(`"$1": "XXXX"`))

		rxServerStatus := regexp.MustCompile(`"http://.*?":\s+(".*")`)
		sanitizedExpected = rxServerStatus.ReplaceAll(sanitizedExpected, []byte(`"http://XXXX": $1`))
		sanitizedGot = rxServerStatus.ReplaceAll(sanitizedGot, []byte(`"http://XXXX": $1`))

		if bytes.Equal(sanitizedExpected, sanitizedGot) {
			return nil
		}

		diff := difflib.UnifiedDiff{
			FromFile: "Expected",
			A:        difflib.SplitLines(string(sanitizedExpected)),
			ToFile:   "Want",
			B:        difflib.SplitLines(string(sanitizedGot)),
			Context:  3,
		}

		text, err := difflib.GetUnifiedDiffString(diff)
		if err != nil {
			return err
		}
		return errors.New(text)
	}
}
