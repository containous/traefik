package version

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/google/go-github/v28/github"
	"github.com/gorilla/mux"
	goversion "github.com/hashicorp/go-version"
	"github.com/traefik/traefik/v2/pkg/log"
	"github.com/unrolled/render"
)

var (
	// Version holds the current version of traefik.
	Version = "dev"
	// Codename holds the current version codename of traefik.
	Codename = "cheddar" // beta cheese
	// BuildDate holds the build date of traefik.
	BuildDate = "I don't remember exactly"
	// StartDate holds the start date of traefik.
	StartDate = time.Now()
	// UUID instance uuid.
	UUID string
	// PilotEnabled activate integration of pilot into the dashboard.
	PilotEnabled bool
)

// Handler expose version routes.
type Handler struct{}

var templatesRenderer = render.New(render.Options{
	Directory: "nowhere",
})

// Append adds version routes on a router.
func (v Handler) Append(router *mux.Router) {
	router.Methods(http.MethodGet).Path("/api/version").
		HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			v := struct {
				Version      string
				Codename     string
				StartDate    time.Time `json:"startDate"`
				UUID         string    `json:"uuid,omitempty"`
				PilotEnabled bool      `json:"pilotEnabled"`
			}{
				Version:      Version,
				Codename:     Codename,
				StartDate:    StartDate,
				UUID:         UUID,
				PilotEnabled: PilotEnabled,
			}

			if err := templatesRenderer.JSON(response, http.StatusOK, v); err != nil {
				log.WithoutContext().Error(err)
			}
		})
}

// CheckNewVersion checks if a new version is available.
func CheckNewVersion() {
	if Version == "dev" {
		return
	}

	logger := log.WithoutContext()

	client := github.NewClient(nil)

	updateURL, err := url.Parse("https://update.traefik.io/")
	if err != nil {
		logger.Warnf("Error checking new version: %s", err)
		return
	}
	client.BaseURL = updateURL

	releases, resp, err := client.Repositories.ListReleases(context.Background(), "traefik", "traefik", nil)
	if err != nil {
		logger.Warnf("Error checking new version: %s", err)
		return
	}

	if resp.StatusCode != http.StatusOK {
		logger.Warnf("Error checking new version: status=%s", resp.Status)
		return
	}

	currentVersion, err := goversion.NewVersion(Version)
	if err != nil {
		logger.Warnf("Error checking new version: %s", err)
		return
	}

	for _, release := range releases {
		releaseVersion, err := goversion.NewVersion(*release.TagName)
		if err != nil {
			logger.Warnf("Error checking new version: %s", err)
			return
		}

		if len(currentVersion.Prerelease()) == 0 && len(releaseVersion.Prerelease()) > 0 {
			continue
		}

		if releaseVersion.GreaterThan(currentVersion) {
			logger.Warnf("A new release has been found: %s. Please consider updating.", releaseVersion.String())
			return
		}
	}
}
