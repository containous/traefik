package replacequeryregex

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/containous/traefik/v2/pkg/config/dynamic"
	"github.com/containous/traefik/v2/pkg/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReplaceQueryRegex(t *testing.T) {
	testCases := []struct {
		desc          string
		target        string
		regex         string
		replacement   string
		expectedQuery string
	}{
		{
			desc:          "no query to match",
			target:        "/foo",
			regex:         `(.*)`,
			replacement:   "bar=baz",
			expectedQuery: "",
		},
		{
			desc:          "remove query parameter",
			target:        "/foo?remove=yes",
			regex:         `.*(.*)`, // greedy leaves nothing
			replacement:   "$1",
			expectedQuery: "",
		},
		{
			desc:          "overwrite query parameters",
			target:        "/foo?dropped=yes",
			regex:         `.*`,
			replacement:   "bar=baz",
			expectedQuery: "bar=baz",
		},
		{
			desc:          "append query parameter",
			target:        "/foo?keep=yes",
			regex:         `(.*)`,
			replacement:   "$1&bar=baz",
			expectedQuery: "keep=yes&bar=baz",
		},
		{
			desc:          "modify query parameter",
			target:        "/foo?a=a",
			regex:         `a=a`,
			replacement:   "a=A",
			expectedQuery: "a=A",
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			var actualQuery string
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				splitURI := strings.SplitN(r.RequestURI, "?", 2)

				if len(splitURI) == 2 {
					actualQuery = splitURI[1]
				}
			})

			config := dynamic.ReplaceQueryRegex{
				Regex:       test.regex,
				Replacement: test.replacement,
			}

			handler, err := New(context.Background(), next, config, "foo-replace-query-regexp")
			require.NoError(t, err)

			req := testhelpers.MustNewRequest(http.MethodGet, "http://localhost"+test.target, nil)
			req.RequestURI = test.target

			handler.ServeHTTP(nil, req)

			assert.Equal(t, test.expectedQuery, actualQuery, "Unexpected query, wanted '%s', got '%s'.", test.expectedQuery, actualQuery)
		})
	}
}

func TestReplaceQueryRegexError(t *testing.T) {
	testCases := []struct {
		desc          string
		target        string
		regex         string
		replacement   string
		expectedQuery string
	}{
		{
			desc:          "bad regex",
			target:        "/foo",
			regex:         `(?!`,
			replacement:   "",
			expectedQuery: "",
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

			config := dynamic.ReplaceQueryRegex{
				Regex:       test.regex,
				Replacement: test.replacement,
			}

			_, err := New(context.Background(), next, config, "foo-replace-query-regexp")
			require.Error(t, err)
		})
	}
}