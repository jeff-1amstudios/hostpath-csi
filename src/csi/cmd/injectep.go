package main

import (
	"context"
	"fmt"
	"os"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	if len(os.Args) != 4 {
		fmt.Print(os.Args)
		panic("expected three args")
	}
	podNamespace := os.Args[1]
	podName := os.Args[2]
	kubeconfigPath := os.Args[3]

	// Create the client
	client, err := getKubernetesClients(kubeconfigPath)
	if err != nil {
		panic(fmt.Errorf("could not create client: %w", err))
	}
	ctx := context.Background()

	// Get the Pod
	pod, err := client.CoreV1().Pods(podNamespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		panic(fmt.Errorf("could not get pod: %w", err))
	}

	// Add a new ephemeral container
	trueValue := true
	ephemeralContainer := corev1.EphemeralContainer{
		EphemeralContainerCommon: corev1.EphemeralContainerCommon{
			Name:  "debug",
			Image: "busybox",
			TTY:   true,
			SecurityContext: &corev1.SecurityContext{
				Privileged:               &trueValue,
				AllowPrivilegeEscalation: &trueValue,
			},
		},
	}
	pod.Spec.EphemeralContainers = append(pod.Spec.EphemeralContainers, ephemeralContainer)
	pod, err = client.CoreV1().Pods(pod.Namespace).UpdateEphemeralContainers(ctx, pod.Name, pod, metav1.UpdateOptions{})
	if err != nil {
		panic(fmt.Errorf("could not add ephemeral container: %w", err))
	}
}

func getKubernetesClients(path string) (kubernetes.Interface, error) {
	cfg, err := clientcmd.BuildConfigFromFlags("", path)
	if err != nil {
		return nil, err
	}
	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	return client, nil
}
