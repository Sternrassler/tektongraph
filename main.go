package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
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
