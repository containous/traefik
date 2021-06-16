package metrics

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"regexp"
	"time"

	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics/influx"
	influxdb "github.com/influxdata/influxdb1-client/v2"
	"github.com/traefik/traefik/v2/pkg/log"
	"github.com/traefik/traefik/v2/pkg/safe"
	"github.com/traefik/traefik/v2/pkg/types"
)

var influxDBClient *influx.Influx

type influxDBWriter struct {
	buf    bytes.Buffer
	config *types.InfluxDB
}

var influxDBTicker *time.Ticker

const (
	influxDBConfigReloadsName           = "traefik.config.reload.total"
	influxDBConfigReloadsFailureName    = influxDBConfigReloadsName + ".failure"
	influxDBLastConfigReloadSuccessName = "traefik.config.reload.lastSuccessTimestamp"
	influxDBLastConfigReloadFailureName = "traefik.config.reload.lastFailureTimestamp"

	influxDBTLSCertsNotAfterTimestampName = "traefik.tls.certs.notAfterTimestamp"

	influxDBEntryPointReqsName        = "traefik.entrypoint.requests.total"
	influxDBEntryPointReqsTLSName     = "traefik.entrypoint.requests.tls.total"
	influxDBEntryPointReqDurationName = "traefik.entrypoint.request.duration"
	influxDBEntryPointOpenConnsName   = "traefik.entrypoint.connections.open"

	influxDBRouterReqsName         = "traefik.router.requests.total"
	influxDBRouterReqsTLSName      = "traefik.router.requests.tls.total"
	influxDBRouterReqsDurationName = "traefik.router.request.duration"
	influxDBORouterOpenConnsName   = "traefik.router.connections.open"

	influxDBServiceReqsName         = "traefik.service.requests.total"
	influxDBServiceReqsTLSName      = "traefik.service.requests.tls.total"
	influxDBServiceReqsDurationName = "traefik.service.request.duration"
	influxDBServiceRetriesTotalName = "traefik.service.retries.total"
	influxDBServiceOpenConnsName    = "traefik.service.connections.open"
	influxDBServiceServerUpName     = "traefik.service.server.up"
)

const (
	protocolHTTP = "http"
	protocolUDP  = "udp"
)

// RegisterInfluxDB registers the metrics pusher if this didn't happen yet and creates a InfluxDB Registry instance.
func RegisterInfluxDB(ctx context.Context, config *types.InfluxDB) Registry {
	if influxDBClient == nil {
		influxDBClient = initInfluxDBClient(ctx, config)
	}
	if influxDBTicker == nil {
		influxDBTicker = initInfluxDBTicker(ctx, config)
	}

	registry := &standardRegistry{
		configReloadsCounter:           influxDBClient.NewCounter(influxDBConfigReloadsName),
		configReloadsFailureCounter:    influxDBClient.NewCounter(influxDBConfigReloadsFailureName),
		lastConfigReloadSuccessGauge:   influxDBClient.NewGauge(influxDBLastConfigReloadSuccessName),
		lastConfigReloadFailureGauge:   influxDBClient.NewGauge(influxDBLastConfigReloadFailureName),
		tlsCertsNotAfterTimestampGauge: influxDBClient.NewGauge(influxDBTLSCertsNotAfterTimestampName),
	}

	if config.AddEntryPointsLabels {
		registry.epEnabled = config.AddEntryPointsLabels
		registry.entryPointReqsCounter = influxDBClient.NewCounter(influxDBEntryPointReqsName)
		registry.entryPointReqsTLSCounter = influxDBClient.NewCounter(influxDBEntryPointReqsTLSName)
		registry.entryPointReqDurationHistogram, _ = NewHistogramWithScale(influxDBClient.NewHistogram(influxDBEntryPointReqDurationName), time.Second)
		registry.entryPointOpenConnsGauge = influxDBClient.NewGauge(influxDBEntryPointOpenConnsName)
	}

	if config.AddRoutersLabels {
		registry.routerEnabled = config.AddRoutersLabels
		registry.routerReqsCounter = influxDBClient.NewCounter(influxDBRouterReqsName)
		registry.routerReqsTLSCounter = influxDBClient.NewCounter(influxDBRouterReqsTLSName)
		registry.routerReqDurationHistogram, _ = NewHistogramWithScale(influxDBClient.NewHistogram(influxDBRouterReqsDurationName), time.Second)
		registry.routerOpenConnsGauge = influxDBClient.NewGauge(influxDBORouterOpenConnsName)
	}

	if config.AddServicesLabels {
		registry.svcEnabled = config.AddServicesLabels
		registry.serviceReqsCounter = influxDBClient.NewCounter(influxDBServiceReqsName)
		registry.serviceReqsTLSCounter = influxDBClient.NewCounter(influxDBServiceReqsTLSName)
		registry.serviceReqDurationHistogram, _ = NewHistogramWithScale(influxDBClient.NewHistogram(influxDBServiceReqsDurationName), time.Second)
		registry.serviceRetriesCounter = influxDBClient.NewCounter(influxDBServiceRetriesTotalName)
		registry.serviceOpenConnsGauge = influxDBClient.NewGauge(influxDBServiceOpenConnsName)
		registry.serviceServerUpGauge = influxDBClient.NewGauge(influxDBServiceServerUpName)
	}

	return registry
}

// initInfluxDBTicker creates a influxDBClient.
func initInfluxDBClient(ctx context.Context, config *types.InfluxDB) *influx.Influx {
	logger := log.FromContext(ctx)

	// TODO deprecated: move this switch into configuration.SetEffectiveConfiguration when web provider will be removed.
	switch config.Protocol {
	case protocolUDP:
		if len(config.Database) > 0 || len(config.RetentionPolicy) > 0 {
			logger.Warn("Database and RetentionPolicy options have no effect with UDP.")
			config.Database = ""
			config.RetentionPolicy = ""
		}
	case protocolHTTP:
		if u, err := url.Parse(config.Address); err == nil {
			if u.Scheme != "http" && u.Scheme != "https" {
				logger.Warnf("InfluxDB address %s should specify a scheme (http or https): falling back on HTTP.", config.Address)
				config.Address = "http://" + config.Address
			}
		} else {
			logger.Errorf("Unable to parse the InfluxDB address %v: falling back on UDP.", err)
			config.Protocol = protocolUDP
			config.Database = ""
			config.RetentionPolicy = ""
		}
	default:
		logger.Warnf("Unsupported protocol %s: falling back on UDP.", config.Protocol)
		config.Protocol = protocolUDP
		config.Database = ""
		config.RetentionPolicy = ""
	}

	return influx.New(
		map[string]string{},
		influxdb.BatchPointsConfig{
			Database:        config.Database,
			RetentionPolicy: config.RetentionPolicy,
		},
		kitlog.LoggerFunc(func(keyvals ...interface{}) error {
			log.WithoutContext().WithField(log.MetricsProviderName, "influxdb").Info(keyvals)
			return nil
		}))
}

// initInfluxDBTicker initializes metrics pusher.
func initInfluxDBTicker(ctx context.Context, config *types.InfluxDB) *time.Ticker {
	report := time.NewTicker(time.Duration(config.PushInterval))

	safe.Go(func() {
		var buf bytes.Buffer
		influxDBClient.WriteLoop(ctx, report.C, &influxDBWriter{buf: buf, config: config})
	})

	return report
}

// StopInfluxDB stops internal influxDBTicker which controls the pushing of metrics to InfluxDB Agent and resets it to `nil`.
func StopInfluxDB() {
	if influxDBTicker != nil {
		influxDBTicker.Stop()
	}
	influxDBTicker = nil
}

// Write creates a http or udp client and attempts to write BatchPoints.
// If a "database not found" error is encountered, a CREATE DATABASE
// query is attempted when using protocol http.
func (w *influxDBWriter) Write(bp influxdb.BatchPoints) error {
	c, err := w.initWriteClient()
	if err != nil {
		return err
	}

	defer c.Close()

	if writeErr := c.Write(bp); writeErr != nil {
		ctx := log.With(context.Background(), log.Str(log.MetricsProviderName, "influxdb"))
		log.FromContext(ctx).Errorf("Error while writing to InfluxDB: %s", writeErr.Error())

		if handleErr := w.handleWriteError(ctx, c, writeErr); handleErr != nil {
			return handleErr
		}
		// Retry write after successful handling of writeErr
		return c.Write(bp)
	}
	return nil
}

func (w *influxDBWriter) initWriteClient() (influxdb.Client, error) {
	if w.config.Protocol == "http" {
		return influxdb.NewHTTPClient(influxdb.HTTPConfig{
			Addr:     w.config.Address,
			Username: w.config.Username,
			Password: w.config.Password,
		})
	}

	return influxdb.NewUDPClient(influxdb.UDPConfig{
		Addr: w.config.Address,
	})
}

func (w *influxDBWriter) handleWriteError(ctx context.Context, c influxdb.Client, writeErr error) error {
	if w.config.Protocol != protocolHTTP {
		return writeErr
	}

	match, matchErr := regexp.MatchString("database not found", writeErr.Error())

	if matchErr != nil || !match {
		return writeErr
	}

	qStr := fmt.Sprintf("CREATE DATABASE \"%s\"", w.config.Database)
	if w.config.RetentionPolicy != "" {
		qStr = fmt.Sprintf("%s WITH NAME \"%s\"", qStr, w.config.RetentionPolicy)
	}

	logger := log.FromContext(ctx)

	logger.Debugf("InfluxDB database not found: attempting to create one with %s", qStr)

	q := influxdb.NewQuery(qStr, "", "")
	response, queryErr := c.Query(q)
	if queryErr == nil && response.Error() != nil {
		queryErr = response.Error()
	}
	if queryErr != nil {
		logger.Errorf("Error while creating the InfluxDB database %s", queryErr)
		return queryErr
	}

	logger.Debugf("Successfully created the InfluxDB database %s", w.config.Database)
	return nil
}
