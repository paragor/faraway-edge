# Faraway Edge

A dynamic configuration control plane for Envoy proxies, enabling flexible HTTP/HTTPS traffic routing based on domain names.

## Purpose

Faraway Edge acts as a central configuration server for Envoy proxy instances, allowing you to define high-level routing rules that are automatically translated into Envoy configurations. Instead of manually configuring each Envoy proxy, you define your routing logic once, and all connected proxies receive updates automatically.

## Key Features

- **Dynamic Configuration**: Proxies automatically receive configuration updates without restarts
- **Domain-Based Routing**: Route HTTP and HTTPS traffic based on domain names to different backend services
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

Configuration files define routing rules using a JSON format. Each configuration specifies:

- **HTTP and HTTPS backend services**: Where to route traffic for each protocol
- **Domain mappings**: Which domains should route to which services
- **Connection settings**: Timeouts and other connection parameters

Use the `example` command to see a sample configuration structure.

## Monitoring

The control plane exposes diagnostic endpoints on port 8080:

- **`/healthz`**: Always returns healthy status
- **`/readyz`**: Returns ready when the control plane is operational
- **`/metrics`**: Metrics endpoint (placeholder for future implementation)

These endpoints can be used with container orchestration platforms, load balancers, or monitoring systems.

## Architecture

Faraway Edge operates as a control plane server that:

1. Reads routing configuration from JSON files
2. Validates the configuration for correctness
3. Translates high-level routing rules into detailed proxy configurations
4. Serves configurations to connected Envoy proxies
5. Automatically pushes updates when configurations change

Envoy proxies connect to the control plane and receive their routing configurations dynamically, eliminating the need for manual proxy configuration or restarts when routing rules change.

## Signal Handling

The application handles `SIGINT` (Ctrl+C) and `SIGTERM` signals gracefully:

- Stops accepting new connections
- Waits for active requests to complete
- Shuts down cleanly

This ensures zero-downtime deployments when integrated with orchestration platforms.

## Requirements

- Envoy proxy instances configured to connect to this control plane
- Network connectivity between Envoy proxies and the control plane server
- Valid JSON configuration file defining routing rules