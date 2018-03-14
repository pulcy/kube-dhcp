package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ericchiang/k8s"
	corev1 "github.com/ericchiang/k8s/apis/core/v1"
	"github.com/ghodss/yaml"
)

// watchForConfigChanges starts a process that continues to watch for configuration
// changes until the given context is canceled.
func watchForConfigChanges(ctx context.Context, cli *k8s.Client, configMapName, namespace, nodeIP string, configChan chan DHCPConfig) {
	// Load config, then watch for changes
	var configMap corev1.ConfigMap
	watcher, err := cli.Watch(ctx, namespace, &configMap)
	if err != nil {
		log.Fatalf("Failed to create ConfigMap watcher")
	}
	defer watcher.Close()
	for {
		cm := new(corev1.ConfigMap)
		eventType, err := watcher.Next(cm)
		if err != nil {
			log.Fatalf("Failed to watch next event: %v", err)
		}
		fmt.Println(eventType, *cm.Metadata.Name)
		if cm.Metadata.GetName() != configMapName {
			continue
		}
		// Parse the config map
		data, found := cm.GetData()["config"]
		if !found {
			log.Printf("ConfigMap is missing a `config` data item\n")
			continue
		}
		var config DHCPConfig
		if err := yaml.Unmarshal([]byte(data), &config); err != nil {
			log.Printf("Failed to parse ConfigMap data: %v\n", err)
			continue
		}
		if err := config.Validate(nodeIP); err != nil {
			log.Printf("ConfigMap data is not valid: %v\n", err)
			continue
		}
		// We found a valid config
		configChan <- config
	}
}
