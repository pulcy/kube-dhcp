# Kube-DHCP; DHCP4 server running in Kubernetes

Status: Early Development

## Building

```bash
docker build -f Dockerfile.build -t kube-dhcp .
```

## Installation

```bash
kubectl apply -f deployment.yaml
```

## Configuration

Edit configuration file (e.g. example-config.yaml) and deploy it.

```bash
kubectl apply -f example-config.yaml
```