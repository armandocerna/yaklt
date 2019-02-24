package main

import (
	"bufio"
	"flag"
	"fmt"
	. "github.com/logrusorgru/aurora"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"time"
	//
	// Uncomment to load all auth plugins
	// _ "k8s.io/client-go/plugin/pkg/client/auth"
	//
	// Or uncomment to load specific auth plugins
	// _ "k8s.io/client-go/plugin/pkg/client/auth/azure"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/openstack"
)

var namespace string
var allNamespaces bool

func init() {
	flag.BoolVar(&allNamespaces, "a", false, "all namespaces")
	flag.StringVar(&namespace, "n", "", "specify namespace)")
}

func main() {

	var kubeconfig *string
	knownPods := make(map[string]bool)
	podColors := make(map[string]Color)
	availableColors := []Color{RedFg, BlueFg, CyanFg, MagentaFg, GreenFg, BrownFg}
	rand.Seed(time.Now().Unix())
	flag.Parse()

	if home := homeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}
	oc, err := clientcmd.LoadFromFile(*kubeconfig)
	dcc := clientcmd.NewDefaultClientConfig(*oc, &clientcmd.ConfigOverrides{})
	defaultNamespace, ok, err := dcc.Namespace()
	if err != nil {
		log.Println("namespace error: %v", err)
		if !ok {
			log.Println("namespace lookup error: %v", err)
			defaultNamespace = "default"
		}
	}

	//dc := clientcmd.DirectClientConfig{}
	//dc.Namespace()kk
	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	for {
		if (namespace != "" || foundNamespace) {
			fmt.Println("SPECIFY NAMESPACE")
			nsList := clientset.CoreV1().Namespaces().List(metav1.ListOptions{})
			foundNamespace := false
			for _, n := range nsList {
				if n == namespace { foundNamespace = true }
			}
		} else if allNamespaces {
			namespace = ""
		} else {
			namespace = defaultNamespace
		}
		pods, err := clientset.CoreV1().Pods(namespace).List(metav1.ListOptions{})
		if err != nil {
			panic(err.Error())
		}
		for _, p := range pods.Items {
			if !knownPods[p.Name] && p.Status.Phase == v1.PodRunning  {
				go func() {
					pod := p
					for _, c := range pod.Spec.Containers {

						// Set random pod color
						if _, ok := podColors[pod.Name]; !ok {
							podColors[pod.Name] = availableColors[rand.Intn(len(availableColors))]
						}

						color := podColors[pod.Name]
						req := clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &v1.PodLogOptions{Container:c.Name, Follow:true})
						logs, err := req.Stream()
						if err != nil {
							log.Fatalf("error opening stream %v", err)
						}
						scanner := bufio.NewScanner(logs)
						for scanner.Scan() {
							l := fmt.Sprintf("%s(%s) - %s\n", pod.Name, c.Name, scanner.Text())
							fmt.Println(Colorize(l, color))
						}

					}
				}()
			}
			time.Sleep(1 * time.Second)
			knownPods[p.Name] = true
		}
		//time.Sleep(30 * time.Second)
	}
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
