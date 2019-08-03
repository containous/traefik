package rancher

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/containous/traefik/v2/pkg/config/dynamic"
	"github.com/containous/traefik/v2/pkg/config/label"
	"github.com/containous/traefik/v2/pkg/log"
	"github.com/containous/traefik/v2/pkg/provider"
	"github.com/containous/traefik/v2/pkg/provider/constraints"
)

func (p *Provider) buildConfiguration(ctx context.Context, services []rancherData) *dynamic.Configuration {
	configurations := make(map[string]*dynamic.Configuration)

	for _, service := range services {
		ctxService := log.With(ctx, log.Str("service", service.Name))

		if !p.keepService(ctx, service) {
			continue
		}

		logger := log.FromContext(ctxService)

		confFromLabel, err := label.DecodeConfiguration(service.Labels)
		if err != nil {
			logger.Error(err)
			continue
		}

		if len(confFromLabel.TCP.Routers) > 0 || len(confFromLabel.TCP.Services) > 0 {
			err := p.buildTCPServiceConfiguration(ctxService, service, confFromLabel.TCP)
			if err != nil {
				logger.Error(err)
				continue
			}
			provider.BuildTCPRouterConfiguration(ctxService, confFromLabel.TCP)
			if len(confFromLabel.HTTP.Routers) == 0 &&
				len(confFromLabel.HTTP.Middlewares) == 0 &&
				len(confFromLabel.HTTP.Services) == 0 {
				configurations[service.Name] = confFromLabel
				continue
			}
		}

		err = p.buildServiceConfiguration(ctx, service, confFromLabel.HTTP)
		if err != nil {
			logger.Error(err)
			continue
		}

		model := struct {
			Name   string
			Labels map[string]string
		}{
			Name:   service.Name,
			Labels: service.Labels,
		}

		provider.BuildRouterConfiguration(ctx, confFromLabel.HTTP, service.Name, p.defaultRuleTpl, model)

		configurations[service.Name] = confFromLabel
	}

	return provider.Merge(ctx, configurations)
}

func (p *Provider) buildTCPServiceConfiguration(ctx context.Context, service rancherData, configuration *dynamic.TCPConfiguration) error {
	serviceName := service.Name

	if len(configuration.Services) == 0 {
		configuration.Services = make(map[string]*dynamic.TCPService)
		lb := &dynamic.TCPLoadBalancerService{}
		configuration.Services[serviceName] = &dynamic.TCPService{
			LoadBalancer: lb,
		}
	}

	for _, confService := range configuration.Services {
		err := p.addServerTCP(ctx, service, confService.LoadBalancer)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *Provider) buildServiceConfiguration(ctx context.Context, service rancherData, configuration *dynamic.HTTPConfiguration) error {

	serviceName := service.Name

	if len(configuration.Services) == 0 {
		configuration.Services = make(map[string]*dynamic.Service)
		lb := &dynamic.LoadBalancerService{}
		lb.SetDefaults()
		configuration.Services[serviceName] = &dynamic.Service{
			LoadBalancer: lb,
		}
	}

	for _, confService := range configuration.Services {
		err := p.addServers(ctx, service, confService.LoadBalancer)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *Provider) keepService(ctx context.Context, service rancherData) bool {
	logger := log.FromContext(ctx)

	if !service.ExtraConf.Enable {
		logger.Debug("Filtering disabled service.")
		return false
	}

	matches, err := constraints.Match(service.Labels, p.Constraints)
	if err != nil {
		logger.Errorf("Error matching constraints expression: %v", err)
		return false
	}
	if !matches {
		logger.Debugf("Service pruned by constraint expression: %q", p.Constraints)
		return false
	}

	if p.EnableServiceHealthFilter {
		if service.Health != "" && service.Health != healthy && service.Health != updatingHealthy {
			logger.Debugf("Filtering service %s with healthState of %s \n", service.Name, service.Health)
			return false
		}
		if service.State != "" && service.State != active && service.State != updatingActive && service.State != upgraded && service.State != upgrading {
			logger.Debugf("Filtering service %s with state of %s \n", service.Name, service.State)
			return false
		}
	}

	return true
}

func (p *Provider) addServerTCP(ctx context.Context, service rancherData, loadBalancer *dynamic.TCPLoadBalancerService) error {
	log.FromContext(ctx).Debugf("Trying to add servers for service  %s \n", service.Name)

	serverPort := ""

	if loadBalancer != nil && len(loadBalancer.Servers) > 0 {
		serverPort = loadBalancer.Servers[0].Port
	}

	port := getServicePort(service)

	if len(loadBalancer.Servers) == 0 {
		server := dynamic.TCPServer{}

		loadBalancer.Servers = []dynamic.TCPServer{server}
	}

	if serverPort != "" {
		port = serverPort
		loadBalancer.Servers[0].Port = ""
	}

	if port == "" {
		return errors.New("port is missing")
	}

	var servers []dynamic.TCPServer
	for _, containerIP := range service.Containers {
		servers = append(servers, dynamic.TCPServer{
			Address: net.JoinHostPort(containerIP, port),
		})
	}

	loadBalancer.Servers = servers
	return nil

}

func (p *Provider) addServers(ctx context.Context, service rancherData, loadBalancer *dynamic.LoadBalancerService) error {
	log.FromContext(ctx).Debugf("Trying to add servers for service  %s \n", service.Name)

	serverPort := getLBServerPort(loadBalancer)
	port := getServicePort(service)

	if len(loadBalancer.Servers) == 0 {
		server := dynamic.Server{}
		server.SetDefaults()

		loadBalancer.Servers = []dynamic.Server{server}
	}

	if serverPort != "" {
		port = serverPort
		loadBalancer.Servers[0].Port = ""
	}

	if port == "" {
		return errors.New("port is missing")
	}

	var servers []dynamic.Server
	for _, containerIP := range service.Containers {
		servers = append(servers, dynamic.Server{
			URL: fmt.Sprintf("%s://%s", loadBalancer.Servers[0].Scheme, net.JoinHostPort(containerIP, port)),
		})
	}

	loadBalancer.Servers = servers
	return nil
}

func getLBServerPort(loadBalancer *dynamic.LoadBalancerService) string {
	if loadBalancer != nil && len(loadBalancer.Servers) > 0 {
		return loadBalancer.Servers[0].Port
	}
	return ""
}

func getServicePort(data rancherData) string {
	rawPort := strings.Split(data.Port, "/")[0]
	hostPort := strings.Split(rawPort, ":")

	if len(hostPort) >= 2 {
		return hostPort[1]
	}
	if len(hostPort) > 0 && hostPort[0] != "" {
		return hostPort[0]
	}
	return rawPort
}
