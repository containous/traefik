package v1alpha1

// ObjectReference is a generic reference to a Traefik resource.
type ObjectReference struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
}
