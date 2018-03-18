package main

import (
	"context"
	"log"
	"os"

	"github.com/ericchiang/k8s"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"

	"github.com/pulcy/kube-dhcp/pkg/registry"
	"github.com/pulcy/kube-dhcp/pkg/service"
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
	name := os.Getenv("METADATA_NAME")
	if name == "" {
		log.Fatal("METADATA_NAME not set\n")
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
	registry := registry.NewKubeLeaseRegistry(client)

	// Watch for config changes, relaunch handler on a valid change.
	ctx := context.Background()
	configChan := make(chan service.DHCPConfig)
	go service.WatchForConfigChanges(ctx, client, options.configMapName, namespace, nodeIP, configChan)
	leaderChan := make(chan bool, 32)
	go service.PerformLeaderElection(ctx, client, namespace, name, leaderChan)

	log.Printf("Starting kube-dhcp on %s\n", nodeIP)

	var stopFunc context.CancelFunc
	isLeader := false
	var config *service.DHCPConfig
	for {
		select {
		case l := <-leaderChan:
			if isLeader == l {
				continue
			}
			isLeader = l
		case cfg := <-configChan:
			config = &cfg
		}
		// Stop current handler (if any)
		if stopFunc != nil {
			stopFunc()
			stopFunc = nil
		}
		// Create handler if leader & have config
		if isLeader && config != nil {
			handler, err := service.NewHandler(*config, registry)
			if err != nil {
				log.Fatalf("Creating handler failed: %s\n", err)
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
