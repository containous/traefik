package v1alpha1

import (
	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	"github.com/traefik/traefik/v2/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// IngressRouteTCPSpec is a specification for a IngressRouteTCPSpec resource.
type IngressRouteTCPSpec struct {
	Routes      []RouteTCP `json:"routes"`
	EntryPoints []string   `json:"entryPoints,omitempty"`
	TLS         *TLSTCP    `json:"tls,omitempty"`
}

// RouteTCP contains the set of routes.
type RouteTCP struct {
	Match    string       `json:"match"`
	Services []ServiceTCP `json:"services,omitempty"`
	// Middlewares contains references to MiddlewareTCP resources.
	Middlewares []ObjectReference `json:"middlewares,omitempty"`
}

// TLSTCP contains the TLS certificates configuration of the routes.
// To enable Let's Encrypt, use an empty TLS struct,
// e.g. in YAML:
//
//	 tls: {} # inline format
//
//	 tls:
//	   secretName: # block format
type TLSTCP struct {
	// SecretName is the name of the referenced Kubernetes Secret to specify the
	// certificate details.
	SecretName  string `json:"secretName,omitempty"`
	Passthrough bool   `json:"passthrough,omitempty"`
	// Options is a reference to a TLSOption, that specifies the parameters of the TLS connection.
	Options *ObjectReference `json:"options,omitempty"`
	// Store is a reference to a TLSStore, that specifies the parameters of the TLS store.
	Store        *ObjectReference `json:"store,omitempty"`
	CertResolver string           `json:"certResolver,omitempty"`
	Domains      []types.Domain   `json:"domains,omitempty"`
}

// ServiceTCP defines an upstream to proxy traffic.
type ServiceTCP struct {
	Name             string                 `json:"name"`
	Namespace        string                 `json:"namespace,omitempty"`
	Port             intstr.IntOrString     `json:"port"`
	Weight           *int                   `json:"weight,omitempty"`
	TerminationDelay *int                   `json:"terminationDelay,omitempty"`
	ProxyProtocol    *dynamic.ProxyProtocol `json:"proxyProtocol,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:storageversion

// IngressRouteTCP is an Ingress CRD specification.
type IngressRouteTCP struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec IngressRouteTCPSpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// IngressRouteTCPList is a list of IngressRoutes.
type IngressRouteTCPList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []IngressRouteTCP `json:"items"`
}
