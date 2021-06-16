package k8s

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/traefik/traefik/v2/pkg/log"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
)

// MustParseYaml parses a YAML to objects.
func MustParseYaml(content []byte) []runtime.Object {
	acceptedK8sTypes := regexp.MustCompile(`^(Deployment|Endpoints|Service|Ingress|IngressRoute|IngressRouteTCP|IngressRouteUDP|Middleware|MiddlewareTCP|Secret|TLSOption|TLSStore|TraefikService|IngressClass|ServersTransport|GatewayClass|Gateway|HTTPRoute|TCPRoute|TLSRoute)$`)

	files := strings.Split(string(content), "---")
	retVal := make([]runtime.Object, 0, len(files))
	for _, file := range files {
		if file == "\n" || file == "" {
			continue
		}

		decode := scheme.Codecs.UniversalDeserializer().Decode
		obj, groupVersionKind, err := decode([]byte(file), nil, nil)
		if err != nil {
			panic(fmt.Sprintf("Error while decoding YAML object. Err was: %s", err))
		}

		if !acceptedK8sTypes.MatchString(groupVersionKind.Kind) {
			log.WithoutContext().Debugf("The custom-roles configMap contained K8s object types which are not supported! Skipping object with type: %s", groupVersionKind.Kind)
		} else {
			retVal = append(retVal, obj)
		}
	}
	return retVal
}
