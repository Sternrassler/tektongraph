package main

import (
	"context"
	"fmt"

	pipelineclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	triggersclientset "github.com/tektoncd/triggers/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

func processNamespace(config *rest.Config, namespace string, outputDir string) error {
	pipelineClient, err := pipelineclientset.NewForConfig(config)
	if err != nil {
		return err
	}

	triggersClient, err := triggersclientset.NewForConfig(config)
	if err != nil {
		return err
	}

	pipelines, err := pipelineClient.TektonV1().Pipelines(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return err
	}

	eventListeners, err := triggersClient.TriggersV1beta1().EventListeners(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return err
	}

	if len(pipelines.Items) == 0 && len(eventListeners.Items) == 0 {
		fmt.Printf("Keine relevanten Ressourcen im Namespace '%s' gefunden.\n", namespace)
		return nil
	}

	for _, pipeline := range pipelines.Items {
		err := buildAndVisualizeMermaidForPipeline(&pipeline, namespace, outputDir)
		if err != nil {
			fmt.Printf("Fehler beim Visualisieren der Pipeline %s in Namespace %s: %v\n", pipeline.Name, namespace, err)
		}
	}

	for _, eventListener := range eventListeners.Items {
		err := buildAndVisualizeMermaidForEventListener(&eventListener, namespace, triggersClient, outputDir)
		if err != nil {
			fmt.Printf("Fehler beim Visualisieren des EventListeners %s in Namespace %s: %v\n", eventListener.Name, namespace, err)
		}
	}

	return nil
}
