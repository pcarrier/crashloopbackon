package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"log"
	"os"
	"strconv"
)

func main() {
	flag.Parse()

	ctx := datadog.NewDefaultContext(context.Background())
	ddEvents := datadogV1.NewEventsApi(datadog.NewAPIClient(datadog.NewConfiguration()))

	configPath, found := os.LookupEnv("KUBECONFIG")
	if !found {
		dir, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("Could not get home directory: %v", err)
		}
		candidateConfigPath := dir + "/.kube/config"
		if _, err := os.Stat(candidateConfigPath); err == nil {
			configPath = candidateConfigPath
		}
	}
	config, err := clientcmd.BuildConfigFromFlags("", configPath)
	if err != nil {
		log.Fatalf("Could not build kubeconfig: %v", err)
	}
	cs := kubernetes.NewForConfigOrDie(config)

	watch, err := cs.CoreV1().Pods("").Watch(ctx, metav1.ListOptions{})

watchloop:
	for evt := range watch.ResultChan() {
		pod := evt.Object.(*v1.Pod)
		if v, found := pod.Labels["crashloopbackon"]; found {
			if b, _ := strconv.ParseBool(v); !b {
				continue watchloop
			}
			switch evt.Type {
			case "ADDED", "MODIFIED":
				for _, status := range pod.Status.ContainerStatuses {
					if w := status.State.Waiting; w != nil {
						if w.Reason == "CrashLoopBackOff" {
							if err := handle(pod, cs, ctx, ddEvents, status.Name); err != nil {
								log.Fatalf("Could not handle %v: %v", pod, err)
							}
							continue watchloop
						}
					}
				}
			}
		}
	}
}

func handle(pod *v1.Pod, cs *kubernetes.Clientset, ctx context.Context, events *datadogV1.EventsApi, containerName string) error {
	tags := []string{
		fmt.Sprintf("namespace:%s", pod.Namespace),
		fmt.Sprintf("pod_name:%s", pod.Name),
		fmt.Sprintf("workload:%s", containerName),
	}
	if pteam, found := pod.Labels["com.apollographql/primaryTeam"]; found {
		tags = append(tags, fmt.Sprintf("team:%s", pteam))
	}
	if _, _, err := events.CreateEvent(ctx, datadogV1.EventCreateRequest{
		Title: fmt.Sprintf("crashlooping %s/%s (%s)", containerName, pod.Namespace, pod.Name),
		Text:  fmt.Sprintf("A pod entered `CrashLoopBackOff` and was restarted by `crashloopbackon`(https://github.com/pcarrier/crashloopbackon).\n"),
		Tags:  tags,
	}); err != nil {
		return fmt.Errorf("datadog submission failed: %w", err)
	}

	if err := cs.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{}); err != nil {
		return fmt.Errorf("pod deletion failed: %w", err)
	}

	return nil
}
