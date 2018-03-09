package main

import (
	"context"
	"encoding/json"
	fmtlog "log"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/cenk/backoff"
	"github.com/containous/flaeg"
	"github.com/containous/staert"
	"github.com/containous/traefik/cmd"
	"github.com/containous/traefik/cmd/bug"
	"github.com/containous/traefik/cmd/healthcheck"
	"github.com/containous/traefik/cmd/storeconfig"
	cmdVersion "github.com/containous/traefik/cmd/version"
	"github.com/containous/traefik/collector"
	"github.com/containous/traefik/configuration"
	"github.com/containous/traefik/job"
	"github.com/containous/traefik/log"
	"github.com/containous/traefik/provider/acme"
	"github.com/containous/traefik/provider/ecs"
	"github.com/containous/traefik/provider/kubernetes"
	"github.com/containous/traefik/safe"
	"github.com/containous/traefik/server"
	"github.com/containous/traefik/server/uuid"
	traefiktls "github.com/containous/traefik/tls"
	"github.com/containous/traefik/types"
	"github.com/containous/traefik/version"
	"github.com/coreos/go-systemd/daemon"
	"github.com/ogier/pflag"
	"github.com/sirupsen/logrus"
)

func main() {
	// traefik config inits
	traefikConfiguration := cmd.NewTraefikConfiguration()
	traefikPointersConfiguration := cmd.NewTraefikDefaultPointersConfiguration()

	// traefik Command init
	traefikCmd := &flaeg.Command{
		Name: "traefik",
		Description: `traefik is a modern HTTP reverse proxy and load balancer made to deploy microservices with ease.
Complete documentation is available at https://traefik.io`,
		Config:                traefikConfiguration,
		DefaultPointersConfig: traefikPointersConfiguration,
		Run: func() error {
			runCmd(&traefikConfiguration.GlobalConfiguration, traefikConfiguration.ConfigFile)
			return nil
		},
	}

	// storeconfig Command init
	storeConfigCmd := storeconfig.NewCmd(traefikConfiguration, traefikPointersConfiguration)

	// init flaeg source
	f := flaeg.New(traefikCmd, os.Args[1:])
	// add custom parsers
	f.AddParser(reflect.TypeOf(configuration.EntryPoints{}), &configuration.EntryPoints{})
	f.AddParser(reflect.TypeOf(configuration.DefaultEntryPoints{}), &configuration.DefaultEntryPoints{})
	f.AddParser(reflect.TypeOf(traefiktls.RootCAs{}), &traefiktls.RootCAs{})
	f.AddParser(reflect.TypeOf(types.Constraints{}), &types.Constraints{})
	f.AddParser(reflect.TypeOf(kubernetes.Namespaces{}), &kubernetes.Namespaces{})
	f.AddParser(reflect.TypeOf(ecs.Clusters{}), &ecs.Clusters{})
	f.AddParser(reflect.TypeOf([]types.Domain{}), &types.Domains{})
	f.AddParser(reflect.TypeOf(types.Buckets{}), &types.Buckets{})
	f.AddParser(reflect.TypeOf(types.StatusCodes{}), &types.StatusCodes{})
	f.AddParser(reflect.TypeOf(types.FieldsNames{}), &types.FieldsNames{})
	f.AddParser(reflect.TypeOf(types.FieldsHeadersNames{}), &types.FieldsHeadersNames{})

	// add commands
	f.AddCommand(cmdVersion.NewCmd())
	f.AddCommand(bug.NewCmd(traefikConfiguration, traefikPointersConfiguration))
	f.AddCommand(storeConfigCmd)
	f.AddCommand(healthcheck.NewCmd(traefikConfiguration, traefikPointersConfiguration))

	usedCmd, err := f.GetCommand()
	if err != nil {
		fmtlog.Println(err)
		os.Exit(1)
	}

	if _, err := f.Parse(usedCmd); err != nil {
		if err == pflag.ErrHelp {
			os.Exit(0)
		}
		fmtlog.Printf("Error parsing command: %s\n", err)
		os.Exit(1)
	}

	// staert init
	s := staert.NewStaert(traefikCmd)
	// init TOML source
	toml := staert.NewTomlSource("traefik", []string{traefikConfiguration.ConfigFile, "/etc/traefik/", "$HOME/.traefik/", "."})

	// add sources to staert
	s.AddSource(toml)
	s.AddSource(f)
	if _, err := s.LoadConfig(); err != nil {
		fmtlog.Printf("Error reading TOML config file %s : %s\n", toml.ConfigFileUsed(), err)
		os.Exit(1)
	}

	traefikConfiguration.ConfigFile = toml.ConfigFileUsed()

	kv, err := storeconfig.CreateKvSource(traefikConfiguration)
	if err != nil {
		fmtlog.Printf("Error creating kv store: %s\n", err)
		os.Exit(1)
	}
	storeConfigCmd.Run = storeconfig.Run(kv, traefikConfiguration)

	// if a KV Store is enable and no sub-command called in args
	if kv != nil && usedCmd == traefikCmd {
		if traefikConfiguration.Cluster == nil {
			traefikConfiguration.Cluster = &types.Cluster{Node: uuid.Get()}
		}
		if traefikConfiguration.Cluster.Store == nil {
			traefikConfiguration.Cluster.Store = &types.Store{Prefix: kv.Prefix, Store: kv.Store}
		}
		s.AddSource(kv)
		operation := func() error {
			_, err := s.LoadConfig()
			return err
		}
		notify := func(err error, time time.Duration) {
			log.Errorf("Load config error: %+v, retrying in %s", err, time)
		}
		err := backoff.RetryNotify(safe.OperationWithRecover(operation), job.NewBackOff(backoff.NewExponentialBackOff()), notify)
		if err != nil {
			fmtlog.Printf("Error loading configuration: %s\n", err)
			os.Exit(1)
		}
	}

	if err := s.Run(); err != nil {
		fmtlog.Printf("Error running traefik: %s\n", err)
		os.Exit(1)
	}

	os.Exit(0)
}

func runCmd(globalConfiguration *configuration.GlobalConfiguration, configFile string) {
	configureLogging(globalConfiguration)

	if len(configFile) > 0 {
		log.Infof("Using TOML configuration file %s", configFile)
	}

	http.DefaultTransport.(*http.Transport).Proxy = http.ProxyFromEnvironment

	globalConfiguration.SetEffectiveConfiguration(configFile)
	globalConfiguration.ValidateConfiguration()

	jsonConf, _ := json.Marshal(globalConfiguration)
	log.Infof("Traefik version %s built on %s", version.Version, version.BuildDate)

	if globalConfiguration.CheckNewVersion {
		checkNewVersion()
	}

	stats(globalConfiguration)

	log.Debugf("Global configuration loaded %s", string(jsonConf))
	if acme.IsEnabled() {
		store := acme.NewLocalStore(acme.Get().Storage)
		acme.Get().Store = &store
	}
	svr := server.NewServer(*globalConfiguration, configuration.NewProviderAggregator(globalConfiguration))
	if acme.IsEnabled() && acme.Get().OnHostRule {
		acme.Get().SetConfigListenerChan(make(chan types.Configuration))
		svr.AddListener(acme.Get().ListenConfiguration)
	}
	ctx := cmd.ContextWithSignal(context.Background())
	svr.StartWithContext(ctx)
	defer svr.Close()

	sent, err := daemon.SdNotify(false, "READY=1")
	if !sent && err != nil {
		log.Error("Fail to notify", err)
	}

	t, err := daemon.SdWatchdogEnabled(false)
	if err != nil {
		log.Error("Problem with watchdog", err)
	} else if t != 0 {
		// Send a ping each half time given
		t = t / 2
		log.Info("Watchdog activated with timer each ", t)
		safe.Go(func() {
			tick := time.Tick(t)
			for range tick {
				_, errHealthCheck := healthcheck.Do(*globalConfiguration)
				if globalConfiguration.Ping == nil || errHealthCheck == nil {
					if ok, _ := daemon.SdNotify(false, "WATCHDOG=1"); !ok {
						log.Error("Fail to tick watchdog")
					}
				} else {
					log.Error(errHealthCheck)
				}
			}
		})
	}

	svr.Wait()
	log.Info("Shutting down")
	logrus.Exit(0)
}

func configureLogging(globalConfiguration *configuration.GlobalConfiguration) {
	// configure default log flags
	fmtlog.SetFlags(fmtlog.Lshortfile | fmtlog.LstdFlags)

	if globalConfiguration.Debug {
		globalConfiguration.LogLevel = "DEBUG"
	}

	// configure log level
	level, err := logrus.ParseLevel(strings.ToLower(globalConfiguration.LogLevel))
	if err != nil {
		log.Error("Error getting level", err)
	}
	log.SetLevel(level)

	// configure log output file
	logFile := globalConfiguration.TraefikLogsFile
	if len(logFile) > 0 {
		log.Warn("top-level traefikLogsFile has been deprecated -- please use traefiklog.filepath")
	}
	if globalConfiguration.TraefikLog != nil && len(globalConfiguration.TraefikLog.FilePath) > 0 {
		logFile = globalConfiguration.TraefikLog.FilePath
	}

	// configure log format
	var formatter logrus.Formatter
	if globalConfiguration.TraefikLog != nil && globalConfiguration.TraefikLog.Format == "json" {
		formatter = &logrus.JSONFormatter{}
	} else {
		disableColors := len(logFile) > 0
		formatter = &logrus.TextFormatter{DisableColors: disableColors, FullTimestamp: true, DisableSorting: true}
	}
	log.SetFormatter(formatter)

	if len(logFile) > 0 {
		dir := filepath.Dir(logFile)

		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Errorf("Failed to create log path %s: %s", dir, err)
		}

		err = log.OpenFile(logFile)
		logrus.RegisterExitHandler(func() {
			if err := log.CloseFile(); err != nil {
				log.Error("Error closing log", err)
			}
		})
		if err != nil {
			log.Error("Error opening file", err)
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

func stats(globalConfiguration *configuration.GlobalConfiguration) {
	if globalConfiguration.SendAnonymousUsage {
		log.Info(`
Stats collection is enabled.
Many thanks for contributing to Traefik's improvement by allowing us to receive anonymous information from your configuration.
Help us improve Traefik by leaving this feature on :)
More details on: https://docs.traefik.io/basics/#collected-data
`)
		collect(globalConfiguration)
	} else {
		log.Info(`
Stats collection is disabled.
Help us improve Traefik by turning this feature on :)
More details on: https://docs.traefik.io/basics/#collected-data
`)
	}
}

func collect(globalConfiguration *configuration.GlobalConfiguration) {
	ticker := time.Tick(24 * time.Hour)
	safe.Go(func() {
		for time.Sleep(10 * time.Minute); ; <-ticker {
			if err := collector.Collect(globalConfiguration); err != nil {
				log.Debug(err)
			}
		}
	})
}
