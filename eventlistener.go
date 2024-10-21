package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	triggersv1beta1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	triggersclientset "github.com/tektoncd/triggers/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

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
		if triggerTemplateName != nil && *triggerTemplateName != "" {
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
