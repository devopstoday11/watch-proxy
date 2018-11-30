package fake

import (
	v1alpha3 "github.com/heptio/quartermaster/custom/apis/virtualservice/v1alpha3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeVirtualServices implements VirtualServiceInterface
type FakeVirtualServices struct {
	Fake *FakeVirtualserviceV1alpha3
	ns   string
}

var virtualservicesResource = schema.GroupVersionResource{Group: "virtualservice", Version: "v1alpha3", Resource: "virtualservices"}

var virtualservicesKind = schema.GroupVersionKind{Group: "virtualservice", Version: "v1alpha3", Kind: "VirtualService"}

// Get takes name of the virtualService, and returns the corresponding virtualService object, and an error if there is any.
func (c *FakeVirtualServices) Get(name string, options v1.GetOptions) (result *v1alpha3.VirtualService, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(virtualservicesResource, c.ns, name), &v1alpha3.VirtualService{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha3.VirtualService), err
}

// List takes label and field selectors, and returns the list of VirtualServices that match those selectors.
func (c *FakeVirtualServices) List(opts v1.ListOptions) (result *v1alpha3.VirtualServiceList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(virtualservicesResource, virtualservicesKind, c.ns, opts), &v1alpha3.VirtualServiceList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha3.VirtualServiceList{}
	for _, item := range obj.(*v1alpha3.VirtualServiceList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested virtualServices.
func (c *FakeVirtualServices) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(virtualservicesResource, c.ns, opts))

}

// Create takes the representation of a virtualService and creates it.  Returns the server's representation of the virtualService, and an error, if there is any.
func (c *FakeVirtualServices) Create(virtualService *v1alpha3.VirtualService) (result *v1alpha3.VirtualService, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(virtualservicesResource, c.ns, virtualService), &v1alpha3.VirtualService{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha3.VirtualService), err
}

// Update takes the representation of a virtualService and updates it. Returns the server's representation of the virtualService, and an error, if there is any.
func (c *FakeVirtualServices) Update(virtualService *v1alpha3.VirtualService) (result *v1alpha3.VirtualService, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(virtualservicesResource, c.ns, virtualService), &v1alpha3.VirtualService{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha3.VirtualService), err
}

// Delete takes name of the virtualService and deletes it. Returns an error if one occurs.
func (c *FakeVirtualServices) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(virtualservicesResource, c.ns, name), &v1alpha3.VirtualService{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeVirtualServices) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(virtualservicesResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha3.VirtualServiceList{})
	return err
}

// Patch applies the patch and returns the patched virtualService.
func (c *FakeVirtualServices) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha3.VirtualService, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(virtualservicesResource, c.ns, name, data, subresources...), &v1alpha3.VirtualService{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha3.VirtualService), err
}