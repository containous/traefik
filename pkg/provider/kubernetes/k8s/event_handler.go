package k8s

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ResourceEventHandler handles Add, Update or Delete Events for resources.
type ResourceEventHandler struct {
	Ev chan<- interface{}
}

// OnAdd is called on Add Events.
func (reh *ResourceEventHandler) OnAdd(obj interface{}) {
	eventHandlerFunc(reh.Ev, obj)
}

// OnUpdate is called on Update Events.
// Ignores useless changes.
func (reh *ResourceEventHandler) OnUpdate(oldObj, newObj interface{}) {
	if objChanged(oldObj, newObj) {
		eventHandlerFunc(reh.Ev, newObj)
	}
}

// OnDelete is called on Delete Events.
func (reh *ResourceEventHandler) OnDelete(obj interface{}) {
	eventHandlerFunc(reh.Ev, obj)
}

// eventHandlerFunc will pass the obj on to the events channel or drop it.
// This is so passing the events along won't block in the case of high volume.
// The events are only used for signaling anyway so dropping a few is ok.
func eventHandlerFunc(events chan<- interface{}, obj interface{}) {
	select {
	case events <- obj:
	default:
	}
}

func objChanged(oldObj, newObj interface{}) bool {
	if oldObj == nil || newObj == nil {
		return true
	}

	if oldObj.(metav1.Object).GetResourceVersion() == newObj.(metav1.Object).GetResourceVersion() {
		return false
	}

	if _, ok := oldObj.(*corev1.Endpoints); ok {
		if endpointsChanged(oldObj.(*corev1.Endpoints), newObj.(*corev1.Endpoints)) {
			return true
		}
	}

	return false
}

func endpointsChanged(a, b *corev1.Endpoints) bool {
	if len(a.Subsets) != len(b.Subsets) {
		return true
	}

	for i, sa := range a.Subsets {
		sb := b.Subsets[i]
		if subsetsChanged(sa, sb) {
			return true
		}
	}

	return false
}

func subsetsChanged(sa, sb corev1.EndpointSubset) bool {
	if len(sa.Addresses) != len(sb.Addresses) {
		return true
	}

	if len(sa.Ports) != len(sb.Ports) {
		return true
	}

	// in Addresses and Ports, we should be able to rely on
	// these being sorted and able to be compared
	// they are supposed to be in a canonical format
	for addr, aaddr := range sa.Addresses {
		baddr := sb.Addresses[addr]
		if aaddr.IP != baddr.IP {
			return true
		}

		if aaddr.Hostname != baddr.Hostname {
			return true
		}
	}

	for port, aport := range sa.Ports {
		bport := sb.Ports[port]
		if aport.Name != bport.Name {
			return true
		}
		if aport.Port != bport.Port {
			return true
		}

		if aport.Protocol != bport.Protocol {
			return true
		}
	}

	return false
}
