package kubernetes

import (
	"context"
	"log"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	toolsWatch "k8s.io/client-go/tools/watch"
)

type KubernetesClient struct {
	config    *rest.Config
	clientset *kubernetes.Clientset
}

func New() (*KubernetesClient, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return &KubernetesClient{
		config:    config,
		clientset: clientset,
	}, nil
}

func (k *KubernetesClient) WatchSecret(namespace string, secret string) {
	watchFunc := func(options metav1.ListOptions) (watch.Interface, error) {
		timeOut := int64(60)
		return k.clientset.CoreV1().Secrets(namespace).Watch(context.Background(), metav1.ListOptions{
			FieldSelector: "metadata.name=" + secret, TimeoutSeconds: &timeOut,
		})
	}
	watcher, err := toolsWatch.NewRetryWatcher("1", &cache.ListWatch{WatchFunc: watchFunc})
	if err != nil {
		return
	}

	for event := range watcher.ResultChan() {
		switch event.Type {
		case watch.Modified:
			log.Printf("[watch] secret %s had event %s - exiting to reload", secret, event.Type)
			os.Exit(0)
		case watch.Deleted:
			log.Printf("[watch] secret %s had event %s - exiting to reload", secret, event.Type)
			os.Exit(0)
		}
	}
}
