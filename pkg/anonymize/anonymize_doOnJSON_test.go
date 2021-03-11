package anonymize

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_doOnJSON(t *testing.T) {
	baseConfiguration, err := os.ReadFile("./testdata/example.json")
	require.NoError(t, err)

	anomConfiguration := doOnJSON(string(baseConfiguration))

	expectedConfiguration, err := os.ReadFile("./testdata/expected.json")
	require.NoError(t, err)

	assert.JSONEq(t, string(expectedConfiguration), anomConfiguration)
}

func Test_doOnJSON_simple(t *testing.T) {
	testCases := []struct {
		name           string
		input          string
		expectedOutput string
	}{
		{
			name: "email",
			input: `{
				"email1": "goo@example.com",
				"email2": "foo.bargoo@example.com",
				"email3": "foo.bargoo@example.com.us"
			}`,
			expectedOutput: `{
				"email1": "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
				"email2": "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
				"email3": "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
			}`,
		},
		{
			name: "url",
			input: `{
				"URL": "foo domain.com foo",
				"URL": "foo sub.domain.com foo",
				"URL": "foo sub.sub.domain.com foo",
				"URL": "foo sub.sub.sub.domain.com.us foo"
			}`,
			expectedOutput: `{
				"URL": "foo xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx foo",
				"URL": "foo xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx foo",
				"URL": "foo xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx foo",
				"URL": "foo xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx foo"
			}`,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			output := doOnJSON(test.input)
			assert.Equal(t, test.expectedOutput, output)
		})
	}
}
