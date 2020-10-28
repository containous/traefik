package main

import (
	"context"
	"encoding/json"
	stdlog "log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/coreos/go-systemd/daemon"
	assetfs "github.com/elazarl/go-bindata-assetfs"
	"github.com/sirupsen/logrus"
	"github.com/traefik/paerser/cli"
	"github.com/traefik/traefik/v2/autogen/genstatic"
	"github.com/traefik/traefik/v2/cmd"
	"github.com/traefik/traefik/v2/cmd/healthcheck"
	cmdVersion "github.com/traefik/traefik/v2/cmd/version"
	tcli "github.com/traefik/traefik/v2/pkg/cli"
	"github.com/traefik/traefik/v2/pkg/collector"
	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	"github.com/traefik/traefik/v2/pkg/config/runtime"
	"github.com/traefik/traefik/v2/pkg/config/static"
	"github.com/traefik/traefik/v2/pkg/log"
	"github.com/traefik/traefik/v2/pkg/metrics"
	"github.com/traefik/traefik/v2/pkg/middlewares/accesslog"
	"github.com/traefik/traefik/v2/pkg/pilot"
	"github.com/traefik/traefik/v2/pkg/plugins"
	"github.com/traefik/traefik/v2/pkg/provider/acme"
	"github.com/traefik/traefik/v2/pkg/provider/aggregator"
	"github.com/traefik/traefik/v2/pkg/provider/traefik"
	"github.com/traefik/traefik/v2/pkg/safe"
	"github.com/traefik/traefik/v2/pkg/server"
	"github.com/traefik/traefik/v2/pkg/server/middleware"
	"github.com/traefik/traefik/v2/pkg/server/service"
	traefiktls "github.com/traefik/traefik/v2/pkg/tls"
	"github.com/traefik/traefik/v2/pkg/types"
	"github.com/traefik/traefik/v2/pkg/version"
	"github.com/vulcand/oxy/roundrobin"
)

func main() {
	// traefik config inits
	tConfig := cmd.NewTraefikConfiguration()

	loaders := []cli.ResourceLoader{&tcli.FileLoader{}, &tcli.FlagLoader{}, &tcli.EnvLoader{}}

	cmdTraefik := &cli.Command{
		Name: "traefik",
		Description: `Traefik is a modern HTTP reverse proxy and load balancer made to deploy microservices with ease.
Complete documentation is available at https://traefik.io`,
		Configuration: tConfig,
		Resources:     loaders,
		Run: func(_ []string) error {
			return runCmd(&tConfig.Configuration)
		},
	}

	err := cmdTraefik.AddCommand(healthcheck.NewCmd(&tConfig.Configuration, loaders))
	if err != nil {
		stdlog.Println(err)
		os.Exit(1)
	}

	err = cmdTraefik.AddCommand(cmdVersion.NewCmd())
	if err != nil {
		stdlog.Println(err)
		os.Exit(1)
	}

	err = cli.Execute(cmdTraefik)
	if err != nil {
		stdlog.Println(err)
		logrus.Exit(1)
	}

	logrus.Exit(0)
}

func runCmd(staticConfiguration *static.Configuration) error {
	configureLogging(staticConfiguration)

	http.DefaultTransport.(*http.Transport).Proxy = http.ProxyFromEnvironment

	if err := roundrobin.SetDefaultWeight(0); err != nil {
		log.WithoutContext().Errorf("Could not set round robin default weight: %v", err)
	}

	staticConfiguration.SetEffectiveConfiguration()
	if err := staticConfiguration.ValidateConfiguration(); err != nil {
		return err
	}

	log.WithoutContext().Infof("Traefik version %s built on %s", version.Version, version.BuildDate)

	jsonConf, err := json.Marshal(staticConfiguration)
	if err != nil {
		log.WithoutContext().Errorf("Could not marshal static configuration: %v", err)
		log.WithoutContext().Debugf("Static configuration loaded [struct] %#v", staticConfiguration)
	} else {
		log.WithoutContext().Debugf("Static configuration loaded %s", string(jsonConf))
	}

	if staticConfiguration.API != nil && staticConfiguration.API.Dashboard {
		staticConfiguration.API.DashboardAssets = &assetfs.AssetFS{Asset: genstatic.Asset, AssetInfo: genstatic.AssetInfo, AssetDir: genstatic.AssetDir, Prefix: "static"}
	}

	if staticConfiguration.Global.CheckNewVersion {
		checkNewVersion()
	}

	stats(staticConfiguration)

	svr, err := setupServer(staticConfiguration)
	if err != nil {
		return err
	}

	ctx := cmd.ContextWithSignal(context.Background())

	if staticConfiguration.Experimental != nil && staticConfiguration.Experimental.DevPlugin != nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 30*time.Minute)
		defer cancel()
	}

	if staticConfiguration.Ping != nil {
		staticConfiguration.Ping.WithContext(ctx)
	}

	svr.Start(ctx)
	defer svr.Close()

	sent, err := daemon.SdNotify(false, "READY=1")
	if !sent && err != nil {
		log.WithoutContext().Errorf("Failed to notify: %v", err)
	}

	t, err := daemon.SdWatchdogEnabled(false)
	if err != nil {
		log.WithoutContext().Errorf("Could not enable Watchdog: %v", err)
	} else if t != 0 {
		// Send a ping each half time given
		t /= 2
		log.WithoutContext().Infof("Watchdog activated with timer duration %s", t)
		safe.Go(func() {
			tick := time.Tick(t)
			for range tick {
				resp, errHealthCheck := healthcheck.Do(*staticConfiguration)
				if resp != nil {
					_ = resp.Body.Close()
				}

				if staticConfiguration.Ping == nil || errHealthCheck == nil {
					if ok, _ := daemon.SdNotify(false, "WATCHDOG=1"); !ok {
						log.WithoutContext().Error("Fail to tick watchdog")
					}
				} else {
					log.WithoutContext().Error(errHealthCheck)
				}
			}
		})
	}

	svr.Wait()
	log.WithoutContext().Info("Shutting down")
	return nil
}

func setupServer(staticConfiguration *static.Configuration) (*server.Server, error) {
	providerAggregator := aggregator.NewProviderAggregator(*staticConfiguration.Providers)

	// adds internal provider
	err := providerAggregator.AddProvider(traefik.New(*staticConfiguration))
	if err != nil {
		return nil, err
	}

	tlsManager := traefiktls.NewManager()

	acmeProviders := initACMEProvider(staticConfiguration, &providerAggregator, tlsManager)

	serverEntryPointsTCP, err := server.NewTCPEntryPoints(staticConfiguration.EntryPoints)
	if err != nil {
		return nil, err
	}

	serverEntryPointsUDP, err := server.NewUDPEntryPoints(staticConfiguration.EntryPoints)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	routinesPool := safe.NewPool(ctx)

	metricRegistries := registerMetricClients(staticConfiguration.Metrics)

	var aviator *pilot.Pilot
	if isPilotEnabled(staticConfiguration) {
		pilotRegistry := metrics.RegisterPilot()

		aviator = pilot.New(staticConfiguration.Pilot.Token, pilotRegistry, routinesPool)
		routinesPool.GoCtx(func(ctx context.Context) {
			aviator.Tick(ctx)
		})

		metricRegistries = append(metricRegistries, pilotRegistry)
	}

	metricsRegistry := metrics.NewMultiRegistry(metricRegistries)
	accessLog := setupAccessLog(staticConfiguration.AccessLog)
	chainBuilder := middleware.NewChainBuilder(*staticConfiguration, metricsRegistry, accessLog)
	managerFactory := service.NewManagerFactory(*staticConfiguration, routinesPool, metricsRegistry)

	client, plgs, devPlugin, err := initPlugins(staticConfiguration)
	if err != nil {
		return nil, err
	}

	pluginBuilder, err := plugins.NewBuilder(client, plgs, devPlugin)
	if err != nil {
		return nil, err
	}

	routerFactory := server.NewRouterFactory(*staticConfiguration, managerFactory, tlsManager, chainBuilder, pluginBuilder)

	var defaultEntryPoints []string
	for name, cfg := range staticConfiguration.EntryPoints {
		protocol, err := cfg.GetProtocol()
		if err != nil {
			// Should never happen because Traefik should not start if protocol is invalid.
			log.WithoutContext().Errorf("Invalid protocol: %v", err)
		}

		if protocol != "udp" && name != static.DefaultInternalEntryPointName {
			defaultEntryPoints = append(defaultEntryPoints, name)
		}
	}

	sort.Strings(defaultEntryPoints)

	watcher := server.NewConfigurationWatcher(
		routinesPool,
		providerAggregator,
		time.Duration(staticConfiguration.Providers.ProvidersThrottleDuration),
		defaultEntryPoints,
	)

	watcher.AddListener(func(conf dynamic.Configuration) {
		ctx := context.Background()
		tlsManager.UpdateConfigs(ctx, conf.TLS.Stores, conf.TLS.Options, conf.TLS.Certificates)
	})

	watcher.AddListener(func(_ dynamic.Configuration) {
		metricsRegistry.ConfigReloadsCounter().Add(1)
		metricsRegistry.LastConfigReloadSuccessGauge().Set(float64(time.Now().Unix()))
	})

	watcher.AddListener(switchRouter(routerFactory, acmeProviders, serverEntryPointsTCP, serverEntryPointsUDP, aviator))

	watcher.AddListener(func(conf dynamic.Configuration) {
		if metricsRegistry.IsEpEnabled() || metricsRegistry.IsSvcEnabled() {
			var eps []string
			for key := range serverEntryPointsTCP {
				eps = append(eps, key)
			}

			metrics.OnConfigurationUpdate(conf, eps)
		}
	})

	resolverNames := map[string]struct{}{}
	for _, p := range acmeProviders {
		resolverNames[p.ResolverName] = struct{}{}
		watcher.AddListener(p.ListenConfiguration)
	}

	watcher.AddListener(func(config dynamic.Configuration) {
		for rtName, rt := range config.HTTP.Routers {
			if rt.TLS == nil || rt.TLS.CertResolver == "" {
				continue
			}

			if _, ok := resolverNames[rt.TLS.CertResolver]; !ok {
				log.WithoutContext().Errorf("the router %s uses a non-existent resolver: %s", rtName, rt.TLS.CertResolver)
			}
		}
	})

	watcher.AddConfigLoadErrorListener(func() {
		metricsRegistry.ConfigReloadsFailureCounter().Add(1)
		metricsRegistry.LastConfigReloadFailureGauge().Set(float64(time.Now().Unix()))
	})

	return server.NewServer(routinesPool, serverEntryPointsTCP, serverEntryPointsUDP, watcher, chainBuilder, accessLog), nil
}

func switchRouter(routerFactory *server.RouterFactory, acmeProviders []*acme.Provider, serverEntryPointsTCP server.TCPEntryPoints, serverEntryPointsUDP server.UDPEntryPoints, aviator *pilot.Pilot) func(conf dynamic.Configuration) {
	return func(conf dynamic.Configuration) {
		rtConf := runtime.NewConfig(conf)

		routers, udpRouters := routerFactory.CreateRouters(rtConf)

		for entryPointName, rt := range routers {
			for _, p := range acmeProviders {
				if p != nil && p.HTTPChallenge != nil && p.HTTPChallenge.EntryPoint == entryPointName {
					rt.HTTPHandler(p.CreateHandler(rt.GetHTTPHandler()))
					break
				}
			}
		}

		if aviator != nil {
			aviator.SetRuntimeConfiguration(rtConf)
		}

		serverEntryPointsTCP.Switch(routers)
		serverEntryPointsUDP.Switch(udpRouters)
	}
}

// initACMEProvider creates an acme provider from the ACME part of globalConfiguration.
func initACMEProvider(c *static.Configuration, providerAggregator *aggregator.ProviderAggregator, tlsManager *traefiktls.Manager) []*acme.Provider {
	challengeStore := acme.NewLocalChallengeStore()
	localStores := map[string]*acme.LocalStore{}

	var resolvers []*acme.Provider
	for name, resolver := range c.CertificatesResolvers {
		if resolver.ACME != nil {
			if localStores[resolver.ACME.Storage] == nil {
				localStores[resolver.ACME.Storage] = acme.NewLocalStore(resolver.ACME.Storage)
			}

			p := &acme.Provider{
				Configuration:  resolver.ACME,
				Store:          localStores[resolver.ACME.Storage],
				ChallengeStore: challengeStore,
				ResolverName:   name,
			}

			if err := providerAggregator.AddProvider(p); err != nil {
				log.WithoutContext().Errorf("The ACME resolver %q is skipped from the resolvers list because: %v", name, err)
				continue
			}

			p.SetTLSManager(tlsManager)

			if p.TLSChallenge != nil {
				tlsManager.TLSAlpnGetter = p.GetTLSALPNCertificate
			}

			p.SetConfigListenerChan(make(chan dynamic.Configuration))

			resolvers = append(resolvers, p)
		}
	}
	return resolvers
}

func registerMetricClients(metricsConfig *types.Metrics) []metrics.Registry {
	if metricsConfig == nil {
		return nil
	}

	var registries []metrics.Registry

	if metricsConfig.Prometheus != nil {
		ctx := log.With(context.Background(), log.Str(log.MetricsProviderName, "prometheus"))
		prometheusRegister := metrics.RegisterPrometheus(ctx, metricsConfig.Prometheus)
		if prometheusRegister != nil {
			registries = append(registries, prometheusRegister)
			log.FromContext(ctx).Debug("Configured Prometheus metrics")
		}
	}

	if metricsConfig.Datadog != nil {
		ctx := log.With(context.Background(), log.Str(log.MetricsProviderName, "datadog"))
		registries = append(registries, metrics.RegisterDatadog(ctx, metricsConfig.Datadog))
		log.FromContext(ctx).Debugf("Configured Datadog metrics: pushing to %s once every %s",
			metricsConfig.Datadog.Address, metricsConfig.Datadog.PushInterval)
	}

	if metricsConfig.StatsD != nil {
		ctx := log.With(context.Background(), log.Str(log.MetricsProviderName, "statsd"))
		registries = append(registries, metrics.RegisterStatsd(ctx, metricsConfig.StatsD))
		log.FromContext(ctx).Debugf("Configured StatsD metrics: pushing to %s once every %s",
			metricsConfig.StatsD.Address, metricsConfig.StatsD.PushInterval)
	}

	if metricsConfig.InfluxDB != nil {
		ctx := log.With(context.Background(), log.Str(log.MetricsProviderName, "influxdb"))
		registries = append(registries, metrics.RegisterInfluxDB(ctx, metricsConfig.InfluxDB))
		log.FromContext(ctx).Debugf("Configured InfluxDB metrics: pushing to %s once every %s",
			metricsConfig.InfluxDB.Address, metricsConfig.InfluxDB.PushInterval)
	}

	return registries
}

func setupAccessLog(conf *types.AccessLog) *accesslog.Handler {
	if conf == nil {
		return nil
	}

	accessLoggerMiddleware, err := accesslog.NewHandler(conf)
	if err != nil {
		log.WithoutContext().Warnf("Unable to create access logger : %v", err)
		return nil
	}

	return accessLoggerMiddleware
}

func configureLogging(staticConfiguration *static.Configuration) {
	// configure default log flags
	stdlog.SetFlags(stdlog.Lshortfile | stdlog.LstdFlags)

	// configure log level
	// an explicitly defined log level always has precedence. if none is
	// given and debug mode is disabled, the default is ERROR, and DEBUG
	// otherwise.
	levelStr := "error"
	if staticConfiguration.Log != nil && staticConfiguration.Log.Level != "" {
		levelStr = strings.ToLower(staticConfiguration.Log.Level)
	}

	level, err := logrus.ParseLevel(levelStr)
	if err != nil {
		log.WithoutContext().Errorf("Error getting level: %v", err)
	}
	log.SetLevel(level)

	var logFile string
	if staticConfiguration.Log != nil && len(staticConfiguration.Log.FilePath) > 0 {
		logFile = staticConfiguration.Log.FilePath
	}

	// configure log format
	var formatter logrus.Formatter
	if staticConfiguration.Log != nil && staticConfiguration.Log.Format == "json" {
		formatter = &logrus.JSONFormatter{}
	} else {
		disableColors := len(logFile) > 0
		formatter = &logrus.TextFormatter{DisableColors: disableColors, FullTimestamp: true, DisableSorting: true}
	}
	log.SetFormatter(formatter)

	if len(logFile) > 0 {
		dir := filepath.Dir(logFile)

		if err := os.MkdirAll(dir, 0o755); err != nil {
			log.WithoutContext().Errorf("Failed to create log path %s: %s", dir, err)
		}

		err = log.OpenFile(logFile)
		logrus.RegisterExitHandler(func() {
			if err := log.CloseFile(); err != nil {
				log.WithoutContext().Errorf("Error while closing log: %v", err)
			}
		})
		if err != nil {
			log.WithoutContext().Errorf("Error while opening log file %s: %v", logFile, err)
		}
	}
}

func checkNewVersion() {
	ticker := time.Tick(24 * time.Hour)
	safe.Go(func() {
		for time.Sleep(10 * time.Minute); ; <-ticker {
			version.CheckNewVersion()
		}
	})
}

func stats(staticConfiguration *static.Configuration) {
	logger := log.WithoutContext()

	if staticConfiguration.Global.SendAnonymousUsage {
		logger.Info(`Stats collection is enabled.`)
		logger.Info(`Many thanks for contributing to Traefik's improvement by allowing us to receive anonymous information from your configuration.`)
		logger.Info(`Help us improve Traefik by leaving this feature on :)`)
		logger.Info(`More details on: https://doc.traefik.io/traefik/contributing/data-collection/`)
		collect(staticConfiguration)
	} else {
		logger.Info(`
Stats collection is disabled.
Help us improve Traefik by turning this feature on :)
More details on: https://doc.traefik.io/traefik/contributing/data-collection/
`)
	}
}

func collect(staticConfiguration *static.Configuration) {
	ticker := time.Tick(24 * time.Hour)
	safe.Go(func() {
		for time.Sleep(10 * time.Minute); ; <-ticker {
			if err := collector.Collect(staticConfiguration); err != nil {
				log.WithoutContext().Debug(err)
			}
		}
	})
}
