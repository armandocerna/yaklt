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

func main() {
	var kubeconfig *string
	knownPods := make(map[string]bool)
	availableColors := []Color{RedFg, BlueFg, CyanFg, MagentaFg, GreenFg, BrownFg}
	podColors := make(map[string]Color)
	rand.Seed(time.Now().Unix())

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

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	for {
		pods, err := clientset.CoreV1().Pods("").List(metav1.ListOptions{})
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
