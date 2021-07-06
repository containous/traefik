package metrics

import (
	"context"
	"net/http"
	"strconv"
	"testing"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	httpApi "github.com/influxdata/influxdb-client-go/v2/api/http"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
	"github.com/influxdata/influxdb-client-go/v2/domain"
	"github.com/stretchr/testify/assert"
	"github.com/traefik/traefik/v2/pkg/types"
)

func TestInfluxDB2(t *testing.T) {
	influxDB2Client = &mockInfluxDB2Client{}
	influxDB2WriteAPI = &mockInfluxDB2WriteAPI{}

	c := &types.InfluxDB2{}
	c.SetDefaults()
	influxDB2Registry := RegisterInfluxDB2(context.Background(), c)
	defer StopInfluxDB2()

	if !influxDB2Registry.IsEpEnabled() || !influxDB2Registry.IsRouterEnabled() || !influxDB2Registry.IsSvcEnabled() {
		t.Fatalf("InfluxDB2Registry should return true for IsEnabled(), IsRouterEnabled() and IsSvcEnabled()")
	}

	influxDB2Registry.ConfigReloadsCounter().Add(1)
	influxDB2Registry.ConfigReloadsFailureCounter().Add(1)
	influxDB2Registry.LastConfigReloadSuccessGauge().Set(1)
	influxDB2Registry.LastConfigReloadFailureGauge().Set(1)

	assert.Equal(t, 4, len(mockInfluxDB2Points))
	assert.Equal(t, influxDB2ConfigReloadsName, mockInfluxDB2Points[0].FieldList()[0].Key)
	assert.Equal(t, 1.0, mockInfluxDB2Points[0].FieldList()[0].Value)
	assert.Equal(t, influxDB2ConfigReloadsFailureName, mockInfluxDB2Points[1].FieldList()[0].Key)
	assert.Equal(t, 1.0, mockInfluxDB2Points[1].FieldList()[0].Value)
	assert.Equal(t, influxDB2LastConfigReloadSuccessName, mockInfluxDB2Points[2].FieldList()[0].Key)
	assert.Equal(t, 1.0, mockInfluxDB2Points[2].FieldList()[0].Value)
	assert.Equal(t, influxDB2LastConfigReloadFailureName, mockInfluxDB2Points[3].FieldList()[0].Key)
	assert.Equal(t, 1.0, mockInfluxDB2Points[3].FieldList()[0].Value)

	mockInfluxDB2Points = make([]*write.Point, 0)

	influxDB2Registry.TLSCertsNotAfterTimestampGauge().With("key", "value").Set(1)

	assert.Equal(t, 1, len(mockInfluxDB2Points))
	assert.Equal(t, influxDB2TLSCertsNotAfterTimestampName, mockInfluxDB2Points[0].FieldList()[0].Key)
	assert.Equal(t, 1.0, mockInfluxDB2Points[0].FieldList()[0].Value)
	assert.Equal(t, "key", mockInfluxDB2Points[0].TagList()[0].Key)
	assert.Equal(t, "value", mockInfluxDB2Points[0].TagList()[0].Value)

	mockInfluxDB2Points = make([]*write.Point, 0)

	influxDB2Registry.EntryPointReqsCounter().With("entrypoint", "test", "code", strconv.Itoa(http.StatusOK), "method", http.MethodGet).Add(1)
	influxDB2Registry.EntryPointReqsTLSCounter().With("entrypoint", "test", "tls_version", "foo", "tls_cipher", "bar").Add(1)
	influxDB2Registry.EntryPointReqDurationHistogram().With("entrypoint", "test").Observe(10000)
	influxDB2Registry.EntryPointOpenConnsGauge().With("entrypoint", "test").Set(1)

	assert.Equal(t, 4, len(mockInfluxDB2Points))

	assert.Equal(t, influxDB2EntryPointReqsName, mockInfluxDB2Points[0].FieldList()[0].Key)
	assert.Equal(t, 1.0, mockInfluxDB2Points[0].FieldList()[0].Value)
	// Tags are sorted alphabetically while being processed
	assert.Equal(t, "code", mockInfluxDB2Points[0].TagList()[0].Key)
	assert.Equal(t, "200", mockInfluxDB2Points[0].TagList()[0].Value)
	assert.Equal(t, "entrypoint", mockInfluxDB2Points[0].TagList()[1].Key)
	assert.Equal(t, "test", mockInfluxDB2Points[0].TagList()[1].Value)
	assert.Equal(t, "method", mockInfluxDB2Points[0].TagList()[2].Key)
	assert.Equal(t, "GET", mockInfluxDB2Points[0].TagList()[2].Value)

	assert.Equal(t, influxDB2EntryPointReqsTLSName, mockInfluxDB2Points[1].FieldList()[0].Key)
	assert.Equal(t, 1.0, mockInfluxDB2Points[1].FieldList()[0].Value)
	assert.Equal(t, "entrypoint", mockInfluxDB2Points[1].TagList()[0].Key)
	assert.Equal(t, "test", mockInfluxDB2Points[1].TagList()[0].Value)
	assert.Equal(t, "tls_cipher", mockInfluxDB2Points[1].TagList()[1].Key)
	assert.Equal(t, "bar", mockInfluxDB2Points[1].TagList()[1].Value)
	assert.Equal(t, "tls_version", mockInfluxDB2Points[1].TagList()[2].Key)
	assert.Equal(t, "foo", mockInfluxDB2Points[1].TagList()[2].Value)

	assert.Equal(t, influxDB2EntryPointReqDurationName, mockInfluxDB2Points[2].FieldList()[0].Key)
	assert.Equal(t, 10000.0, mockInfluxDB2Points[2].FieldList()[0].Value)
	assert.Equal(t, "entrypoint", mockInfluxDB2Points[2].TagList()[0].Key)
	assert.Equal(t, "test", mockInfluxDB2Points[2].TagList()[0].Value)

	assert.Equal(t, influxDB2EntryPointOpenConnsName, mockInfluxDB2Points[3].FieldList()[0].Key)
	assert.Equal(t, 1.0, mockInfluxDB2Points[3].FieldList()[0].Value)
	assert.Equal(t, "entrypoint", mockInfluxDB2Points[3].TagList()[0].Key)
	assert.Equal(t, "test", mockInfluxDB2Points[3].TagList()[0].Value)

	mockInfluxDB2Points = make([]*write.Point, 0)

	influxDB2Registry.RouterReqsCounter().With("router", "demo", "service", "test", "code", strconv.Itoa(http.StatusNotFound), "method", http.MethodGet).Add(1)
	influxDB2Registry.RouterReqsCounter().With("router", "demo", "service", "test", "code", strconv.Itoa(http.StatusOK), "method", http.MethodGet).Add(1)
	influxDB2Registry.RouterReqsTLSCounter().With("router", "demo", "service", "test", "tls_version", "foo", "tls_cipher", "bar").Add(1)
	influxDB2Registry.RouterReqDurationHistogram().With("router", "demo", "service", "test", "code", strconv.Itoa(http.StatusOK)).Observe(10000)
	influxDB2Registry.RouterOpenConnsGauge().With("router", "demo", "service", "test").Set(1)

	assert.Equal(t, 5, len(mockInfluxDB2Points))

	assert.Equal(t, influxDB2RouterReqsName, mockInfluxDB2Points[0].FieldList()[0].Key)
	assert.Equal(t, 1.0, mockInfluxDB2Points[0].FieldList()[0].Value)
	assert.Equal(t, "code", mockInfluxDB2Points[0].TagList()[0].Key)
	assert.Equal(t, "404", mockInfluxDB2Points[0].TagList()[0].Value)
	assert.Equal(t, "method", mockInfluxDB2Points[0].TagList()[1].Key)
	assert.Equal(t, "GET", mockInfluxDB2Points[0].TagList()[1].Value)
	assert.Equal(t, "router", mockInfluxDB2Points[0].TagList()[2].Key)
	assert.Equal(t, "demo", mockInfluxDB2Points[0].TagList()[2].Value)
	assert.Equal(t, "service", mockInfluxDB2Points[0].TagList()[3].Key)
	assert.Equal(t, "test", mockInfluxDB2Points[0].TagList()[3].Value)

	assert.Equal(t, influxDB2RouterReqsName, mockInfluxDB2Points[1].FieldList()[0].Key)
	assert.Equal(t, 1.0, mockInfluxDB2Points[1].FieldList()[0].Value)
	assert.Equal(t, "code", mockInfluxDB2Points[1].TagList()[0].Key)
	assert.Equal(t, "200", mockInfluxDB2Points[1].TagList()[0].Value)
	assert.Equal(t, "method", mockInfluxDB2Points[1].TagList()[1].Key)
	assert.Equal(t, "GET", mockInfluxDB2Points[1].TagList()[1].Value)
	assert.Equal(t, "router", mockInfluxDB2Points[1].TagList()[2].Key)
	assert.Equal(t, "demo", mockInfluxDB2Points[1].TagList()[2].Value)
	assert.Equal(t, "service", mockInfluxDB2Points[1].TagList()[3].Key)
	assert.Equal(t, "test", mockInfluxDB2Points[1].TagList()[3].Value)

	assert.Equal(t, influxDB2RouterReqsTLSName, mockInfluxDB2Points[2].FieldList()[0].Key)
	assert.Equal(t, 1.0, mockInfluxDB2Points[2].FieldList()[0].Value)
	assert.Equal(t, "router", mockInfluxDB2Points[2].TagList()[0].Key)
	assert.Equal(t, "demo", mockInfluxDB2Points[2].TagList()[0].Value)
	assert.Equal(t, "service", mockInfluxDB2Points[2].TagList()[1].Key)
	assert.Equal(t, "test", mockInfluxDB2Points[2].TagList()[1].Value)
	assert.Equal(t, "tls_cipher", mockInfluxDB2Points[2].TagList()[2].Key)
	assert.Equal(t, "bar", mockInfluxDB2Points[2].TagList()[2].Value)
	assert.Equal(t, "tls_version", mockInfluxDB2Points[2].TagList()[3].Key)
	assert.Equal(t, "foo", mockInfluxDB2Points[2].TagList()[3].Value)

	assert.Equal(t, influxDB2RouterReqsDurationName, mockInfluxDB2Points[3].FieldList()[0].Key)
	assert.Equal(t, 10000.0, mockInfluxDB2Points[3].FieldList()[0].Value)
	assert.Equal(t, "code", mockInfluxDB2Points[3].TagList()[0].Key)
	assert.Equal(t, "200", mockInfluxDB2Points[3].TagList()[0].Value)
	assert.Equal(t, "router", mockInfluxDB2Points[3].TagList()[1].Key)
	assert.Equal(t, "demo", mockInfluxDB2Points[3].TagList()[1].Value)
	assert.Equal(t, "service", mockInfluxDB2Points[3].TagList()[2].Key)
	assert.Equal(t, "test", mockInfluxDB2Points[3].TagList()[2].Value)

	assert.Equal(t, influxDB2RouterOpenConnsName, mockInfluxDB2Points[4].FieldList()[0].Key)
	assert.Equal(t, 1.0, mockInfluxDB2Points[4].FieldList()[0].Value)
	assert.Equal(t, "router", mockInfluxDB2Points[4].TagList()[0].Key)
	assert.Equal(t, "demo", mockInfluxDB2Points[4].TagList()[0].Value)
	assert.Equal(t, "service", mockInfluxDB2Points[4].TagList()[1].Key)
	assert.Equal(t, "test", mockInfluxDB2Points[4].TagList()[1].Value)

	mockInfluxDB2Points = make([]*write.Point, 0)

	influxDB2Registry.ServiceReqsCounter().With("service", "test", "code", strconv.Itoa(http.StatusOK), "method", http.MethodGet).Add(1)
	influxDB2Registry.ServiceReqsCounter().With("service", "test", "code", strconv.Itoa(http.StatusNotFound), "method", http.MethodGet).Add(1)
	influxDB2Registry.ServiceReqsTLSCounter().With("service", "test", "tls_version", "foo", "tls_cipher", "bar").Add(1)
	influxDB2Registry.ServiceReqDurationHistogram().With("service", "test", "code", strconv.Itoa(http.StatusOK)).Observe(10000)
	influxDB2Registry.ServiceOpenConnsGauge().With("service", "test").Set(1)
	influxDB2Registry.ServiceRetriesCounter().With("service", "test").Add(1)
	influxDB2Registry.ServiceRetriesCounter().With("service", "test").Add(1)
	influxDB2Registry.ServiceServerUpGauge().With("service", "test", "url", "http://127.0.0.1").Set(1)

	assert.Equal(t, 8, len(mockInfluxDB2Points))

	assert.Equal(t, influxDB2MetricsServiceReqsName, mockInfluxDB2Points[0].FieldList()[0].Key)
	assert.Equal(t, 1.0, mockInfluxDB2Points[0].FieldList()[0].Value)
	assert.Equal(t, "code", mockInfluxDB2Points[0].TagList()[0].Key)
	assert.Equal(t, "200", mockInfluxDB2Points[0].TagList()[0].Value)
	assert.Equal(t, "method", mockInfluxDB2Points[0].TagList()[1].Key)
	assert.Equal(t, "GET", mockInfluxDB2Points[0].TagList()[1].Value)
	assert.Equal(t, "service", mockInfluxDB2Points[0].TagList()[2].Key)
	assert.Equal(t, "test", mockInfluxDB2Points[0].TagList()[2].Value)

	assert.Equal(t, influxDB2MetricsServiceReqsName, mockInfluxDB2Points[1].FieldList()[0].Key)
	assert.Equal(t, 1.0, mockInfluxDB2Points[1].FieldList()[0].Value)
	assert.Equal(t, "code", mockInfluxDB2Points[1].TagList()[0].Key)
	assert.Equal(t, "404", mockInfluxDB2Points[1].TagList()[0].Value)
	assert.Equal(t, "method", mockInfluxDB2Points[1].TagList()[1].Key)
	assert.Equal(t, "GET", mockInfluxDB2Points[1].TagList()[1].Value)
	assert.Equal(t, "service", mockInfluxDB2Points[1].TagList()[2].Key)
	assert.Equal(t, "test", mockInfluxDB2Points[1].TagList()[2].Value)

	assert.Equal(t, influxDB2MetricsServiceReqsTLSName, mockInfluxDB2Points[2].FieldList()[0].Key)
	assert.Equal(t, 1.0, mockInfluxDB2Points[2].FieldList()[0].Value)
	assert.Equal(t, "service", mockInfluxDB2Points[2].TagList()[0].Key)
	assert.Equal(t, "test", mockInfluxDB2Points[2].TagList()[0].Value)
	assert.Equal(t, "tls_cipher", mockInfluxDB2Points[2].TagList()[1].Key)
	assert.Equal(t, "bar", mockInfluxDB2Points[2].TagList()[1].Value)
	assert.Equal(t, "tls_version", mockInfluxDB2Points[2].TagList()[2].Key)
	assert.Equal(t, "foo", mockInfluxDB2Points[2].TagList()[2].Value)

	assert.Equal(t, influxDB2MetricsServiceLatencyName, mockInfluxDB2Points[3].FieldList()[0].Key)
	assert.Equal(t, 10000.0, mockInfluxDB2Points[3].FieldList()[0].Value)
	assert.Equal(t, "code", mockInfluxDB2Points[3].TagList()[0].Key)
	assert.Equal(t, "200", mockInfluxDB2Points[3].TagList()[0].Value)
	assert.Equal(t, "service", mockInfluxDB2Points[3].TagList()[1].Key)
	assert.Equal(t, "test", mockInfluxDB2Points[3].TagList()[1].Value)

	assert.Equal(t, influxDB2OpenConnsName, mockInfluxDB2Points[4].FieldList()[0].Key)
	assert.Equal(t, 1.0, mockInfluxDB2Points[4].FieldList()[0].Value)
	assert.Equal(t, "service", mockInfluxDB2Points[4].TagList()[0].Key)
	assert.Equal(t, "test", mockInfluxDB2Points[4].TagList()[0].Value)

	assert.Equal(t, influxDB2RetriesTotalName, mockInfluxDB2Points[5].FieldList()[0].Key)
	assert.Equal(t, 1.0, mockInfluxDB2Points[5].FieldList()[0].Value)
	assert.Equal(t, "service", mockInfluxDB2Points[5].TagList()[0].Key)
	assert.Equal(t, "test", mockInfluxDB2Points[5].TagList()[0].Value)
	assert.Equal(t, influxDB2RetriesTotalName, mockInfluxDB2Points[6].FieldList()[0].Key)
	assert.Equal(t, 2.0, mockInfluxDB2Points[6].FieldList()[0].Value)
	assert.Equal(t, "service", mockInfluxDB2Points[6].TagList()[0].Key)
	assert.Equal(t, "test", mockInfluxDB2Points[6].TagList()[0].Value)

	assert.Equal(t, influxDB2ServerUpName, mockInfluxDB2Points[7].FieldList()[0].Key)
	assert.Equal(t, 1.0, mockInfluxDB2Points[7].FieldList()[0].Value)
	assert.Equal(t, "service", mockInfluxDB2Points[7].TagList()[0].Key)
	assert.Equal(t, "test", mockInfluxDB2Points[7].TagList()[0].Value)
	assert.Equal(t, "url", mockInfluxDB2Points[7].TagList()[1].Key)
	assert.Equal(t, "http://127.0.0.1", mockInfluxDB2Points[7].TagList()[1].Value)
}

type mockInfluxDB2Client struct{}

func (c *mockInfluxDB2Client) Setup(ctx context.Context, username, password, org, bucket string, retentionPeriodHours int) (*domain.OnboardingResponse, error) {
	return nil, nil
}
func (c *mockInfluxDB2Client) Ready(ctx context.Context) (bool, error) {
	return true, nil
}
func (c *mockInfluxDB2Client) Health(ctx context.Context) (*domain.HealthCheck, error) {
	return nil, nil
}
func (c *mockInfluxDB2Client) Close() {}
func (c *mockInfluxDB2Client) Options() *influxdb2.Options {
	return nil
}
func (c *mockInfluxDB2Client) ServerURL() string {
	return ""
}
func (c *mockInfluxDB2Client) HTTPService() httpApi.Service {
	return nil
}
func (c *mockInfluxDB2Client) WriteAPI(org, bucket string) api.WriteAPI {
	return nil
}
func (c *mockInfluxDB2Client) WriteAPIBlocking(org, bucket string) api.WriteAPIBlocking {
	return nil
}
func (c *mockInfluxDB2Client) QueryAPI(org string) api.QueryAPI {
	return nil
}
func (c *mockInfluxDB2Client) AuthorizationsAPI() api.AuthorizationsAPI {
	return nil
}
func (c *mockInfluxDB2Client) OrganizationsAPI() api.OrganizationsAPI {
	return nil
}
func (c *mockInfluxDB2Client) UsersAPI() api.UsersAPI {
	return nil
}
func (c *mockInfluxDB2Client) DeleteAPI() api.DeleteAPI {
	return nil
}
func (c *mockInfluxDB2Client) BucketsAPI() api.BucketsAPI {
	return nil
}
func (c *mockInfluxDB2Client) LabelsAPI() api.LabelsAPI {
	return nil
}
func (c *mockInfluxDB2Client) TasksAPI() api.TasksAPI {
	return nil
}

type mockInfluxDB2WriteAPI struct{}

var mockInfluxDB2Points = make([]*write.Point, 0)

func (w *mockInfluxDB2WriteAPI) WriteRecord(line string) {}
func (w *mockInfluxDB2WriteAPI) WritePoint(point *write.Point) {
	mockInfluxDB2Points = append(mockInfluxDB2Points, point)
}
func (w *mockInfluxDB2WriteAPI) Flush() {}
func (w *mockInfluxDB2WriteAPI) Errors() <-chan error {
	return make(chan error)
}
