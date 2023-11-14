package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/slack-go/slack"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"log"
	"os"
	"time"
)

var (
	channelId = flag.String("slack-channel-id", "C05SP2XRK7G", "Slack channel for announcements")
)

func main() {
	flag.Parse()

	slackToken, found := os.LookupEnv("SLACK_TOKEN")
	if !found {
		log.Fatal("Please set SLACK_TOKEN")
	}
	chat := slack.New(slackToken)

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

	for {
		watch, err := cs.CoreV1().Pods("").Watch(context.Background(), metav1.ListOptions{
			LabelSelector: "crashloopbackon = true",
		})

		if err != nil {
			log.Fatalf("Failed to set up watch: %v", err)
		}

	watchloop:
		for evt := range watch.ResultChan() {
			pod := evt.Object.(*v1.Pod)
			switch evt.Type {
			case "ADDED", "MODIFIED":
				for _, status := range pod.Status.ContainerStatuses {
					if w := status.State.Waiting; w != nil {
						if w.Reason == "CrashLoopBackOff" {
							log.Printf("Pod %v/%v is in CrashLoopBackOff, deleting", pod.Namespace, pod.Name)
							for {
								if _, _, err = chat.PostMessage(*channelId, slack.MsgOptionText(
									fmt.Sprintf("Deleting crashlooping pod `%s/%s`", pod.Namespace, pod.Name), false)); err != nil {
									var rlError *slack.RateLimitedError
									if errors.As(err, &rlError) {
										log.Printf("Rate limited, sleeping %s", rlError.RetryAfter)
										time.Sleep(rlError.RetryAfter)
										break
									}
									log.Fatalf("Could not post Slack message: %v", err)
								} else {
									break
								}
							}
							err := cs.CoreV1().Pods(pod.Namespace).Delete(context.Background(), pod.Name, metav1.DeleteOptions{})
							if err != nil {
								log.Fatalf("Could not delete %v/%v: %v", pod.Namespace, pod.Name, err)
							}
							continue watchloop
						}
					}
				}
			}
		}

		log.Printf("Now the watch has ended. Re-watching.")
	}
}
