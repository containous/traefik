package docker

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"text/template"

	"github.com/BurntSushi/ty/fun"
	"github.com/containous/traefik/log"
	"github.com/containous/traefik/provider"
	"github.com/containous/traefik/provider/label"
	"github.com/containous/traefik/types"
	"github.com/docker/go-connections/nat"
)

func (p *Provider) buildConfigurationV2(containersInspected []dockerData) *types.Configuration {
	dockerFuncMap := template.FuncMap{
		"getSubDomain":     getSubDomain,
		"isBackendLBSwarm": isBackendLBSwarm,
		"getDomain":        getFuncStringLabel(label.TraefikDomain, p.Domain),

		// Backend functions
		"getIPAddress":      p.getIPAddress,
		"getServers":        p.getServers,
		"getMaxConn":        getMaxConn,
		"getHealthCheck":    getHealthCheck,
		"getBuffering":      getBuffering,
		"getCircuitBreaker": getCircuitBreaker,
		"getLoadBalancer":   getLoadBalancer,

		// Frontend functions
		"getBackend":              getBackendName, // FIXME keep ???
		"getPriority":             getFuncIntLabel(label.TraefikFrontendPriority, label.DefaultFrontendPriorityInt),
		"getPassHostHeader":       getFuncBoolLabel(label.TraefikFrontendPassHostHeader, label.DefaultPassHostHeaderBool),
		"getPassTLSCert":          getFuncBoolLabel(label.TraefikFrontendPassTLSCert, label.DefaultPassTLSCert),
		"getEntryPoints":          getFuncSliceStringLabel(label.TraefikFrontendEntryPoints),
		"getBasicAuth":            getFuncSliceStringLabel(label.TraefikFrontendAuthBasic),
		"getWhitelistSourceRange": getFuncSliceStringLabel(label.TraefikFrontendWhitelistSourceRange),
		"getFrontendRule":         p.getFrontendRule,
		"getBackendName":          getBackendName,
		"getRedirect":             getRedirect,
		"getErrorPages":           getErrorPages,
		"getRateLimit":            getRateLimit,
		"getHeaders":              getHeaders,
	}

	// filter containers
	filteredContainers := fun.Filter(p.containerFilter, containersInspected).([]dockerData)

	frontends := map[string][]dockerData{}
	servers := map[string][]dockerData{}

	serviceNames := make(map[string]struct{})

	for idx, container := range filteredContainers {
		roadProperties := label.ExtractTraefikLabels(container.Labels)
		for roadName, labels := range roadProperties {
			container.RoadLabels = labels
			container.RoadName = roadName

			// Frontends
			if _, exists := serviceNames[container.ServiceName+roadName]; !exists {
				frontendName := p.getFrontendName(container, idx)
				frontends[frontendName] = append(frontends[frontendName], container)
				if len(container.ServiceName+roadName) > 0 {
					serviceNames[container.ServiceName+roadName] = struct{}{}
				}
			}

			// Backends
			backendName := getBackendName(container)

			// Servers
			servers[backendName] = append(servers[backendName], container)
		}
	}

	templateObjects := struct {
		Containers []dockerData
		Frontends  map[string][]dockerData
		Servers    map[string][]dockerData
		Domain     string
	}{
		Containers: filteredContainers,
		Frontends:  frontends,
		Servers:    servers,
		Domain:     p.Domain,
	}

	configuration, err := p.GetConfiguration("templates/docker.tmpl", dockerFuncMap, templateObjects)
	if err != nil {
		log.Error(err)
	}

	return configuration
}

func (p *Provider) containerFilter(container dockerData) bool {
	if !label.IsEnabled(container.Labels, p.ExposedByDefault) {
		log.Debugf("Filtering disabled container %s", container.Name)
		return false
	}

	roadProperties := label.ExtractTraefikLabels(container.Labels)

	var errPort error
	for roadName, labels := range roadProperties {
		errPort = checkRoadPort(labels, roadName)

		if len(p.getFrontendRule(container)) == 0 {
			log.Debugf("Filtering container with empty frontend rule %s %s", container.Name, roadName)
			return false
		}
	}

	if len(container.NetworkSettings.Ports) == 0 && errPort != nil {
		log.Debugf("Filtering container without port, %s: %v", container.Name, errPort)
		return false
	}

	constraintTags := label.SplitAndTrimString(container.Labels[label.TraefikTags], ",")
	if ok, failingConstraint := p.MatchConstraints(constraintTags); !ok {
		if failingConstraint != nil {
			log.Debugf("Container %s pruned by %q constraint", container.Name, failingConstraint.String())
		}
		return false
	}

	if container.Health != "" && container.Health != "healthy" {
		log.Debugf("Filtering unhealthy or starting container %s", container.Name)
		return false
	}

	return true
}

func checkRoadPort(labels map[string]string, roadName string) error {
	if port, ok := labels[label.TraefikPort]; ok {
		_, err := strconv.Atoi(port)
		if err != nil {
			return fmt.Errorf("invalid port value %q for the road %q: %v", port, roadName, err)
		}
	} else {
		return fmt.Errorf("port label is missing, please use %s as default value or define port label for all roads ('traefik.<roadName>.port')", label.TraefikPort)
	}
	return nil
}

func (p *Provider) getFrontendName(container dockerData, idx int) string {
	var name string
	if len(container.RoadName) > 0 {
		name = getBackendName(container)
	} else {
		name = p.getFrontendRule(container) + "-" + strconv.Itoa(idx)
	}

	return provider.Normalize(name)
}

func (p *Provider) getFrontendRule(container dockerData) string {
	if value := label.GetStringValue(container.RoadLabels, label.TraefikFrontendRule, ""); len(value) != 0 {
		return value
	}

	if values, err := label.GetStringMultipleStrict(container.Labels, labelDockerComposeProject, labelDockerComposeService); err == nil {
		return "Host:" + getSubDomain(values[labelDockerComposeService]+"."+values[labelDockerComposeProject]) + "." + p.Domain
	}

	if len(p.Domain) > 0 {
		return "Host:" + getSubDomain(container.ServiceName) + "." + p.Domain
	}

	return ""
}

func (p Provider) getIPAddress(container dockerData) string {

	if value := label.GetStringValue(container.Labels, labelDockerNetwork, ""); value != "" {
		networkSettings := container.NetworkSettings
		if networkSettings.Networks != nil {
			network := networkSettings.Networks[value]
			if network != nil {
				return network.Addr
			}

			log.Warnf("Could not find network named '%s' for container '%s'! Maybe you're missing the project's prefix in the label? Defaulting to first available network.", value, container.Name)
		}
	}

	if container.NetworkSettings.NetworkMode.IsHost() {
		if container.Node != nil {
			if container.Node.IPAddress != "" {
				return container.Node.IPAddress
			}
		}
		return "127.0.0.1"
	}

	if container.NetworkSettings.NetworkMode.IsContainer() {
		dockerClient, err := p.createClient()
		if err != nil {
			log.Warnf("Unable to get IP address for container %s, error: %s", container.Name, err)
			return ""
		}

		connectedContainer := container.NetworkSettings.NetworkMode.ConnectedContainer()
		containerInspected, err := dockerClient.ContainerInspect(context.Background(), connectedContainer)
		if err != nil {
			log.Warnf("Unable to get IP address for container %s : Failed to inspect container ID %s, error: %s", container.Name, connectedContainer, err)
			return ""
		}
		return p.getIPAddress(parseContainer(containerInspected))
	}

	if p.UseBindPortIP {
		port := getPortV1(container)
		for netPort, portBindings := range container.NetworkSettings.Ports {
			if string(netPort) == port+"/TCP" || string(netPort) == port+"/UDP" {
				for _, p := range portBindings {
					return p.HostIP
				}
			}
		}
	}

	for _, network := range container.NetworkSettings.Networks {
		return network.Addr
	}
	return ""
}

// Escape beginning slash "/", convert all others to dash "-", and convert underscores "_" to dash "-"
func getSubDomain(name string) string {
	return strings.Replace(strings.Replace(strings.TrimPrefix(name, "/"), "/", "-", -1), "_", "-", -1)
}

func isBackendLBSwarm(container dockerData) bool {
	return label.GetBoolValue(container.Labels, labelBackendLoadBalancerSwarm, false)
}

func getRoadBackendName(container dockerData) string {
	if value := label.GetStringValue(container.RoadLabels, label.TraefikFrontendBackend, ""); len(value) > 0 {
		return provider.Normalize(container.ServiceName + "-" + value)
	}

	return provider.Normalize(container.ServiceName + "-" + getDefaultBackendName(container) + "-" + container.RoadName)
}

func getDefaultBackendName(container dockerData) string {
	if value := label.GetStringValue(container.RoadLabels, label.TraefikBackend, ""); len(value) != 0 {
		return provider.Normalize(value)
	}

	if values, err := label.GetStringMultipleStrict(container.Labels, labelDockerComposeProject, labelDockerComposeService); err == nil {
		return provider.Normalize(values[labelDockerComposeService] + "_" + values[labelDockerComposeProject])
	}

	return provider.Normalize(container.ServiceName)
}

func getBackendName(container dockerData) string {
	if len(container.RoadName) > 0 {
		return getRoadBackendName(container)
	}

	return getDefaultBackendName(container)
}

func getRedirect(labels map[string]string) *types.Redirect {
	permanent := label.GetBoolValue(labels, label.TraefikFrontendRedirectPermanent, false)

	if label.Has(labels, label.TraefikFrontendRedirectEntryPoint) {
		return &types.Redirect{
			EntryPoint: label.GetStringValue(labels, label.TraefikFrontendRedirectEntryPoint, ""),
			Permanent:  permanent,
		}
	}

	if label.Has(labels, label.TraefikFrontendRedirectRegex) &&
		label.Has(labels, label.TraefikFrontendRedirectReplacement) {
		return &types.Redirect{
			Regex:       label.GetStringValue(labels, label.TraefikFrontendRedirectRegex, ""),
			Replacement: label.GetStringValue(labels, label.TraefikFrontendRedirectReplacement, ""),
			Permanent:   permanent,
		}
	}

	return nil
}

func getErrorPages(labels map[string]string) map[string]*types.ErrorPage {
	prefix := label.Prefix + label.BaseFrontendErrorPage
	return label.ParseErrorPages(labels, prefix, label.RegexpFrontendErrorPage)
}

func getRateLimit(labels map[string]string) *types.RateLimit {
	extractorFunc := label.GetStringValue(labels, label.TraefikFrontendRateLimitExtractorFunc, "")
	if len(extractorFunc) == 0 {
		return nil
	}

	prefix := label.Prefix + label.BaseFrontendRateLimit
	limits := label.ParseRateSets(labels, prefix, label.RegexpFrontendRateLimit)

	return &types.RateLimit{
		ExtractorFunc: extractorFunc,
		RateSet:       limits,
	}
}

func getHeaders(labels map[string]string) *types.Headers {
	headers := &types.Headers{
		CustomRequestHeaders:    label.GetMapValue(labels, label.TraefikFrontendRequestHeaders),
		CustomResponseHeaders:   label.GetMapValue(labels, label.TraefikFrontendResponseHeaders),
		SSLProxyHeaders:         label.GetMapValue(labels, label.TraefikFrontendSSLProxyHeaders),
		AllowedHosts:            label.GetSliceStringValue(labels, label.TraefikFrontendAllowedHosts),
		HostsProxyHeaders:       label.GetSliceStringValue(labels, label.TraefikFrontendHostsProxyHeaders),
		STSSeconds:              label.GetInt64Value(labels, label.TraefikFrontendSTSSeconds, 0),
		SSLRedirect:             label.GetBoolValue(labels, label.TraefikFrontendSSLRedirect, false),
		SSLTemporaryRedirect:    label.GetBoolValue(labels, label.TraefikFrontendSSLTemporaryRedirect, false),
		STSIncludeSubdomains:    label.GetBoolValue(labels, label.TraefikFrontendSTSIncludeSubdomains, false),
		STSPreload:              label.GetBoolValue(labels, label.TraefikFrontendSTSPreload, false),
		ForceSTSHeader:          label.GetBoolValue(labels, label.TraefikFrontendForceSTSHeader, false),
		FrameDeny:               label.GetBoolValue(labels, label.TraefikFrontendFrameDeny, false),
		ContentTypeNosniff:      label.GetBoolValue(labels, label.TraefikFrontendContentTypeNosniff, false),
		BrowserXSSFilter:        label.GetBoolValue(labels, label.TraefikFrontendBrowserXSSFilter, false),
		IsDevelopment:           label.GetBoolValue(labels, label.TraefikFrontendIsDevelopment, false),
		SSLHost:                 label.GetStringValue(labels, label.TraefikFrontendSSLHost, ""),
		CustomFrameOptionsValue: label.GetStringValue(labels, label.TraefikFrontendCustomFrameOptionsValue, ""),
		ContentSecurityPolicy:   label.GetStringValue(labels, label.TraefikFrontendContentSecurityPolicy, ""),
		PublicKey:               label.GetStringValue(labels, label.TraefikFrontendPublicKey, ""),
		ReferrerPolicy:          label.GetStringValue(labels, label.TraefikFrontendReferrerPolicy, ""),
		CustomBrowserXSSValue:   label.GetStringValue(labels, label.TraefikFrontendCustomBrowserXSSValue, ""),
	}

	if !headers.HasSecureHeadersDefined() && !headers.HasCustomHeadersDefined() {
		return nil
	}

	return headers
}

func getPort(container dockerData) string {
	if value := label.GetStringValue(container.RoadLabels, label.TraefikPort, ""); len(value) != 0 {
		return value
	}

	// See iteration order in https://blog.golang.org/go-maps-in-action
	var ports []nat.Port
	for port := range container.NetworkSettings.Ports {
		ports = append(ports, port)
	}

	less := func(i, j nat.Port) bool {
		return i.Int() < j.Int()
	}
	nat.Sort(ports, less)

	if len(ports) > 0 {
		min := ports[0]
		return min.Port()
	}

	return ""
}

func getMaxConn(labels map[string]string) *types.MaxConn {
	amount := label.GetInt64Value(labels, label.TraefikBackendMaxConnAmount, math.MinInt64)
	extractorFunc := label.GetStringValue(labels, label.TraefikBackendMaxConnExtractorFunc, label.DefaultBackendMaxconnExtractorFunc)

	if amount == math.MinInt64 || len(extractorFunc) == 0 {
		return nil
	}

	return &types.MaxConn{
		Amount:        amount,
		ExtractorFunc: extractorFunc,
	}
}

func getHealthCheck(labels map[string]string) *types.HealthCheck {
	path := label.GetStringValue(labels, label.TraefikBackendHealthCheckPath, "")
	if len(path) == 0 {
		return nil
	}

	port := label.GetIntValue(labels, label.TraefikBackendHealthCheckPort, label.DefaultBackendHealthCheckPort)
	interval := label.GetStringValue(labels, label.TraefikBackendHealthCheckInterval, "")

	return &types.HealthCheck{
		Path:     path,
		Port:     port,
		Interval: interval,
	}
}

func getBuffering(labels map[string]string) *types.Buffering {
	if !label.HasPrefix(labels, label.TraefikBackendBuffering) {
		return nil
	}

	return &types.Buffering{
		MaxRequestBodyBytes:  label.GetInt64Value(labels, label.TraefikBackendBufferingMaxRequestBodyBytes, 0),
		MaxResponseBodyBytes: label.GetInt64Value(labels, label.TraefikBackendBufferingMaxResponseBodyBytes, 0),
		MemRequestBodyBytes:  label.GetInt64Value(labels, label.TraefikBackendBufferingMemRequestBodyBytes, 0),
		MemResponseBodyBytes: label.GetInt64Value(labels, label.TraefikBackendBufferingMemResponseBodyBytes, 0),
		RetryExpression:      label.GetStringValue(labels, label.TraefikBackendBufferingRetryExpression, ""),
	}
}

func getCircuitBreaker(labels map[string]string) *types.CircuitBreaker {
	circuitBreaker := label.GetStringValue(labels, label.TraefikBackendCircuitBreakerExpression, "")
	if len(circuitBreaker) == 0 {
		return nil
	}
	return &types.CircuitBreaker{Expression: circuitBreaker}
}

func getLoadBalancer(labels map[string]string) *types.LoadBalancer {
	if !label.HasPrefix(labels, label.TraefikBackendLoadBalancer) {
		return nil
	}

	method := label.GetStringValue(labels, label.TraefikBackendLoadBalancerMethod, label.DefaultBackendLoadBalancerMethod)

	lb := &types.LoadBalancer{
		Method: method,
		Sticky: getSticky(labels),
	}

	if label.GetBoolValue(labels, label.TraefikBackendLoadBalancerStickiness, false) {
		cookieName := label.GetStringValue(labels, label.TraefikBackendLoadBalancerStickinessCookieName, label.DefaultBackendLoadbalancerStickinessCookieName)
		lb.Stickiness = &types.Stickiness{CookieName: cookieName}
	}

	return lb
}

// TODO: Deprecated
// replaced by Stickiness
// Deprecated
func getSticky(labels map[string]string) bool {
	if label.Has(labels, label.TraefikBackendLoadBalancerSticky) {
		log.Warnf("Deprecated configuration found: %s. Please use %s.", label.TraefikBackendLoadBalancerSticky, label.TraefikBackendLoadBalancerStickiness)
	}

	return label.GetBoolValue(labels, label.TraefikBackendLoadBalancerSticky, false)
}

func (p *Provider) getServers(containers []dockerData) map[string]types.Server {
	var servers map[string]types.Server

	for i, container := range containers {
		if servers == nil {
			servers = make(map[string]types.Server)
		}

		protocol := label.GetStringValue(container.RoadLabels, label.TraefikProtocol, label.DefaultProtocol)
		ip := p.getIPAddress(container)
		port := getPort(container)

		serverName := "server-" + container.RoadName + "-" + container.Name
		if len(container.RoadName) > 0 {
			serverName += "-" + strconv.Itoa(i)
		}

		servers[provider.Normalize(serverName)] = types.Server{
			URL:    fmt.Sprintf("%s://%s:%s", protocol, ip, port),
			Weight: label.GetIntValue(container.RoadLabels, label.TraefikWeight, label.DefaultWeightInt),
		}
	}

	return servers
}

func getFuncStringLabel(labelName string, defaultValue string) func(map[string]string) string {
	return func(labels map[string]string) string {
		return label.GetStringValue(labels, labelName, defaultValue)
	}
}

func getFuncIntLabel(labelName string, defaultValue int) func(map[string]string) int {
	return func(labels map[string]string) int {
		return label.GetIntValue(labels, labelName, defaultValue)
	}
}

func getFuncBoolLabel(labelName string, defaultValue bool) func(map[string]string) bool {
	return func(labels map[string]string) bool {
		return label.GetBoolValue(labels, labelName, defaultValue)
	}
}

func getFuncSliceStringLabel(labelName string) func(map[string]string) []string {
	return func(labels map[string]string) []string {
		return label.GetSliceStringValue(labels, labelName)
	}
}
