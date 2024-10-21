# tektongraph
The attempt to read out all tecton artefacts from the K8s cluster and display them graphically.

Um das Programm auszuführen, folgen Sie diesen Schritten:

## Projektstruktur

- `main.go`
- `kubeconfig.go`
- `namespace.go`
- `pipeline.go`
- `eventlistener.go`
- `utils.go`

## Go-Modul initialisieren

Öffnen Sie ein Terminal im Verzeichnis, in dem sich Ihre Go-Dateien befinden, und initialisieren Sie ein neues Go-Modul:

```bash
go mod init main
```

## Abhängigkeiten verwalten

Führen Sie den folgenden Befehl aus, um alle benötigten Abhängigkeiten herunterzuladen:

```bash
go mod tidy
```

Dieser Befehl liest Ihre Go-Dateien, identifiziert die benötigten Pakete und aktualisiert die `go.mod` und `go.sum` Dateien entsprechend.

## Programm kompilieren

Um das Programm zu kompilieren, verwenden Sie:

```bash
go build
```

Dieser Befehl erstellt eine ausführbare Datei in Ihrem aktuellen Verzeichnis. Unter Linux und macOS wird die Datei standardmäßig nach dem Verzeichnis benannt, in dem Sie sich befinden. Unter Windows wird sie `meinprogramm.exe` heißen.

## Programm ausführen

Führen Sie das Programm mit den erforderlichen Flags aus. Sie müssen mindestens den `-n` Flag für die Namespaces angeben.

**Syntax:**

```bash
./meinprogramm -n namespace1,namespace2 -o /pfad/zum/ausgabeverzeichnis
```

- **`-n`**: (Erforderlich) Eine kommagetrennte Liste der Kubernetes-Namespaces, z.B. `-n "dev,test"`.
- **`-o`**: (Optional) Das Verzeichnis, in dem die Ausgabedateien gespeichert werden sollen. Wenn nicht angegeben, wird das aktuelle Verzeichnis verwendet.

**Beispiel:**

Angenommen, die ausführbare Datei heißt `meinprogramm`, und Sie möchten die Namespaces `dev` und `test` verarbeiten und die Ausgaben im Verzeichnis `/home/user/ausgaben` speichern:

```bash
./meinprogramm -n dev,test -o /home/user/ausgaben
```

## Hinweise zur Ausführung

- **Kubernetes-Konfiguration:** Stellen Sie sicher, dass Ihre Kubernetes-Konfigurationsdatei (`kubeconfig`) korrekt ist. Das Programm sucht standardmäßig nach der Umgebungsvariable `KUBECONFIG`. Wenn diese nicht gesetzt ist, wird `~/.kube/config` verwendet.

- **Berechtigungen:** Sie benötigen ausreichende Berechtigungen, um auf die Tekton-Ressourcen (`Pipelines`, `EventListeners` usw.) in den angegebenen Namespaces zuzugreifen.

- **Ausgabeverzeichnis:** Die generierten Mermaid-Diagramme und Informationen werden im Unterverzeichnis `dokumentation` des angegebenen Ausgabeverzeichnisses gespeichert.

## Beispielhafte Ausgabe

Nach erfolgreicher Ausführung sollten Sie Meldungen wie die folgenden sehen:

```
Mermaid-Diagramm für Pipeline 'pipeline-name' wurde in der Datei /home/user/ausgaben/dokumentation/namespace_pipeline_pipeline-name.md gespeichert.
Mermaid-Diagramm für EventListener 'eventlistener-name' wurde in der Datei /home/user/ausgaben/dokumentation/namespace_eventlistener_eventlistener-name.md gespeichert.
```

Diese Dateien enthalten die generierten Mermaid-Diagramme und können mit einem geeigneten Markdown-Viewer angezeigt werden.

## Zusätzliche Informationen

- **Fehlerbehandlung:** Wenn das Programm Fehler ausgibt, prüfen Sie, ob die angegebenen Namespaces existieren und ob Sie die erforderlichen Berechtigungen besitzen.

- **Debugging:** Sie können `fmt.Printf` oder `log.Printf` Statements im Code verwenden, um zusätzliche Debugging-Informationen auszugeben.

## Zusammenfassung der Schritte

1. Alle Go-Dateien in ein Verzeichnis legen.
2. Go-Modul initialisieren mit `go mod init`.
3. Abhängigkeiten mit `go mod tidy` herunterladen.
4. Programm mit `go build` kompilieren.
5. Programm mit den erforderlichen Flags ausführen.


