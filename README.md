# Faraway Edge

A dynamic configuration control plane for Envoy proxies, enabling flexible HTTP/HTTPS traffic routing based on domain names.

## Purpose

Faraway Edge acts as a central configuration server for Envoy proxy instances, allowing you to define high-level routing rules that are automatically translated into Envoy configurations. Instead of manually configuring each Envoy proxy, you define your routing logic once, and all connected proxies receive updates automatically.

## Key Features

- **Dynamic Configuration**: Proxies automatically receive configuration updates without restarts
- **Domain-Based Routing**: Route HTTP and HTTPS traffic based on domain names to different backend services
- **Kubernetes Integration**: Automatically discover and configure routing from Kubernetes Ingress resources
- **Automatic Configuration Sync**: Connected proxies stay synchronized with the latest routing rules
- **Health Monitoring**: Built-in health check and readiness endpoints for integration with orchestration platforms
- **Secure Communication**: Optional token-based authentication to secure the control plane
- **Configuration Validation**: Validates routing configurations before applying them to prevent errors
- **Graceful Shutdown**: Handles termination signals cleanly to ensure in-flight requests complete

## Installation

### Docker

```bash
docker pull ghcr.io/paragor/faraway-edge:latest
```

### Helm (Kubernetes)

```bash
helm repo add faraway-edge https://paragor.github.io/faraway-edge
helm install faraway-edge faraway-edge/faraway-edge
```

**Kubernetes Configuration:**

The Helm chart supports automatic Ingress discovery. Configure in `values.yaml`:

```yaml
k8sDiscovery:
  enabled: true
  clusterName: "k8s-local"
  ingressClasses: []  # Empty = watch all ingress classes, or specify: ["nginx", "traefik"]

xdsAuthToken: "your-secret-token"  # Optional authentication
```

The chart automatically creates RBAC permissions to watch Ingress resources cluster-wide.

### Binary

Download from [GitHub Releases](https://github.com/paragor/faraway-edge/releases) or build from source:

```bash
go build -o faraway-edge .
```

## Quick Start

### Using Docker

```bash
# Generate example configuration
docker run --rm ghcr.io/paragor/faraway-edge:latest example > config.json

# Run the control plane
docker run -v $(pwd)/config.json:/config.json -p 18000:18000 -p 8080:8080 \
  ghcr.io/paragor/faraway-edge:latest run --static-path /config.json
```

### Using Binary

```bash
# Generate example configuration
./faraway-edge example > config.json

# Run the control plane
./faraway-edge run --static-path config.json
```

Configure your Envoy proxies to connect to the control plane at `localhost:18000`

## Usage

### Basic Usage

Run the control plane with a configuration file:

```bash
./faraway-edge run --static-path config.json
```

### Custom Port

Specify a custom port for the control plane:

```bash
./faraway-edge run --static-path config.json --xds-port 19000
```

### With Authentication

Enable token-based authentication for added security:

```bash
./faraway-edge run --static-path config.json --token "your-secret-token"
```

When using authentication, ensure your Envoy proxies include the token in their configuration.

## Configuration

### Static Configuration (JSON)

Configuration files define routing rules using a JSON format. Each configuration specifies:

- **HTTP and HTTPS backend services**: Where to route traffic for each protocol
- **Domain mappings**: Which domains should route to which services
- **Connection settings**: Timeouts and other connection parameters

Use the `example` command to see a sample configuration structure.

### Kubernetes Configuration

When deployed to Kubernetes with `k8sDiscovery.enabled: true`, the control plane automatically watches Ingress resources and generates routing configurations dynamically. This eliminates the need for static JSON configuration files.

**How it works:**

The control plane watches all Ingress resources in the cluster and automatically configures Envoy to route traffic to the LoadBalancer IPs specified in the Ingress status.

**Ingress Requirements:**

An Ingress resource will be included if:
- It has a LoadBalancer IP in `status.loadBalancer.ingress`
- It has at least one host defined in `spec.rules`
- It matches the configured `ingressClasses` filter (if specified)

**Supported Annotations:**

- `faraway-edge.paragor.net/timeout` - Connection timeout (e.g., `5s`, `10s`)
- `nginx.ingress.kubernetes.io/server-alias` - Additional domain aliases (comma-separated)

**Example Ingress:**

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: my-app
  annotations:
    faraway-edge.paragor.net/timeout: "10s"
    nginx.ingress.kubernetes.io/server-alias: "app.example.com, app2.example.com"
spec:
  ingressClassName: nginx
  rules:
    - host: myapp.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: my-app
                port:
                  number: 80
```

The control plane will route traffic for `myapp.example.com`, `app.example.com`, and `app2.example.com` to the LoadBalancer IPs on ports 80 (HTTP) and 443 (HTTPS).

## Monitoring

The control plane exposes diagnostic endpoints on port 8080:

- **`/healthz`**: Always returns healthy status
- **`/readyz`**: Returns ready when the control plane is operational
- **`/metrics`**: Metrics endpoint (placeholder for future implementation)

These endpoints can be used with container orchestration platforms, load balancers, or monitoring systems.

## Architecture

Faraway Edge operates as a control plane server that:

1. **Discovers routing rules** from either:
   - Static JSON configuration files, or
   - Kubernetes Ingress resources (watching for changes in real-time)
2. **Validates** the configuration for correctness
3. **Translates** high-level routing rules into detailed Envoy proxy configurations
4. **Serves** configurations to connected Envoy proxies via the xDS protocol
5. **Automatically pushes updates** when configurations change (file updates or Ingress changes)

Envoy proxies connect to the control plane and receive their routing configurations dynamically, eliminating the need for manual proxy configuration or restarts when routing rules change.

**Kubernetes Mode:** When running in Kubernetes, the control plane watches Ingress resources and uses the LoadBalancer IPs to route traffic, effectively creating a second layer of routing that can span multiple ingress controllers or provide additional features.

## Signal Handling

The application handles `SIGINT` (Ctrl+C) and `SIGTERM` signals gracefully:

- Stops accepting new connections
- Waits for active requests to complete
- Shuts down cleanly

This ensures zero-downtime deployments when integrated with orchestration platforms.

## Requirements

- Envoy proxy instances configured to connect to this control plane
- Network connectivity between Envoy proxies and the control plane server
- Valid JSON configuration file (for static mode) or Kubernetes cluster with Ingress resources (for Kubernetes mode)
- For Kubernetes deployment: RBAC permissions to watch Ingress resources (automatically configured by Helm chart)