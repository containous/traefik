package v1alpha1

import (
	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MiddlewareTCP is a specification for a MiddlewareTCP resource.
type MiddlewareTCP struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec MiddlewareTCPSpec `json:"spec"`
}

// +k8s:deepcopy-gen=true

// MiddlewareTCPSpec holds the MiddlewareTCP configuration.
type MiddlewareTCPSpec struct {
	IPWhiteList *dynamic.TCPIPWhiteList `json:"ipWhiteList,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MiddlewareTCPList is a list of MiddlewareTCP resources.
type MiddlewareTCPList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []MiddlewareTCP `json:"items"`
}
