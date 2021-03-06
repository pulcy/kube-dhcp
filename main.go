package main

import (
	"context"
	"log"
	"os"

	"github.com/ericchiang/k8s"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
)

var (
	maskAny = errors.WithStack
	options struct {
		configMapName string
	}
)

func init() {
	pflag.StringVar(&options.configMapName, "config-map", "kube-dhcp-config", "Name of ConfigMap in current namespace containing the DHCP configuration")
}

func main() {
	// Check options & env
	namespace := os.Getenv("METADATA_NAMESPACE")
	if namespace == "" {
		log.Fatal("METADATA_NAMESPACE not set\n")
	}
	nodeIP := os.Getenv("METADATA_NODE_IP")
	if nodeIP == "" {
		log.Fatal("METADATA_NODE_IP not set\n")
	}
	// Create k8s client
	client, err := k8s.NewInClusterClient()
	if err != nil {
		log.Fatal(err)
	}

	// Watch for config changes, relaunch handler on a valid change.
	ctx := context.Background()
	configChan := make(chan DHCPConfig)
	go watchForConfigChanges(ctx, client, options.configMapName, namespace, nodeIP, configChan)

	var stopFunc context.CancelFunc
	for {
		select {
		case config := <-configChan:
			// Create handler
			handler, err := NewHandler(config)
			if err != nil {
				log.Fatalf("Creating handler failed: %s\n", err)
			}
			// Stop current handler
			if stopFunc != nil {
				stopFunc()
			}
			// Prepare context for new handler
			handlerCtx, cancel := context.WithCancel(ctx)
			go func() {
				if err := handler.Run(handlerCtx); err != nil {
					log.Printf("Run failed: %v\n", err)
				}
			}()
			stopFunc = cancel
			log.Printf("Launched updated handler on %s\n", config.ServerIP)
		}
	}
}
