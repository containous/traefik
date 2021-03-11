package try

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"

	"github.com/abronan/valkeyrie/store"
)

// ResponseCondition is a retry condition function.
// It receives a response, and returns an error if the response failed the condition.
type ResponseCondition func(*http.Response) error

// BodyContains returns a retry condition function.
// The condition returns an error if the request body does not contain all the given strings.
func BodyContains(values ...string) ResponseCondition {
	return func(res *http.Response) error {
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body: %w", err)
		}

		for _, value := range values {
			if !strings.Contains(string(body), value) {
				return fmt.Errorf("could not find '%s' in body '%s'", value, string(body))
			}
		}
		return nil
	}
}

// BodyNotContains returns a retry condition function.
// The condition returns an error if the request body  contain one of the given strings.
func BodyNotContains(values ...string) ResponseCondition {
	return func(res *http.Response) error {
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body: %w", err)
		}

		for _, value := range values {
			if strings.Contains(string(body), value) {
				return fmt.Errorf("find '%s' in body '%s'", value, string(body))
			}
		}
		return nil
	}
}

// BodyContainsOr returns a retry condition function.
// The condition returns an error if the request body does not contain one of the given strings.
func BodyContainsOr(values ...string) ResponseCondition {
	return func(res *http.Response) error {
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body: %w", err)
		}

		for _, value := range values {
			if strings.Contains(string(body), value) {
				return nil
			}
		}
		return fmt.Errorf("could not find '%v' in body '%s'", values, string(body))
	}
}

// HasBody returns a retry condition function.
// The condition returns an error if the request body does not have body content.
func HasBody() ResponseCondition {
	return func(res *http.Response) error {
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body: %w", err)
		}

		if len(body) == 0 {
			return errors.New("response doesn't have body content")
		}
		return nil
	}
}

// HasCn returns a retry condition function.
// The condition returns an error if the cn is not correct.
func HasCn(cn string) ResponseCondition {
	return func(res *http.Response) error {
		if res.TLS == nil {
			return errors.New("response doesn't have TLS")
		}

		if len(res.TLS.PeerCertificates) == 0 {
			return errors.New("response TLS doesn't have peer certificates")
		}

		if res.TLS.PeerCertificates[0] == nil {
			return errors.New("first peer certificate is nil")
		}

		commonName := res.TLS.PeerCertificates[0].Subject.CommonName
		if cn != commonName {
			return fmt.Errorf("common name don't match: %s != %s", cn, commonName)
		}

		return nil
	}
}

// StatusCodeIs returns a retry condition function.
// The condition returns an error if the given response's status code is not the
// given HTTP status code.
func StatusCodeIs(status int) ResponseCondition {
	return func(res *http.Response) error {
		if res.StatusCode != status {
			return fmt.Errorf("got status code %d, wanted %d", res.StatusCode, status)
		}
		return nil
	}
}

// HasHeader returns a retry condition function.
// The condition returns an error if the response does not have a header set.
func HasHeader(header string) ResponseCondition {
	return func(res *http.Response) error {
		if _, ok := res.Header[header]; !ok {
			return errors.New("response doesn't contain header: " + header)
		}
		return nil
	}
}

// HasHeaderValue returns a retry condition function.
// The condition returns an error if the response does not have a header set, and a value for that header.
// Has an option to test for an exact header match only, not just contains.
func HasHeaderValue(header, value string, exactMatch bool) ResponseCondition {
	return func(res *http.Response) error {
		if _, ok := res.Header[header]; !ok {
			return errors.New("response doesn't contain header: " + header)
		}

		matchFound := false
		for _, hdr := range res.Header[header] {
			if value != hdr && exactMatch {
				return fmt.Errorf("got header %s with value %s, wanted %s", header, hdr, value)
			}
			if value == hdr {
				matchFound = true
			}
		}

		if !matchFound {
			return fmt.Errorf("response doesn't contain header %s with value %s", header, value)
		}
		return nil
	}
}

// HasHeaderStruct returns a retry condition function.
// The condition returns an error if the response does contain the headers set, and matching contents.
func HasHeaderStruct(header http.Header) ResponseCondition {
	return func(res *http.Response) error {
		for key := range header {
			if _, ok := res.Header[key]; !ok {
				return fmt.Errorf("header %s not present in the response. Expected headers: %v Got response headers: %v", key, header, res.Header)
			}

			// Header exists in the response, test it.
			if !reflect.DeepEqual(header[key], res.Header[key]) {
				return fmt.Errorf("for header %s got values %v, wanted %v", key, res.Header[key], header[key])
			}
		}
		return nil
	}
}

// DoCondition is a retry condition function.
// It returns an error.
type DoCondition func() error

// KVExists is a retry condition function.
// Verify if a Key exists in the store.
func KVExists(kv store.Store, key string) DoCondition {
	return func() error {
		_, err := kv.Exists(key, nil)
		return err
	}
}
