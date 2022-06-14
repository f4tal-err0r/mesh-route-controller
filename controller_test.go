package main

import (
	"context"
	"reflect"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"istio.io/api/networking/v1beta1"
	spec "istio.io/client-go/pkg/apis/networking/v1beta1"

	versionedclient "istio.io/client-go/pkg/clientset/versioned/fake"
)

var (
	controller              *Controller
	virtualServiceInterface *spec.VirtualService
)

// TestController tests the controller
func TestController(t *testing.T) {
	//create fake versioned clientset
	versionedclientset := versionedclient.NewSimpleClientset()

	//create fake service entry for haproxy

	haproxyServiceEntry := &spec.ServiceEntry{
		ObjectMeta: metav1.ObjectMeta{
			Name: "haproxy",
		},
		Spec: v1beta1.ServiceEntry{
			Hosts: []string{"test.haproxy.example.com"},
			Ports: []*v1beta1.Port{
				{
					Name:       "http",
					TargetPort: 80,
					Protocol:   "HTTP",
				},
			},
			Location:   0,
			Resolution: 2,
		},
	}

	//create fake virtual service spec
	virtualServiceSpec := &spec.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name: "mesh-routing",
		},
		Spec: v1beta1.VirtualService{
			Hosts: []string{"test.haproxy.example.com", "*.devbettercloud.internal"},
			Gateways: []string{
				"mesh",
			},
			Http: []*v1beta1.HTTPRoute{
				{
					Name: "default-haproxy",
					Route: []*v1beta1.HTTPRouteDestination{
						{
							Destination: &v1beta1.Destination{
								Host: "test.haproxy.example.com",
							},
						},
					},
				},
			},
		},
	}

	//add service entry to fake clientset
	_, err := versionedclientset.NetworkingV1beta1().ServiceEntries("infra").Create(context.TODO(), haproxyServiceEntry, metav1.CreateOptions{})
	if err != nil {
		t.Error(err)
	}

	//test service entry was added to fake clientset
	serviceEntry, err := versionedclientset.NetworkingV1beta1().ServiceEntries("infra").Get(context.TODO(), "haproxy", metav1.GetOptions{})
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(serviceEntry.Spec, haproxyServiceEntry.Spec) {
		t.Errorf("Expected %v,\n\n got %v", haproxyServiceEntry.Spec, serviceEntry.Spec)
	}

	//create fake controller
	controller = NewController(versionedclientset)

	//add virtualservice to clientset
	vs, err := versionedclientset.NetworkingV1beta1().VirtualServices("infra").Get(context.TODO(), "mesh-routing", metav1.GetOptions{})

	if !reflect.DeepEqual(virtualServiceSpec.Spec, vs.Spec) {
		t.Errorf("Expected %v,\n\n got %v", virtualServiceSpec.Spec, vs.Spec)
	}
}

// TestControllerAdd tests the controller add function
func TestControllerAdd(t *testing.T) {
	//Define an interface for the function to test
	type testFunc func(*Controller, *spec.VirtualService)

	//create a fake virtualservice interface
	virtualServiceInterface = &spec.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service",
			Namespace: "testing",
			Labels:    map[string]string{"bc-network": "edge"},
		},
		Spec: v1beta1.VirtualService{
			Hosts: []string{"test.service.testing"},
			Gateways: []string{
				"test/gateway",
			},
			Http: []*v1beta1.HTTPRoute{
				{
					Name: "test-route",
					Match: []*v1beta1.HTTPMatchRequest{
						{
							Uri: &v1beta1.StringMatch{
								MatchType: &v1beta1.StringMatch_Prefix{
									Prefix: "/test/service",
								},
							},
						},
					},
					Route: []*v1beta1.HTTPRouteDestination{
						{
							Destination: &v1beta1.Destination{
								Host: "test.service.testing",
							},
						},
					},
				},
			},
		},
	}

	controller.AddRoute(virtualServiceInterface)
	controller.ic.NetworkingV1beta1().VirtualServices("testing").Create(context.TODO(), virtualServiceInterface, metav1.CreateOptions{})

	//Check that the virtualservice was added to the fake clientset
	virtualService, err := controller.ic.NetworkingV1beta1().VirtualServices("testing").Get(context.TODO(), "test-service", metav1.GetOptions{})
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(virtualServiceInterface.Spec, virtualService.Spec) {
		t.Errorf("Expected %v,\n\n got %v", virtualServiceInterface.Spec, virtualService.Spec)
	}

	controller.vsClient.Update(context.TODO(), controller.meshRouteManifest, metav1.UpdateOptions{})

	//Check that the virtualservice was added to the mesh-routing virtualservice
	meshRouting, err := controller.ic.NetworkingV1beta1().VirtualServices("infra").Get(context.TODO(), "mesh-routing", metav1.GetOptions{})
	if err != nil {
		t.Error(err)
	}

	if len(meshRouting.Spec.Http) != 2 {
		t.Errorf("Expected 2 http routes, got %v", len(meshRouting.Spec.Http))
	}

	if !reflect.DeepEqual(virtualServiceInterface.Spec.Http[0], meshRouting.Spec.Http[1]) {
		t.Errorf("Expected %v,\n\n got %v", virtualServiceInterface.Spec.Http[0], meshRouting.Spec.Http[1])
	}

}

//TestControllerUpdate tests the controller update function
func TestControllerUpdate(t *testing.T) {

	updatedVS := virtualServiceInterface.DeepCopy()

	updatedVS.Spec.Http[0].Match[0].Uri.MatchType = &v1beta1.StringMatch_Prefix{Prefix: "/test/service/updated"}
	updatedVS.Spec.Http[0].Route[0].Destination.Host = "test.service.updated"

	controller.ic.NetworkingV1beta1().VirtualServices("testing").Update(context.TODO(), updatedVS, metav1.UpdateOptions{})

	controller.UpdateRoute(virtualServiceInterface, updatedVS)

	controller.vsClient.Update(context.TODO(), controller.meshRouteManifest, metav1.UpdateOptions{})

	//Check that the virtualservice was updated in the mesh-routing virtualservice
	meshRouting, err := controller.ic.NetworkingV1beta1().VirtualServices("infra").Get(context.TODO(), "mesh-routing", metav1.GetOptions{})
	if err != nil {
		t.Error(err)
	}

	if len(meshRouting.Spec.Http) != 2 {
		t.Errorf("Expected 2 http routes, got %v", len(meshRouting.Spec.Http))
	}

	if !reflect.DeepEqual(updatedVS.Spec.Http[0], meshRouting.Spec.Http[1]) {
		t.Errorf("Expected %v,\n\n got %v", updatedVS.Spec.Http[0], meshRouting.Spec.Http[1])
	}

}

//TestControllerDelete tests the controller delete function
func TestControllerDelete(t *testing.T) {
	updatedVS := virtualServiceInterface.DeepCopy()

	updatedVS.Spec.Http[0].Match[0].Uri.MatchType = &v1beta1.StringMatch_Prefix{Prefix: "/test/service/updated"}
	updatedVS.Spec.Http[0].Route[0].Destination.Host = "test.service.updated"

	controller.ic.NetworkingV1beta1().VirtualServices("testing").Delete(context.TODO(), "test-service", metav1.DeleteOptions{})

	controller.DeleteRoute(updatedVS)

	controller.vsClient.Update(context.TODO(), controller.meshRouteManifest, metav1.UpdateOptions{})

	//Check that the virtualservice was created in the mesh-routing virtualservice
	meshRouting, err := controller.ic.NetworkingV1beta1().VirtualServices("infra").Get(context.TODO(), "mesh-routing", metav1.GetOptions{})
	if err != nil {
		t.Error(err)
	}

	if len(meshRouting.Spec.Http) != 1 {
		t.Errorf("Expected 1 http routes, got %v", len(meshRouting.Spec.Http))
	}
}
