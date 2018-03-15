package kubecluster

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/pborman/uuid"

	"github.com/heptio/quartermaster/config"
	"github.com/heptio/quartermaster/emitter"
	"github.com/heptio/quartermaster/inventory"

	"k8s.io/api/apps/v1beta2"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// Takes a snapshot of the cluster and creates the Cluster object
var (
	uuidLock sync.Mutex
	lastUUID uuid.UUID
)

func Initialize(client *kubernetes.Clientset, config config.Config) {

	cluster := new(inventory.Cluster)
	cluster.Deployments = make(map[string][]inventory.Deployment)
	cluster.Pods = make(map[string][]inventory.Pod)
	clusterPods := []inventory.Pod{}
	clusterPod := inventory.Pod{}
	clusterDeployment := inventory.Deployment{}
	cluster.UID = NewUID()
	cluster.Version = Version(client)
	cluster.Name = GetClusterName(client)

	//Get namespaces, since the other objects are based on namespace names as Keys
	//we willl loop over the namespaces to construct the rest of the inventory.
	namespaces, err := client.CoreV1().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		log.Println("Could not get namespaces", err)
	}
	for _, ns := range namespaces.Items {
		//create the cluster namespace struct and add it to the cluster list
		clusterNamespace := inventory.Namespace{
			Name:  ns.Name,
			Event: "created",
			Kind:  "namespace",
			UID:   NewUID(),
		}

		cluster.Namespaces = append(cluster.Namespaces, clusterNamespace)

		deployments, err := client.AppsV1().Deployments(ns.Name).List(metav1.ListOptions{})
		if err != nil {
			log.Println("Could not get deployments", err)
		}
		for _, deployment := range deployments.Items {
			clusterDeployment = inventory.Deployment{
				Name:            deployment.ObjectMeta.Name,
				Namespace:       deployment.ObjectMeta.Namespace,
				Labels:          deployment.ObjectMeta.Labels,
				ReplicasDesired: *deployment.Spec.Replicas,
				Event:           "created",
				Kind:            "deployment",
				UID:             NewUID(),
			}
			//clusterDeployments = append(clusterDeployments, clusterDeployment)
			cluster.Deployments[ns.Name] = append(cluster.Deployments[ns.Name], clusterDeployment)

		}

		pods, err := client.CoreV1().Pods(ns.Name).List(metav1.ListOptions{})
		if err != nil {
			log.Println("Could not get pods", err)
		}
		for _, pod := range pods.Items {
			clusterPod = inventory.Pod{
				Name:      pod.ObjectMeta.Name,
				Namespace: pod.ObjectMeta.Namespace,
				Labels:    pod.ObjectMeta.Labels,
				Images:    imagesFromContainers(pod.Spec.Containers),
				Event:     "created",
				Kind:      "pod",
				UID:       NewUID(),
			}
			clusterPods = append(clusterPods, clusterPod)
		}
		cluster.Pods[ns.Name] = clusterPods

	}
	fmt.Printf("Constructed this cluster: %v", cluster)
	emitter.EmitChanges(cluster, config.RemoteEndpoint)
}

// StartWatchers - start up the k8s resource watchers
func StartWatchers(client *kubernetes.Clientset, config config.Config) map[string]chan bool {
	// loop through list of resources to watch and startup watchers
	// for those resources.
	resources := []string{}
	// decide if we want to load the full set of resources. Usually at startup
	// or just start up new watchers after config change
	if len(config.NewResources) > 0 {
		resources = config.NewResources
	} else {
		resources = config.ResourcesWatch
	}

	// create map to capture the channels used to signal done to the watcher
	// go routines
	doneChannels := make(map[string]chan bool)

	// start the watchers
	for _, resource := range resources {
		switch resource {
		case "namespaces":
			nameDone := make(chan bool)
			doneChannels[resource] = nameDone
			// watch namespaces
			go func() {
				Namespaces(client, config, nameDone)
			}()
		case "pods":
			// watch pods
			podDone := make(chan bool)
			doneChannels[resource] = podDone
			// watch pods
			go func() {
				Pods(client, config, podDone)
			}()
		case "deployments":
			depDone := make(chan bool)
			doneChannels[resource] = depDone
			// watch deployments
			go func() {
				Deployments(client, config, depDone)
			}()
		}
	}

	// return the map of done channels so we can stop things later if need be
	return doneChannels
}

// StopWatchers - stop the watchers we no longer want running
func StopWatchers(doneChannels map[string]chan bool, config config.Config) {
	// loop through list of stale resource types and stop those watchers
	for _, staleRes := range config.StaleResources {
		log.Println("Stopping", staleRes)
		// send true to the done channel for the respective resource
		// to close the go routine
		doneChannels[staleRes] <- true
	}
}

// Version - get the version of k8s running on the cluster
func Version(client *kubernetes.Clientset) string {

	version, err := client.DiscoveryClient.ServerVersion()
	if err != nil {
		log.Println("Could not get server version", err)
	}
	return version.GitVersion

}

func GetClusterName(client *kubernetes.Clientset) string {
	nodes, err := client.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		log.Println("Could not get nodes", err)
	}
	node := nodes.Items[0]
	clusterName := node.ObjectMeta.Labels["cluster-name"]

	return clusterName
}

// Namespaces - Emit events on Namespace create and deletions
func Namespaces(client *kubernetes.Clientset, config config.Config, done chan bool) {

	watchlist := cache.NewListWatchFromClient(client.Core().RESTClient(), "namespaces", v1.NamespaceAll,
		fields.Everything())
	_, controller := cache.NewInformer(
		watchlist,
		&v1.Namespace{},
		time.Second*60,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				ns := obj.(*v1.Namespace)
				inv := inventory.Namespace{
					Name:  ns.ObjectMeta.Name,
					Event: "created",
					Kind:  "namespace",
					UID:   NewUID(),
				}

				// sent the update to remote endpoint
				emitter.EmitChanges(inv, config.RemoteEndpoint)

				log.Printf("Namespace Created: %s",
					ns.ObjectMeta.Name,
				)

			},
			DeleteFunc: func(obj interface{}) {
				ns := obj.(*v1.Namespace)

				inv := inventory.Namespace{
					Name:  ns.ObjectMeta.Name,
					Event: "deleted",
					Kind:  "namespace",
					UID:   NewUID(),
				}

				// sent the update to remote endpoint
				emitter.EmitChanges(inv, config.RemoteEndpoint)

				log.Printf("Namespace Deleted: %s",
					ns.ObjectMeta.Name,
				)

			},
		},
	)

	stop := make(chan struct{})
	go controller.Run(stop)
	log.Println("Started Watching Namespaces")
	<-done
	// if we have made it past done close the stop channel
	// to tell the controller to stop as well.
	close(stop)
	log.Println("Stopped Watching Namespaces")

}

// Deployments - Emit events on Deployment changes
func Deployments(client *kubernetes.Clientset, config config.Config, done chan bool) {
	watchlist := cache.NewListWatchFromClient(client.AppsV1beta2().RESTClient(), "deployments", v1.NamespaceAll,
		fields.Everything())
	_, controller := cache.NewInformer(
		watchlist,
		&v1beta2.Deployment{},
		time.Second*0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				dep := obj.(*v1beta2.Deployment)
				inv := inventory.Deployment{
					Name:            dep.ObjectMeta.Name,
					Namespace:       dep.ObjectMeta.Namespace,
					Labels:          dep.ObjectMeta.Labels,
					ReplicasDesired: *dep.Spec.Replicas,
					Event:           "created",
					Kind:            "deployment",
					UID:             NewUID(),
				}
				// sent the update to remote endpoint
				emitter.EmitChanges(inv, config.RemoteEndpoint)
			},
			DeleteFunc: func(obj interface{}) {
				dep := obj.(*v1beta2.Deployment)
				inv := inventory.Deployment{
					Name:            dep.ObjectMeta.Name,
					Namespace:       dep.ObjectMeta.Namespace,
					Labels:          dep.ObjectMeta.Labels,
					ReplicasDesired: *dep.Spec.Replicas,
					Event:           "deleted",
					Kind:            "deployment",
					UID:             NewUID(),
				}
				// sent the update to remote endpoint
				emitter.EmitChanges(inv, config.RemoteEndpoint)
			},
		},
	)

	stop := make(chan struct{})
	go controller.Run(stop)
	log.Println("Started Watching Deployments")
	<-done
	// if we have made it past done close the stop channel
	// to tell the controller to stop as well.
	close(stop)
	log.Println("Stopped Watching Deployments")
}

func Pods(client *kubernetes.Clientset, config config.Config, done chan bool) {

	watchlist := cache.NewListWatchFromClient(client.Core().RESTClient(), "pods", v1.NamespaceAll,
		fields.Everything())
	_, controller := cache.NewInformer(
		watchlist,
		&v1.Pod{},
		time.Second*0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				pod := obj.(*v1.Pod)
				inv := inventory.Pod{
					Name:      pod.ObjectMeta.Name,
					Namespace: pod.ObjectMeta.Namespace,
					Labels:    pod.ObjectMeta.Labels,
					Images:    imagesFromContainers(pod.Spec.Containers),
					Event:     "created",
					Kind:      "pod",
					UID:       NewUID(),
				}
				// sent the update to remote endpoint
				emitter.EmitChanges(inv, config.RemoteEndpoint)
			},

			DeleteFunc: func(obj interface{}) {
				pod := obj.(*v1.Pod)
				inv := inventory.Pod{
					Name:      pod.ObjectMeta.Name,
					Namespace: pod.ObjectMeta.Namespace,
					Labels:    pod.ObjectMeta.Labels,
					Images:    imagesFromContainers(pod.Spec.Containers),
					Event:     "deleted",
					Kind:      "pod",
					UID:       NewUID(),
				}
				// sent the update to remote endpoint
				emitter.EmitChanges(inv, config.RemoteEndpoint)
			},

			UpdateFunc: func(_, obj interface{}) {
				pod := obj.(*v1.Pod)
				inv := inventory.Pod{
					Name:      pod.ObjectMeta.Name,
					Namespace: pod.ObjectMeta.Namespace,
					Labels:    pod.ObjectMeta.Labels,
					Images:    imagesFromContainers(pod.Spec.Containers),
					Event:     "modified",
					Kind:      "pod",
					UID:       NewUID(),
				}
				// sent the update to remote endpoint
				emitter.EmitChanges(inv, config.RemoteEndpoint)
			},
		},
	)
	stop := make(chan struct{})
	go controller.Run(stop)
	log.Println("Started Watching Pods")
	<-done
	// if we have made it past done close the stop channel
	// to tell the controller to stop as well.
	close(stop)
	log.Println("Stopped Watching Pods")
}

func imagesFromContainers(containers []v1.Container) []string {

	images := []string{}
	for _, cont := range containers {
		images = append(images, cont.Image)
	}

	return images
}

// Copied from the Kubernetes repo: https://github.com/kubernetes/apimachinery/blob/release-1.10/pkg/util/uuid/uuid.go
func NewUID() types.UID {
	uuidLock.Lock()
	defer uuidLock.Unlock()
	result := uuid.NewUUID()
	// The UUID package is naive and can generate identical UUIDs if the
	// time interval is quick enough.
	// The UUID uses 100 ns increments so it's short enough to actively
	// wait for a new value.
	for uuid.Equal(lastUUID, result) == true {
		result = uuid.NewUUID()
	}
	lastUUID = result
	return types.UID(result.String())
}