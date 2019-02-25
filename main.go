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
)

var specifiedNamespace, namespace string
var allNamespaces bool

func init() {
	flag.BoolVar(&allNamespaces, "a", false, "all namespaces")
	flag.StringVar(&specifiedNamespace, "n", "", "specify namespace")
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

	config, err := dcc.ClientConfig()
	if err != nil {
		log.Fatalf("error creating client config: %v", err)
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("error creating clientset: %v", err)
	}
	for {
		if specifiedNamespace != "" {
			nsList, err := clientset.CoreV1().Namespaces().List(metav1.ListOptions{})
			if err != nil {
				log.Fatalf("error listing namespaces: %v", err)
			}
			foundNamespace := false
			for _, n := range nsList.Items {
				if n.Name == specifiedNamespace {
					foundNamespace = true
					namespace = specifiedNamespace
				}
			}
			if !foundNamespace {
				log.Fatalf("error finding requested namespace: %s", specifiedNamespace)
			}
		} else if allNamespaces {
			namespace = ""
		} else {
			namespace = defaultNamespace
		}
		pods, err := clientset.CoreV1().Pods(namespace).List(metav1.ListOptions{})
		if err != nil {
			log.Fatalf("error listing pods: %v", err)
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
