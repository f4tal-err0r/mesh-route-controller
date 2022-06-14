package main

import (
	"context"
	"log"

	"mesh-route-generator/pkg/utils"

	"istio.io/api/meta/v1alpha1"
	"istio.io/api/networking/v1beta1"
	spec "istio.io/client-go/pkg/apis/networking/v1beta1"
	versionedclient "istio.io/client-go/pkg/clientset/versioned"
	vs "istio.io/client-go/pkg/clientset/versioned/typed/networking/v1beta1"
	informers "istio.io/client-go/pkg/informers/externalversions"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/cache"
)

type Controller struct {
	ic                versionedclient.Interface
	vsInformer        cache.SharedIndexInformer
	vsClient          vs.VirtualServiceInterface
	meshRouteManifest *spec.VirtualService

	utils.SliceLock
}

func NewController(ic versionedclient.Interface) *Controller {
	var err error

	controller := Controller{}
	controller.SliceLock = utils.NewLock()

	controller.ic = ic

	controller.vsClient = ic.NetworkingV1beta1().VirtualServices("infra")

	sharedInformer := informers.NewSharedInformerFactory(
		ic,
		0,
	)

	controller.vsInformer = sharedInformer.Networking().V1beta1().VirtualServices().Informer()

	controller.meshRouteManifest, err = controller.vsClient.Get(context.TODO(), "mesh-routing", metav1.GetOptions{})
	if errors.IsNotFound(err) {
		log.Println("Mesh routing manifest not found, initializing...")
		controller.meshRouteManifest, err = generate_routes(controller.vsClient, ic)
		if err != nil {
			log.Fatal(err)
		}
	} else if err != nil {
		log.Println(err)
	} else {
		log.Println("Found mesh-routing manifest")
	}

	controller.vsInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(vs interface{}) {
			controller.AddRoute(vs)
		},
		UpdateFunc: func(oldvs, vs interface{}) {
			controller.UpdateRoute(oldvs, vs)
		},
		DeleteFunc: func(vs interface{}) {
			controller.DeleteRoute(vs)
		},
	},
	)

	return &controller
}

func generate_routes(vsClient vs.VirtualServiceInterface, ic versionedclient.Interface) (*spec.VirtualService, error) {
	var meshHttpRoutes []*v1beta1.HTTPRoute

	vsList, err := ic.NetworkingV1beta1().VirtualServices("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Fatalf("Failed to get VirtualService's from Kubernetes: %s", err)
	}

	//List all virtualservices in all namespaces and iterate over it
	for iter := range vsList.Items {
		vs := vsList.Items[iter]

		//Fetch the http route rules from the spec only in resources with the correct label
		for _, httpRule := range vs.Spec.GetHttp() {
			if vs.Labels["bc-network"] == "edge" {
				meshHttpRoutes = append(meshHttpRoutes, httpRule)
			}
		}
	}

	//Initialize default route
	haproxySe, err := ic.NetworkingV1beta1().ServiceEntries("infra").Get(context.TODO(), "haproxy", metav1.GetOptions{})
	if err != nil {
		log.Fatal("Failed to find haproxy default ServiceEntry manifest in infra namespace.")
	}

	haproxyHost := haproxySe.Spec.GetHosts()[0]

	routeHaproxy := &v1beta1.HTTPRoute{
		Name: "default-haproxy",
		Route: []*v1beta1.HTTPRouteDestination{
			&v1beta1.HTTPRouteDestination{
				Destination: &v1beta1.Destination{
					Host: haproxyHost,
				},
			},
		},
	}

	meshHttpRoutes = append(meshHttpRoutes, routeHaproxy)

	mergedManifest := &spec.VirtualService{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "networking.istio.io/v1beta1",
			Kind:       "VirtualService",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mesh-routing",
			Namespace: "infra",
			Labels:    map[string]string{"bc-network": "mesh"},
		},
		Spec: v1beta1.VirtualService{
			Hosts:    []string{haproxyHost, "*.devbettercloud.internal"},
			Gateways: []string{"mesh"},
			Http:     meshHttpRoutes,
		},
		Status: v1alpha1.IstioStatus{},
	}

	vsClient.Create(context.TODO(), mergedManifest, metav1.CreateOptions{})

	return mergedManifest, err
}

func (c *Controller) AddRoute(vs interface{}) {
	vsspec := vs.(*spec.VirtualService)

	if vsspec.Labels["bc-network"] == "edge" {
		log.Printf("Detected added virtualservice %s/%s", vsspec.Namespace, vsspec.Name)

		c.meshRouteManifest.Spec.Http = c.Append(c.meshRouteManifest.Spec.Http, vsspec.Spec.Http)
	}
}

func (c *Controller) UpdateRoute(oldvs, vs interface{}) {
	vsspec := vs.(*spec.VirtualService)
	oldvsspec := oldvs.(*spec.VirtualService)

	if vsspec.Labels["bc-network"] == "edge" {
		log.Printf("Detected update in virtualservice %s/%s", vsspec.Namespace, vsspec.Name)

		c.meshRouteManifest.Spec.Http = c.Update(c.meshRouteManifest.Spec.Http, vsspec.Spec.Http, oldvsspec.Spec.Http)
	}
}

func (c *Controller) DeleteRoute(vs interface{}) {
	vsspec := vs.(*spec.VirtualService)
	if vsspec.Labels["bc-network"] == "edge" {
		log.Printf("Detected deletion of virtualservice %s/%s", vsspec.Namespace, vsspec.Name)

		c.meshRouteManifest.Spec.Http = c.Delete(c.meshRouteManifest.Spec.Http, vsspec.Spec.Http)
	}
}
