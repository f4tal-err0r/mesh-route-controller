package main

import (
	"context"
	"time"

	"log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	versionedclient "istio.io/client-go/pkg/clientset/versioned"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
)

func main() {

	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	} else {
		log.Println("Rest connection to kubernetes successful")
	}

	ics, err := versionedclient.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed to create istio client: %s", err)
	} else {
		log.Println("Fetched istio Clientset successfully")
	}

	controller := NewController(ics)

	stop := make(chan struct{})
	defer close(stop)
	go controller.vsInformer.Run(stop)
	for {
		controller.vsClient.Update(context.TODO(), controller.meshRouteManifest, metav1.UpdateOptions{})
		time.Sleep(time.Second * 30) //seconds between updates
	}
}
