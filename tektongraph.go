package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	pipelineclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	triggersv1beta1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	triggersclientset "github.com/tektoncd/triggers/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	namespacesFlag := flag.String("n", "", "Kubernetes Namespaces (kommagetrennt, z.B. ns1,ns2)")
	outputDirFlag := flag.String("o", ".", "Das Verzeichnis, in dem die Dateien gespeichert werden sollen")
	flag.Parse()

	if *namespacesFlag == "" {
		log.Fatal("Der Parameter -n ist erforderlich und muss mindestens einen Namespace enthalten.")
	}

	namespaces := strings.Split(*namespacesFlag, ",")

	config, err := getKubeConfig()
	if err != nil {
		log.Fatalf("Fehler beim Laden der Kubernetes-Konfiguration: %v", err)
	}

	// Unterordner "dokumentation" im angegebenen Verzeichnis erstellen
	outputDir := filepath.Join(*outputDirFlag, "dokumentation")

	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		err := os.MkdirAll(outputDir, os.ModePerm)
		if err != nil {
			log.Fatalf("Fehler beim Erstellen des Verzeichnisses: %v", err)
		}
	}

	for _, namespace := range namespaces {
		namespace = strings.TrimSpace(namespace)
		if namespace == "" {
			continue
		}
		err := processNamespace(config, namespace, outputDir)
		if err != nil {
			log.Printf("Fehler beim Verarbeiten des Namespaces %s: %v", namespace, err)
		}
	}

	fmt.Printf("Mermaid-Diagramme und Triggerinformationen wurden im Verzeichnis %s gespeichert.\n", outputDir)
}

func getKubeConfig() (*rest.Config, error) {
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		kubeconfig = filepath.Join(os.Getenv("HOME"), ".kube", "config")
	}
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
	}
	return config, nil
}

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
		err := buildAndVisualizeMermaidForPipeline(&pipeline, namespace, triggersClient, outputDir)
		if err != nil {
			log.Printf("Fehler beim Visualisieren der Pipeline %s in Namespace %s: %v", pipeline.Name, namespace, err)
		}
	}

	for _, eventListener := range eventListeners.Items {
		err := buildAndVisualizeMermaidForEventListener(&eventListener, namespace, triggersClient, outputDir)
		if err != nil {
			log.Printf("Fehler beim Visualisieren des EventListeners %s in Namespace %s: %v", eventListener.Name, namespace, err)
		}
	}

	return nil
}

func buildAndVisualizeMermaidForPipeline(pipeline *pipelinev1.Pipeline, namespace string, triggersClient *triggersclientset.Clientset, outputDir string) error {
	// Dateiname basierend auf dem Pipeline-Namen erstellen
	fileName := fmt.Sprintf("%s_pipeline_%s.md", namespace, pipeline.Name)
	filePath := filepath.Join(outputDir, fileName)

	// Datei erstellen
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("fehler beim Erstellen der Datei: %v", err)
	}
	defer file.Close()

	// Mermaid-Diagramm erstellen
	var mermaidDiagram strings.Builder
	mermaidDiagram.WriteString("```mermaid\n")
	mermaidDiagram.WriteString("graph TD;\n")

	// Füge Pipeline als Knoten hinzu
	pipelineNode := fmt.Sprintf("Pipeline_%s[Pipeline: %s]", pipeline.Name, pipeline.Name)
	mermaidDiagram.WriteString(pipelineNode + ";\n")

	// Füge Tasks als Knoten hinzu und verknüpfe sie mit der Pipeline
	for _, task := range pipeline.Spec.Tasks {
		taskNode := fmt.Sprintf("Task_%s[Task: %s]", task.Name, task.Name)
		mermaidDiagram.WriteString(taskNode + ";\n")
		mermaidDiagram.WriteString(fmt.Sprintf("%s --> %s;\n", pipelineNode, taskNode))

		// runAfter dependencies
		for _, dep := range task.RunAfter {
			depNode := fmt.Sprintf("Task_%s", dep)
			mermaidDiagram.WriteString(fmt.Sprintf("%s --> %s;\n", depNode, taskNode))
		}

		// params dependencies
		for _, param := range task.Params {
			if param.Value.Type == pipelinev1.ParamTypeString {
				matches := extractTaskReferences(param.Value.StringVal)
				for _, match := range matches {
					fromNode := fmt.Sprintf("Task_%s", match)
					mermaidDiagram.WriteString(fmt.Sprintf("%s --> %s;\n", fromNode, taskNode))
				}
			}
		}
	}

	// EventListener, TriggerBinding und TriggerTemplate hinzufügen
	err = addTriggersToMermaid(&mermaidDiagram, pipeline.Name, namespace, triggersClient)
	if err != nil {
		return err
	}

	mermaidDiagram.WriteString("```\n")

	// Schreibe das Diagramm in die Datei
	_, err = file.WriteString(mermaidDiagram.String())
	if err != nil {
		return err
	}

	fmt.Printf("Mermaid-Diagramm für Pipeline '%s' wurde in der Datei %s gespeichert.\n", pipeline.Name, filePath)
	return nil
}

// Hilfsfunktion, um redundante Kanten (Pfeile) im Mermaid-Diagramm zu vermeiden
func addEdgeIfNotExists(mermaidDiagram *strings.Builder, fromNode, toNode string, existingEdges map[string]bool) {
	edge := fmt.Sprintf("%s --> %s", fromNode, toNode)
	if !existingEdges[edge] {
		mermaidDiagram.WriteString(edge + ";\n")
		existingEdges[edge] = true
	}
}

func buildAndVisualizeMermaidForEventListener(eventListener *triggersv1beta1.EventListener, namespace string, _ *triggersclientset.Clientset, outputDir string) error {
	// Dateiname basierend auf dem EventListener-Namen erstellen
	fileName := fmt.Sprintf("%s_eventlistener_%s.md", namespace, eventListener.Name)
	filePath := filepath.Join(outputDir, fileName)

	// Datei erstellen
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("fehler beim Erstellen der Datei: %v", err)
	}
	defer file.Close()

	// Mermaid-Diagramm erstellen
	var mermaidDiagram strings.Builder
	mermaidDiagram.WriteString("```mermaid\n")
	mermaidDiagram.WriteString("graph TD;\n")

	// Füge den EventListener als Knoten hinzu
	eventListenerNode := fmt.Sprintf("EventListener_%s[EventListener: %s]", eventListener.Name, eventListener.Name)
	mermaidDiagram.WriteString(eventListenerNode + ";\n")

	// Vermeide doppelte Kanten durch Speicherung bereits existierender Kanten
	existingEdges := make(map[string]bool)

	// Verarbeite die Triggers und TriggerTemplates
	for _, trigger := range eventListener.Spec.Triggers {
		triggerTemplateName := trigger.Template.Ref
		_, err = file.WriteString(fmt.Sprintf("### Trigger %s:\n\n", trigger.Name))
		if err != nil {
			return err
		}

		// Interceptors
		if trigger.Interceptors != nil {
			for _, interceptor := range trigger.Interceptors {
				if interceptor.Ref.Name != "" {
					_, err = file.WriteString(fmt.Sprintf("- **Interceptor:** %s\n", interceptor.Ref.Name))
					if err != nil {
						return err
					}
				}

				// Interceptor-Parameter (speziell für CEL-Filter)
				if interceptor.Params != nil {
					for _, param := range interceptor.Params {
						if param.Value.Raw != nil {
							decodedValue, err := decodeRawJSON(param.Value.Raw)
							if err != nil {
								_, err = file.WriteString(fmt.Sprintf("  - **%s:** %s (unparsable)\n", param.Name, string(param.Value.Raw)))
								if err != nil {
									return err
								}
							} else {
								_, err = file.WriteString(fmt.Sprintf("  - **%s:** %s\n", param.Name, decodedValue))
								if err != nil {
									return err
								}
							}
						}
					}
				}
			}
		}

		// TriggerBindings hinzufügen
		for _, binding := range trigger.Bindings {
			triggerBindingName := binding.Ref
			if triggerBindingName != "" {
				triggerBindingNode := fmt.Sprintf("TriggerBinding_%s[TriggerBinding: %s]", triggerBindingName, triggerBindingName)
				mermaidDiagram.WriteString(triggerBindingNode + ";\n")

				// Verknüpfe den TriggerBinding-Knoten mit dem TriggerTemplate-Knoten
				if triggerTemplateName != nil && *triggerTemplateName != "" {
					triggerTemplateNode := fmt.Sprintf("TriggerTemplate_%s[TriggerTemplate: %s]", *triggerTemplateName, *triggerTemplateName)
					addEdgeIfNotExists(&mermaidDiagram, triggerBindingNode, triggerTemplateNode, existingEdges)
				}

				// Verknüpfe den EventListener-Knoten mit dem TriggerBinding-Knoten
				addEdgeIfNotExists(&mermaidDiagram, eventListenerNode, triggerBindingNode, existingEdges)

				_, err = file.WriteString(fmt.Sprintf("- **Binding:** %s\n", triggerBindingName))
				if err != nil {
					return err
				}
			}
		}

		// Template hinzufügen
		if *triggerTemplateName != "" {
			_, err = file.WriteString(fmt.Sprintf("- **Template:** %s\n", *triggerTemplateName))
			if err != nil {
				return err
			}
		}
	}

	mermaidDiagram.WriteString("```\n")

	// Schreibe das Diagramm in die Datei
	_, err = file.WriteString(mermaidDiagram.String())
	if err != nil {
		return err
	}

	fmt.Printf("Mermaid-Diagramm für EventListener '%s' wurde in der Datei %s gespeichert.\n", eventListener.Name, filePath)
	return nil
}

// Hilfsfunktion zum Decodieren von rohem JSON
func decodeRawJSON(raw []byte) (string, error) {
	var decoded interface{}
	err := json.Unmarshal(raw, &decoded)
	if err != nil {
		return "", err
	}

	// Konvertiere das decodierte JSON in eine menschenlesbare Form
	prettyJSON, err := json.MarshalIndent(decoded, "", "  ")
	if err != nil {
		return "", err
	}

	// Unicode-Escape-Sequenzen für das &-Zeichen direkt in & umwandeln
	decodedString := strings.ReplaceAll(string(prettyJSON), `\u0026`, "&")

	return decodedString, nil
}

func addTriggersToMermaid(mermaidDiagram *strings.Builder, _, namespace string, triggersClient *triggersclientset.Clientset) error {
	eventListeners, err := triggersClient.TriggersV1beta1().EventListeners(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, eventListener := range eventListeners.Items {
		for _, trigger := range eventListener.Spec.Triggers {
			triggerTemplateName := trigger.Template.Ref
			if triggerTemplateName == nil || *triggerTemplateName == "" {
				continue
			}
			triggerTemplate, err := triggersClient.TriggersV1beta1().TriggerTemplates(namespace).Get(context.Background(), *triggerTemplateName, metav1.GetOptions{})
			if err != nil {
				continue
			}
			for _, res := range triggerTemplate.Spec.ResourceTemplates {
				var unstructuredObj unstructured.Unstructured
				if err := json.Unmarshal(res.Raw, &unstructuredObj); err != nil {
					continue
				}
				if unstructuredObj.GetKind() == "PipelineRun" {
					eventListenerNode := fmt.Sprintf("EventListener_%s[EventListener: %s]", eventListener.Name, eventListener.Name)
					triggerTemplateNode := fmt.Sprintf("TriggerTemplate_%s[TriggerTemplate: %s]", triggerTemplate.Name, triggerTemplate.Name)
					mermaidDiagram.WriteString(eventListenerNode + ";\n")
					mermaidDiagram.WriteString(triggerTemplateNode + ";\n")
					mermaidDiagram.WriteString(fmt.Sprintf("%s --> %s;\n", eventListenerNode, triggerTemplateNode))

					for _, binding := range trigger.Bindings {
						triggerBindingName := binding.Ref
						if triggerBindingName == "" {
							continue
						}
						triggerBindingNode := fmt.Sprintf("TriggerBinding_%s[TriggerBinding: %s]", triggerBindingName, triggerBindingName)
						mermaidDiagram.WriteString(triggerBindingNode + ";\n")
						mermaidDiagram.WriteString(fmt.Sprintf("%s --> %s;\n", eventListenerNode, triggerBindingNode))
						mermaidDiagram.WriteString(fmt.Sprintf("%s --> %s;\n", triggerBindingNode, triggerTemplateNode))
					}
				}
			}
		}
	}
	return nil
}

func extractTaskReferences(value string) []string {
	regex := regexp.MustCompile(`\$\(\s*tasks\.(\w+)\.`)
	matches := regex.FindAllStringSubmatch(value, -1)
	var taskNames []string
	for _, match := range matches {
		if len(match) > 1 {
			taskNames = append(taskNames, match[1])
		}
	}
	return taskNames
}
