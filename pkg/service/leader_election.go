package service

import (
	"context"
	"log"
	"time"

	"github.com/ericchiang/k8s"
	lock "github.com/pulcy/kube-lock/k8s/ericchiang"
)

// PerformLeaderElection starts a process that tries to become leader.
// It updates leader update changes (true=leader, false=not-leader)
// in the given channel, until the given context is canceled.
func PerformLeaderElection(ctx context.Context, cli *k8s.Client, namespace, podName string, leaderChan chan bool) {
	ttl := time.Second * 30
	l, err := lock.NewNamespaceLock(namespace, cli, "", podName, ttl)
	if err != nil {
		log.Fatalf("Failed to create lock: %v\n", err)
	}

	isLeader := false
	for {
		delay := time.Second * 2
		if err := l.Acquire(); err == nil {
			if !isLeader {
				log.Println("Leader lock acquired")
				isLeader = true
				leaderChan <- true
			}
			delay = ttl / 2
		} else {
			if isLeader {
				log.Println("Leader lock lost")
				isLeader = false
				leaderChan <- false
			}
		}

		select {
		case <-time.After(delay):
			// Continue
		case <-ctx.Done():
			if isLeader {
				if err := l.Release(); err != nil {
					log.Printf("Failed to release leader lock: %v\n", err)
				}
			}
		}
	}
}
