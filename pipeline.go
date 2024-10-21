package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	triggersclientset "github.com/tektoncd/triggers/pkg/client/clientset/versioned"
)

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

		// Extrahiere WhenExpressions und füge sie als Markdown-Text hinzu
		for _, whenExpression := range task.When {
			// mermaidDiagram.WriteString(fmt.Sprintf("Note over %s: %s\n", taskNode, conditionText))
			renderCondition(task.Name, whenExpression) // Ausgabe des Traces
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

func renderCondition(taskName string, whenExpression pipelinev1.WhenExpression) {
	fmt.Printf("\tTask: %s\n", taskName)
	fmt.Printf("\t\tTrace: When Input: %s\n", whenExpression.Input)
	fmt.Printf("\t\tTrace: When Operator: %s\n", whenExpression.Operator)
	for _, value := range whenExpression.Values {
		fmt.Printf("\t\t\tTrace: When Value: %s\n", value)
	}
}

// Extrahiert Task-Referenzen aus einem Parameterwert
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

// Hilfsfunktion, um redundante Kanten (Pfeile) im Mermaid-Diagramm zu vermeiden
func addEdgeIfNotExists(mermaidDiagram *strings.Builder, fromNode, toNode string, existingEdges map[string]bool) {
	edge := fmt.Sprintf("%s --> %s", fromNode, toNode)
	if !existingEdges[edge] {
		mermaidDiagram.WriteString(edge + ";\n")
		existingEdges[edge] = true
	}
}
