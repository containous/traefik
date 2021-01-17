package docker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/docker/cli/cli/connhelper"
	dockertypes "github.com/docker/docker/api/types"
	dockercontainertypes "github.com/docker/docker/api/types/container"
	eventtypes "github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	swarmtypes "github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/api/types/versions"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/docker/go-connections/sockets"
	ptypes "github.com/traefik/paerser/types"
	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	"github.com/traefik/traefik/v2/pkg/job"
	"github.com/traefik/traefik/v2/pkg/log"
	"github.com/traefik/traefik/v2/pkg/provider"
	"github.com/traefik/traefik/v2/pkg/safe"
	"github.com/traefik/traefik/v2/pkg/types"
	"github.com/traefik/traefik/v2/pkg/version"
)

const (
	// DockerAPIVersion is a constant holding the version of the Provider API traefik will use.
	DockerAPIVersion = "1.24"

	// SwarmAPIVersion is a constant holding the version of the Provider API traefik will use.
	SwarmAPIVersion = "1.24"
)

// DefaultTemplateRule The default template for the default rule.
const DefaultTemplateRule = "Host(`{{ normalize .Name }}`)"

var _ provider.Provider = (*Provider)(nil)

// Provider holds configurations of the provider.
type Provider struct {
	Constraints             string           `description:"Constraints is an expression that Traefik matches against the container's labels to determine whether to create any route for that container." json:"constraints,omitempty" toml:"constraints,omitempty" yaml:"constraints,omitempty" export:"true"`
	Watch                   bool             `description:"Watch Docker Swarm events." json:"watch,omitempty" toml:"watch,omitempty" yaml:"watch,omitempty" export:"true"`
	Endpoint                string           `description:"Docker server endpoint. Can be a tcp or a unix socket endpoint." json:"endpoint,omitempty" toml:"endpoint,omitempty" yaml:"endpoint,omitempty"`
	DefaultRule             string           `description:"Default rule." json:"defaultRule,omitempty" toml:"defaultRule,omitempty" yaml:"defaultRule,omitempty"`
	TLS                     *types.ClientTLS `description:"Enable Docker TLS support." json:"tls,omitempty" toml:"tls,omitempty" yaml:"tls,omitempty" export:"true"`
	ExposedByDefault        bool             `description:"Expose containers by default." json:"exposedByDefault,omitempty" toml:"exposedByDefault,omitempty" yaml:"exposedByDefault,omitempty" export:"true"`
	UseBindPortIP           bool             `description:"Use the ip address from the bound port, rather than from the inner network." json:"useBindPortIP,omitempty" toml:"useBindPortIP,omitempty" yaml:"useBindPortIP,omitempty" export:"true"`
	SwarmMode               bool             `description:"Use Docker on Swarm Mode." json:"swarmMode,omitempty" toml:"swarmMode,omitempty" yaml:"swarmMode,omitempty" export:"true"`
	Network                 string           `description:"Default Docker network used." json:"network,omitempty" toml:"network,omitempty" yaml:"network,omitempty" export:"true"`
	SwarmModeRefreshSeconds ptypes.Duration  `description:"Polling interval for swarm mode." json:"swarmModeRefreshSeconds,omitempty" toml:"swarmModeRefreshSeconds,omitempty" yaml:"swarmModeRefreshSeconds,omitempty" export:"true"`
	defaultRuleTpl          *template.Template
}

// SetDefaults sets the default values.
func (p *Provider) SetDefaults() {
	p.Watch = true
	p.ExposedByDefault = true
	p.Endpoint = "unix:///var/run/docker.sock"
	p.SwarmMode = false
	p.SwarmModeRefreshSeconds = ptypes.Duration(15 * time.Second)
	p.DefaultRule = DefaultTemplateRule
}

// Init the provider.
func (p *Provider) Init() error {
	defaultRuleTpl, err := provider.MakeDefaultRuleTemplate(p.DefaultRule, nil)
	if err != nil {
		return fmt.Errorf("error while parsing default rule: %w", err)
	}

	p.defaultRuleTpl = defaultRuleTpl
	return nil
}

// dockerData holds the need data to the provider.
type dockerData struct {
	ID              string
	ServiceName     string
	Name            string
	Labels          map[string]string // List of labels set to container or service
	NetworkSettings networkSettings
	Health          string
	Node            *dockertypes.ContainerNode
	ExtraConf       configuration
}

// NetworkSettings holds the networks data to the provider.
type networkSettings struct {
	NetworkMode dockercontainertypes.NetworkMode
	Ports       nat.PortMap
	Networks    map[string]*networkData
}

// Network holds the network data to the provider.
type networkData struct {
	Name     string
	Addr     string
	Port     int
	Protocol string
	ID       string
}

func (p *Provider) createClient() (client.APIClient, error) {
	opts, err := p.getClientOpts()
	if err != nil {
		return nil, err
	}

	httpHeaders := map[string]string{
		"User-Agent": "Traefik " + version.Version,
	}
	opts = append(opts, client.WithHTTPHeaders(httpHeaders))

	apiVersion := DockerAPIVersion
	if p.SwarmMode {
		apiVersion = SwarmAPIVersion
	}
	opts = append(opts, client.WithVersion(apiVersion))

	return client.NewClientWithOpts(opts...)
}

func (p *Provider) getClientOpts() ([]client.Opt, error) {
	helper, err := connhelper.GetConnectionHelper(p.Endpoint)
	if err != nil {
		return nil, err
	}

	// SSH
	if helper != nil {
		// https://github.com/docker/cli/blob/ebca1413117a3fcb81c89d6be226dcec74e5289f/cli/context/docker/load.go#L112-L123

		httpClient := &http.Client{
			Transport: &http.Transport{
				DialContext: helper.Dialer,
			},
		}

		return []client.Opt{
			client.WithHTTPClient(httpClient),
			client.WithHost(helper.Host), // To avoid 400 Bad Request: malformed Host header daemon error
			client.WithDialContext(helper.Dialer),
		}, nil
	}

	opts := []client.Opt{
		client.WithHost(p.Endpoint),
	}

	if p.TLS != nil {
		ctx := log.With(context.Background(), log.Str(log.ProviderName, "docker"))

		conf, err := p.TLS.CreateTLSConfig(ctx)
		if err != nil {
			return nil, err
		}

		hostURL, err := client.ParseHostURL(p.Endpoint)
		if err != nil {
			return nil, err
		}

		tr := &http.Transport{
			TLSClientConfig: conf,
		}

		if err := sockets.ConfigureTransport(tr, hostURL.Scheme, hostURL.Host); err != nil {
			return nil, err
		}

		opts = append(opts, client.WithHTTPClient(&http.Client{Transport: tr}))
	}

	return opts, nil
}

// Provide allows the docker provider to provide configurations to traefik using the given configuration channel.
func (p *Provider) Provide(configurationChan chan<- dynamic.Message, pool *safe.Pool) error {
	pool.GoCtx(func(routineCtx context.Context) {
		ctxLog := log.With(routineCtx, log.Str(log.ProviderName, "docker"))
		logger := log.FromContext(ctxLog)

		operation := func() error {
			var err error
			ctx, cancel := context.WithCancel(ctxLog)
			defer cancel()

			ctx = log.With(ctx, log.Str(log.ProviderName, "docker"))

			dockerClient, err := p.createClient()
			if err != nil {
				logger.Errorf("Failed to create a client for docker, error: %s", err)
				return err
			}

			serverVersion, err := dockerClient.ServerVersion(ctx)
			if err != nil {
				logger.Errorf("Failed to retrieve information of the docker client and server host: %s", err)
				return err
			}
			logger.Debugf("Provider connection established with docker %s (API %s)", serverVersion.Version, serverVersion.APIVersion)
			var dockerDataList []dockerData
			if p.SwarmMode {
				dockerDataList, err = p.listServices(ctx, dockerClient)
				if err != nil {
					logger.Errorf("Failed to list services for docker swarm mode, error %s", err)
					return err
				}
			} else {
				dockerDataList, err = p.listContainers(ctx, dockerClient)
				if err != nil {
					logger.Errorf("Failed to list containers for docker, error %s", err)
					return err
				}
			}

			configuration := p.buildConfiguration(ctxLog, dockerDataList)
			configurationChan <- dynamic.Message{
				ProviderName:  "docker",
				Configuration: configuration,
			}
			if p.Watch {
				if p.SwarmMode {
					errChan := make(chan error)

					// TODO: This need to be change. Linked to Swarm events docker/docker#23827
					ticker := time.NewTicker(time.Duration(p.SwarmModeRefreshSeconds))

					pool.GoCtx(func(ctx context.Context) {
						ctx = log.With(ctx, log.Str(log.ProviderName, "docker"))
						logger := log.FromContext(ctx)

						defer close(errChan)
						for {
							select {
							case <-ticker.C:
								services, err := p.listServices(ctx, dockerClient)
								if err != nil {
									logger.Errorf("Failed to list services for docker, error %s", err)
									errChan <- err
									return
								}

								configuration := p.buildConfiguration(ctx, services)
								if configuration != nil {
									configurationChan <- dynamic.Message{
										ProviderName:  "docker",
										Configuration: configuration,
									}
								}

							case <-ctx.Done():
								ticker.Stop()
								return
							}
						}
					})
					if err, ok := <-errChan; ok {
						return err
					}
					// channel closed
				} else {
					f := filters.NewArgs()
					f.Add("type", "container")
					options := dockertypes.EventsOptions{
						Filters: f,
					}

					startStopHandle := func(m eventtypes.Message) {
						logger.Debugf("Provider event received %+v", m)
						containers, err := p.listContainers(ctx, dockerClient)
						if err != nil {
							logger.Errorf("Failed to list containers for docker, error %s", err)
							// Call cancel to get out of the monitor
							return
						}

						configuration := p.buildConfiguration(ctx, containers)
						if configuration != nil {
							message := dynamic.Message{
								ProviderName:  "docker",
								Configuration: configuration,
							}
							select {
							case configurationChan <- message:
							case <-ctx.Done():
							}
						}
					}

					eventsc, errc := dockerClient.Events(ctx, options)
					for {
						select {
						case event := <-eventsc:
							if event.Action == "start" ||
								event.Action == "die" ||
								strings.HasPrefix(event.Action, "health_status") {
								startStopHandle(event)
							}
						case err := <-errc:
							if errors.Is(err, io.EOF) {
								logger.Debug("Provider event stream closed")
							}
							return err
						case <-ctx.Done():
							return nil
						}
					}
				}
			}
			return nil
		}

		notify := func(err error, time time.Duration) {
			logger.Errorf("Provider connection error %+v, retrying in %s", err, time)
			configurationChan <- dynamic.Message{
				ProviderName:       "docker",
				ErrorLoadingConfig: true,
			}
		}
		err := backoff.RetryNotify(safe.OperationWithRecover(operation), backoff.WithContext(job.NewBackOff(backoff.NewExponentialBackOff()), ctxLog), notify)
		if err != nil {
			logger.Errorf("Cannot connect to docker server %+v", err)
			configurationChan <- dynamic.Message{
				ProviderName:       "docker",
				ErrorLoadingConfig: true,
			}
		}
	})

	return nil
}

func (p *Provider) listContainers(ctx context.Context, dockerClient client.ContainerAPIClient) ([]dockerData, error) {
	containerList, err := dockerClient.ContainerList(ctx, dockertypes.ContainerListOptions{})
	if err != nil {
		return nil, err
	}

	var inspectedContainers []dockerData
	// get inspect containers
	for _, container := range containerList {
		dData := inspectContainers(ctx, dockerClient, container.ID)
		if len(dData.Name) == 0 {
			continue
		}

		extraConf, err := p.getConfiguration(dData)
		if err != nil {
			log.FromContext(ctx).Errorf("Skip container %s: %v", getServiceName(dData), err)
			continue
		}
		dData.ExtraConf = extraConf

		inspectedContainers = append(inspectedContainers, dData)
	}
	return inspectedContainers, nil
}

func inspectContainers(ctx context.Context, dockerClient client.ContainerAPIClient, containerID string) dockerData {
	containerInspected, err := dockerClient.ContainerInspect(ctx, containerID)
	if err != nil {
		log.FromContext(ctx).Warnf("Failed to inspect container %s, error: %s", containerID, err)
		return dockerData{}
	}

	// This condition is here to avoid to have empty IP https://github.com/traefik/traefik/issues/2459
	// We register only container which are running
	if containerInspected.ContainerJSONBase != nil && containerInspected.ContainerJSONBase.State != nil && containerInspected.ContainerJSONBase.State.Running {
		return parseContainer(containerInspected)
	}

	return dockerData{}
}

func parseContainer(container dockertypes.ContainerJSON) dockerData {
	dData := dockerData{
		NetworkSettings: networkSettings{},
	}

	if container.ContainerJSONBase != nil {
		dData.ID = container.ContainerJSONBase.ID
		dData.Name = container.ContainerJSONBase.Name
		dData.ServiceName = dData.Name // Default ServiceName to be the container's Name.
		dData.Node = container.ContainerJSONBase.Node

		if container.ContainerJSONBase.HostConfig != nil {
			dData.NetworkSettings.NetworkMode = container.ContainerJSONBase.HostConfig.NetworkMode
		}

		if container.State != nil && container.State.Health != nil {
			dData.Health = container.State.Health.Status
		}
	}

	if container.Config != nil && container.Config.Labels != nil {
		dData.Labels = container.Config.Labels
	}

	if container.NetworkSettings != nil {
		if container.NetworkSettings.Ports != nil {
			dData.NetworkSettings.Ports = container.NetworkSettings.Ports
		}
		if container.NetworkSettings.Networks != nil {
			dData.NetworkSettings.Networks = make(map[string]*networkData)
			for name, containerNetwork := range container.NetworkSettings.Networks {
				dData.NetworkSettings.Networks[name] = &networkData{
					ID:   containerNetwork.NetworkID,
					Name: name,
					Addr: containerNetwork.IPAddress,
				}
			}
		}
	}
	return dData
}

func (p *Provider) listServices(ctx context.Context, dockerClient client.APIClient) ([]dockerData, error) {
	logger := log.FromContext(ctx)

	serviceList, err := dockerClient.ServiceList(ctx, dockertypes.ServiceListOptions{})
	if err != nil {
		return nil, err
	}

	serverVersion, err := dockerClient.ServerVersion(ctx)
	if err != nil {
		return nil, err
	}

	networkListArgs := filters.NewArgs()
	// https://docs.docker.com/engine/api/v1.29/#tag/Network (Docker 17.06)
	if versions.GreaterThanOrEqualTo(serverVersion.APIVersion, "1.29") {
		networkListArgs.Add("scope", "swarm")
	} else {
		networkListArgs.Add("driver", "overlay")
	}

	networkList, err := dockerClient.NetworkList(ctx, dockertypes.NetworkListOptions{Filters: networkListArgs})
	if err != nil {
		logger.Debugf("Failed to network inspect on client for docker, error: %s", err)
		return nil, err
	}

	networkMap := make(map[string]*dockertypes.NetworkResource)
	for _, network := range networkList {
		networkToAdd := network
		networkMap[network.ID] = &networkToAdd
	}

	var dockerDataList []dockerData
	var dockerDataListTasks []dockerData

	for _, service := range serviceList {
		dData, err := p.parseService(ctx, service, networkMap)
		if err != nil {
			logger.Errorf("Skip container %s: %v", getServiceName(dData), err)
			continue
		}

		if dData.ExtraConf.Docker.LBSwarm {
			if len(dData.NetworkSettings.Networks) > 0 {
				dockerDataList = append(dockerDataList, dData)
			}
		} else {
			isGlobalSvc := service.Spec.Mode.Global != nil
			dockerDataListTasks, err = listTasks(ctx, dockerClient, service.ID, dData, networkMap, isGlobalSvc)
			if err != nil {
				logger.Warn(err)
			} else {
				dockerDataList = append(dockerDataList, dockerDataListTasks...)
			}
		}
	}
	return dockerDataList, err
}

func (p *Provider) parseService(ctx context.Context, service swarmtypes.Service, networkMap map[string]*dockertypes.NetworkResource) (dockerData, error) {
	logger := log.FromContext(ctx)

	dData := dockerData{
		ID:              service.ID,
		ServiceName:     service.Spec.Annotations.Name,
		Name:            service.Spec.Annotations.Name,
		Labels:          service.Spec.Annotations.Labels,
		NetworkSettings: networkSettings{},
	}

	extraConf, err := p.getConfiguration(dData)
	if err != nil {
		return dockerData{}, err
	}
	dData.ExtraConf = extraConf

	if service.Spec.EndpointSpec != nil {
		if service.Spec.EndpointSpec.Mode == swarmtypes.ResolutionModeDNSRR {
			if dData.ExtraConf.Docker.LBSwarm {
				logger.Warnf("Ignored %s endpoint-mode not supported, service name: %s. Fallback to Traefik load balancing", swarmtypes.ResolutionModeDNSRR, service.Spec.Annotations.Name)
			}
		} else if service.Spec.EndpointSpec.Mode == swarmtypes.ResolutionModeVIP {
			dData.NetworkSettings.Networks = make(map[string]*networkData)
			for _, virtualIP := range service.Endpoint.VirtualIPs {
				networkService := networkMap[virtualIP.NetworkID]
				if networkService != nil {
					if len(virtualIP.Addr) > 0 {
						ip, _, _ := net.ParseCIDR(virtualIP.Addr)
						network := &networkData{
							Name: networkService.Name,
							ID:   virtualIP.NetworkID,
							Addr: ip.String(),
						}
						dData.NetworkSettings.Networks[network.Name] = network
					} else {
						logger.Debugf("No virtual IPs found in network %s", virtualIP.NetworkID)
					}
				} else {
					logger.Debugf("Network not found, id: %s", virtualIP.NetworkID)
				}
			}
		}
	}
	return dData, nil
}

func listTasks(ctx context.Context, dockerClient client.APIClient, serviceID string,
	serviceDockerData dockerData, networkMap map[string]*dockertypes.NetworkResource, isGlobalSvc bool) ([]dockerData, error) {
	serviceIDFilter := filters.NewArgs()
	serviceIDFilter.Add("service", serviceID)
	serviceIDFilter.Add("desired-state", "running")

	taskList, err := dockerClient.TaskList(ctx, dockertypes.TaskListOptions{Filters: serviceIDFilter})
	if err != nil {
		return nil, err
	}

	var dockerDataList []dockerData
	for _, task := range taskList {
		if task.Status.State != swarmtypes.TaskStateRunning {
			continue
		}
		dData := parseTasks(ctx, task, serviceDockerData, networkMap, isGlobalSvc)
		if len(dData.NetworkSettings.Networks) > 0 {
			dockerDataList = append(dockerDataList, dData)
		}
	}
	return dockerDataList, err
}

func parseTasks(ctx context.Context, task swarmtypes.Task, serviceDockerData dockerData,
	networkMap map[string]*dockertypes.NetworkResource, isGlobalSvc bool) dockerData {
	dData := dockerData{
		ID:              task.ID,
		ServiceName:     serviceDockerData.Name,
		Name:            serviceDockerData.Name + "." + strconv.Itoa(task.Slot),
		Labels:          serviceDockerData.Labels,
		ExtraConf:       serviceDockerData.ExtraConf,
		NetworkSettings: networkSettings{},
	}

	if isGlobalSvc {
		dData.Name = serviceDockerData.Name + "." + task.ID
	}

	if task.NetworksAttachments != nil {
		dData.NetworkSettings.Networks = make(map[string]*networkData)
		for _, virtualIP := range task.NetworksAttachments {
			if networkService, present := networkMap[virtualIP.Network.ID]; present {
				if len(virtualIP.Addresses) > 0 {
					// Not sure about this next loop - when would a task have multiple IP's for the same network?
					for _, addr := range virtualIP.Addresses {
						ip, _, _ := net.ParseCIDR(addr)
						network := &networkData{
							ID:   virtualIP.Network.ID,
							Name: networkService.Name,
							Addr: ip.String(),
						}
						dData.NetworkSettings.Networks[network.Name] = network
					}
				} else {
					log.FromContext(ctx).Debugf("No IP addresses found for network %s", virtualIP.Network.ID)
				}
			}
		}
	}
	return dData
}
