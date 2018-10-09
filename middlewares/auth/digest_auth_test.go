package auth

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/containous/traefik/config"
	"github.com/containous/traefik/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDigestAuthError(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "traefik")
	})

	auth := config.DigestAuth{
		Users: []string{"test"},
	}
	_, err := NewDigest(context.Background(), next, auth, "authName")
	assert.Error(t, err)
}

func TestDigestAuthFail(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "traefik")
	})

	auth := config.DigestAuth{
		Users: []string{"test:traefik:a2688e031edb4be6a3797f3882655c05"},
	}
	authMiddleware, err := NewDigest(context.Background(), next, auth, "authName")
	require.NoError(t, err)
	assert.NotNil(t, authMiddleware, "this should not be nil")

	ts := httptest.NewServer(authMiddleware)
	defer ts.Close()

	client := http.DefaultClient
	req := testhelpers.MustNewRequest(http.MethodGet, ts.URL, nil)
	req.SetBasicAuth("test", "test")

	res, err := client.Do(req)
	require.NoError(t, err)

	assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
}

func TestDigestAuthUsersFromFile(t *testing.T) {
	testCases := []struct {
		desc            string
		userFileContent string
		expectedUsers   map[string]string
		givenUsers      []string
		realm           string
	}{
		{
			desc:            "Finds the users in the file",
			userFileContent: "test:traefik:a2688e031edb4be6a3797f3882655c05\ntest2:traefik:518845800f9e2bfb1f1f740ec24f074e\n",
			givenUsers:      []string{},
			expectedUsers:   map[string]string{"test": "test", "test2": "test2"},
		},
		{
			desc:            "Merges given users with users from the file",
			userFileContent: "test:traefik:a2688e031edb4be6a3797f3882655c05\n",
			givenUsers:      []string{"test2:traefik:518845800f9e2bfb1f1f740ec24f074e", "test3:traefik:c8e9f57ce58ecb4424407f665a91646c"},
			expectedUsers:   map[string]string{"test": "test", "test2": "test2", "test3": "test3"},
		},
		{
			desc:            "Given users have priority over users in the file",
			userFileContent: "test:traefik:a2688e031edb4be6a3797f3882655c05\ntest2:traefik:518845800f9e2bfb1f1f740ec24f074e\n",
			givenUsers:      []string{"test2:traefik:8de60a1c52da68ccf41f0c0ffb7c51a0"},
			expectedUsers:   map[string]string{"test": "test", "test2": "overridden"},
		},
		{
			desc:            "Should authenticate the correct user based on the realm",
			userFileContent: "test:traefik:a2688e031edb4be6a3797f3882655c05\ntest:traefikee:316a669c158c8b7ab1048b03961a7aa5\n",
			givenUsers:      []string{},
			expectedUsers:   map[string]string{"test": "test2"},
			realm:           "traefikee",
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			// Creates the temporary configuration file with the users
			usersFile, err := ioutil.TempFile("", "auth-users")
			require.NoError(t, err)
			defer os.Remove(usersFile.Name())

			_, err = usersFile.Write([]byte(test.userFileContent))
			require.NoError(t, err)

			// Creates the configuration for our Authenticator
			authenticatorConfiguration := config.DigestAuth{
				Users:     test.givenUsers,
				UsersFile: usersFile.Name(),
				Realm:     test.realm,
			}

			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintln(w, "traefik")
			})

			authenticator, err := NewDigest(context.Background(), next, authenticatorConfiguration, "authName")
			require.NoError(t, err)

			ts := httptest.NewServer(authenticator)
			defer ts.Close()

			for userName, userPwd := range test.expectedUsers {
				req := testhelpers.MustNewRequest(http.MethodGet, ts.URL, nil)
				digestRequest := newDigestRequest(userName, userPwd, http.DefaultClient)

				var res *http.Response
				res, err = digestRequest.Do(req)
				require.NoError(t, err)
				require.Equal(t, http.StatusOK, res.StatusCode, "Cannot authenticate user "+userName)

				var body []byte
				body, err = ioutil.ReadAll(res.Body)
				require.NoError(t, err)
				err = res.Body.Close()
				require.NoError(t, err)

				require.Equal(t, "traefik\n", string(body))
			}

			// Checks that user foo doesn't work
			req := testhelpers.MustNewRequest(http.MethodGet, ts.URL, nil)
			digestRequest := newDigestRequest("foo", "foo", http.DefaultClient)

			var res *http.Response
			res, err = digestRequest.Do(req)
			require.NoError(t, err)
			require.Equal(t, http.StatusUnauthorized, res.StatusCode)

			var body []byte
			body, err = ioutil.ReadAll(res.Body)
			require.NoError(t, err)
			err = res.Body.Close()
			require.NoError(t, err)

			require.NotContains(t, "traefik", string(body))
		})
	}
}
