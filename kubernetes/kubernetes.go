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
	"k8s.io/client-go/tools/clientcmd"
	toolsWatch "k8s.io/client-go/tools/watch"
)

type KubernetesClient struct {
	config    *rest.Config
	clientset *kubernetes.Clientset
}

func New() (*KubernetesClient, error) {
	config, err := clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
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

func (k *KubernetesClient) WatchSecret(namespace string, secret string) error {
	watchFunc := func(options metav1.ListOptions) (watch.Interface, error) {
		timeOut := int64(60)
		return k.clientset.CoreV1().Secrets(namespace).Watch(context.Background(), metav1.ListOptions{TimeoutSeconds: &timeOut})
	}

	watcher, err := toolsWatch.NewRetryWatcher("1", &cache.ListWatch{WatchFunc: watchFunc})
	if err != nil {
		return err
	}

	for event := range watcher.ResultChan() {
		switch event.Type {
		case watch.Modified:
		case watch.Bookmark:
		case watch.Error:
		case watch.Deleted:
		case watch.Added:
			return nil
		}
	}
	return nil // return and error here?
}

func (k *KubernetesClient) ListNamespaces() error {
	list, err := k.clientset.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})

	if err != nil {
		return err
	}

	for _, item := range list.Items {
		log.Printf(item.Name)
	}
	return nil
}
