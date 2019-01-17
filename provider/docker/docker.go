package docker

import (
	"context"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/cenk/backoff"
	"github.com/containous/traefik/config"
	"github.com/containous/traefik/job"
	"github.com/containous/traefik/log"
	"github.com/containous/traefik/provider"
	"github.com/containous/traefik/safe"
	"github.com/containous/traefik/types"
	"github.com/containous/traefik/version"
	dockertypes "github.com/docker/docker/api/types"
	dockercontainertypes "github.com/docker/docker/api/types/container"
	eventtypes "github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	swarmtypes "github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/api/types/versions"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/docker/go-connections/sockets"
)

const (
	// SwarmAPIVersion is a constant holding the version of the Provider API traefik will use
	SwarmAPIVersion = "1.24"
)

var _ provider.Provider = (*Provider)(nil)

// Provider holds configurations of the provider.
type Provider struct {
	provider.BaseProvider   `mapstructure:",squash" export:"true"`
	Endpoint                string           `description:"Docker server endpoint. Can be a tcp or a unix socket endpoint"`
	Domain                  string           `description:"Default domain used"`
	TLS                     *types.ClientTLS `description:"Enable Docker TLS support" export:"true"`
	ExposedByDefault        bool             `description:"Expose containers by default" export:"true"`
	UseBindPortIP           bool             `description:"Use the ip address from the bound port, rather than from the inner network" export:"true"`
	SwarmMode               bool             `description:"Use Docker on Swarm Mode" export:"true"`
	Network                 string           `description:"Default Docker network used" export:"true"`
	SwarmModeRefreshSeconds int              `description:"Polling interval for swarm mode (in seconds)" export:"true"`
}

// Init the provider
func (p *Provider) Init() error {
	return p.BaseProvider.Init()
}

// dockerData holds the need data to the Provider p
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

// NetworkSettings holds the networks data to the Provider p
type networkSettings struct {
	NetworkMode dockercontainertypes.NetworkMode
	Ports       nat.PortMap
	Networks    map[string]*networkData
}

// Network holds the network data to the Provider p
type networkData struct {
	Name     string
	Addr     string
	Port     int
	Protocol string
	ID       string
}

func (p *Provider) createClient() (client.APIClient, error) {
	var httpClient *http.Client

	if p.TLS != nil {
		ctx := log.With(context.Background(), log.Str(log.ProviderName, "docker"))
		conf, err := p.TLS.CreateTLSConfig(ctx)
		if err != nil {
			return nil, err
		}
		tr := &http.Transport{
			TLSClientConfig: conf,
		}

		hostURL, err := client.ParseHostURL(p.Endpoint)
		if err != nil {
			return nil, err
		}
		if err := sockets.ConfigureTransport(tr, hostURL.Scheme, hostURL.Host); err != nil {
			return nil, err
		}

		httpClient = &http.Client{
			Transport: tr,
		}
	}

	httpHeaders := map[string]string{
		"User-Agent": "Traefik " + version.Version,
	}

	var apiVersion string
	if p.SwarmMode {
		apiVersion = SwarmAPIVersion
	} else {
		apiVersion = DockerAPIVersion
	}

	return client.NewClient(p.Endpoint, apiVersion, httpClient, httpHeaders)
}

// Provide allows the docker provider to provide configurations to traefik
// using the given configuration channel.
func (p *Provider) Provide(configurationChan chan<- config.Message, pool *safe.Pool) error {
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
			configurationChan <- config.Message{
				ProviderName:  "docker",
				Configuration: configuration,
			}
			if p.Watch {
				if p.SwarmMode {
					errChan := make(chan error)
					// TODO: This need to be change. Linked to Swarm events docker/docker#23827
					ticker := time.NewTicker(time.Second * time.Duration(p.SwarmModeRefreshSeconds))
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
									configurationChan <- config.Message{
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
							message := config.Message{
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
							if err == io.EOF {
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
		}
		err := backoff.RetryNotify(safe.OperationWithRecover(operation), backoff.WithContext(job.NewBackOff(backoff.NewExponentialBackOff()), ctxLog), notify)
		if err != nil {
			logger.Errorf("Cannot connect to docker server %+v", err)
		}
	})

	return nil
}

func (p *Provider) listContainers(ctx context.Context, dockerClient client.ContainerAPIClient) ([]dockerData, error) {
	containerList, err := dockerClient.ContainerList(ctx, dockertypes.ContainerListOptions{})
	if err != nil {
		return nil, err
	}

	var containersInspected []dockerData
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

		containersInspected = append(containersInspected, dData)
	}
	return containersInspected, nil
}

func inspectContainers(ctx context.Context, dockerClient client.ContainerAPIClient, containerID string) dockerData {
	dData := dockerData{}
	containerInspected, err := dockerClient.ContainerInspect(ctx, containerID)
	if err != nil {
		log.FromContext(ctx).Warnf("Failed to inspect container %s, error: %s", containerID, err)
	} else {
		// This condition is here to avoid to have empty IP https://github.com/containous/traefik/issues/2459
		// We register only container which are running
		if containerInspected.ContainerJSONBase != nil && containerInspected.ContainerJSONBase.State != nil && containerInspected.ContainerJSONBase.State.Running {
			dData = parseContainer(containerInspected)
		}
	}
	return dData
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
