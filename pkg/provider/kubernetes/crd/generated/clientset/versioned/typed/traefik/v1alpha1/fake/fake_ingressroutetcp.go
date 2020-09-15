/*
The MIT License (MIT)

Copyright (c) 2016-2020 Containous SAS

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	v1alpha1 "github.com/traefik/traefik/v2/pkg/provider/kubernetes/crd/traefik/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeIngressRouteTCPs implements IngressRouteTCPInterface
type FakeIngressRouteTCPs struct {
	Fake *FakeTraefikV1alpha1
	ns   string
}

var ingressroutetcpsResource = schema.GroupVersionResource{Group: "traefik.containo.us", Version: "v1alpha1", Resource: "ingressroutetcps"}

var ingressroutetcpsKind = schema.GroupVersionKind{Group: "traefik.containo.us", Version: "v1alpha1", Kind: "IngressRouteTCP"}

// Get takes name of the ingressRouteTCP, and returns the corresponding ingressRouteTCP object, and an error if there is any.
func (c *FakeIngressRouteTCPs) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.IngressRouteTCP, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(ingressroutetcpsResource, c.ns, name), &v1alpha1.IngressRouteTCP{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.IngressRouteTCP), err
}

// List takes label and field selectors, and returns the list of IngressRouteTCPs that match those selectors.
func (c *FakeIngressRouteTCPs) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.IngressRouteTCPList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(ingressroutetcpsResource, ingressroutetcpsKind, c.ns, opts), &v1alpha1.IngressRouteTCPList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.IngressRouteTCPList{ListMeta: obj.(*v1alpha1.IngressRouteTCPList).ListMeta}
	for _, item := range obj.(*v1alpha1.IngressRouteTCPList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested ingressRouteTCPs.
func (c *FakeIngressRouteTCPs) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(ingressroutetcpsResource, c.ns, opts))

}

// Create takes the representation of a ingressRouteTCP and creates it.  Returns the server's representation of the ingressRouteTCP, and an error, if there is any.
func (c *FakeIngressRouteTCPs) Create(ctx context.Context, ingressRouteTCP *v1alpha1.IngressRouteTCP, opts v1.CreateOptions) (result *v1alpha1.IngressRouteTCP, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(ingressroutetcpsResource, c.ns, ingressRouteTCP), &v1alpha1.IngressRouteTCP{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.IngressRouteTCP), err
}

// Update takes the representation of a ingressRouteTCP and updates it. Returns the server's representation of the ingressRouteTCP, and an error, if there is any.
func (c *FakeIngressRouteTCPs) Update(ctx context.Context, ingressRouteTCP *v1alpha1.IngressRouteTCP, opts v1.UpdateOptions) (result *v1alpha1.IngressRouteTCP, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(ingressroutetcpsResource, c.ns, ingressRouteTCP), &v1alpha1.IngressRouteTCP{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.IngressRouteTCP), err
}

// Delete takes name of the ingressRouteTCP and deletes it. Returns an error if one occurs.
func (c *FakeIngressRouteTCPs) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(ingressroutetcpsResource, c.ns, name), &v1alpha1.IngressRouteTCP{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeIngressRouteTCPs) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(ingressroutetcpsResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.IngressRouteTCPList{})
	return err
}

// Patch applies the patch and returns the patched ingressRouteTCP.
func (c *FakeIngressRouteTCPs) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.IngressRouteTCP, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(ingressroutetcpsResource, c.ns, name, pt, data, subresources...), &v1alpha1.IngressRouteTCP{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.IngressRouteTCP), err
}
