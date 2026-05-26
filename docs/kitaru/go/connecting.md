# Connecting to Kubernetes-Deployed Kitaru

This guide covers configuring the kitaru-go SDK to connect to Kitaru deployed in Kubernetes.

## Kubernetes Service Discovery

When Kitaru is deployed in Kubernetes, it exposes a Service. The SDK connects via:

- **Internal**: `http://<service-name>.<namespace>.svc.cluster.local:<port>`
- **Ingress**: `https://kitaru.yourdomain.com`
- **LoadBalancer**: `http://<external-ip>:<port>`

## Configuration Methods

### Method 1: Environment Variables

Set environment variables in your deployment:

```yaml
env:
  - name: KITARU_URL
    value: "http://kitaru-service.default.svc.cluster.local:8080"
  - name: KITARU_API_KEY
    valueFrom:
      secretKeyRef:
        name: kitaru-credentials
        key: api-key
  - name: KITARU_PROJECT
    value: "your-project"
```

Then create the client:

```go
client := kitaru.MustNewClientFromEnv()
```

### Method 2: Explicit Configuration

```go
client := kitaru.NewClient(
    "http://kitaru-service.default.svc.cluster.local:8080",
    "your-api-key",
    "your-project",
)
```

### Method 3: Kubernetes Service Mesh

If using Istio or Linkerd:

```go
client := kitaru.NewClient(
    "http://kitaru-service.namespace.svc.cluster.local:8080",
    "your-api-key",
    "your-project",
    kitaru.WithTimeout(30 * time.Second),
    kitaru.WithRetry(3),
)
```

## Service Discovery Examples

### Standard Deployment

```yaml
# kitaru-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kitaru
spec:
  replicas: 2
  selector:
    matchLabels:
      app: kitaru
  template:
    metadata:
      labels:
        app: kitaru
    spec:
      containers:
      - name: kitaru
        image: kitaru/server:latest
        ports:
        - containerPort: 8080
---
apiVersion: v1
kind: Service
metadata:
  name: kitaru-service
spec:
  selector:
    app: kitaru
  ports:
  - port: 8080
    targetPort: 8080
  type: ClusterIP
```

Connect with:

```go
client := kitaru.NewClient(
    "http://kitaru-service.default.svc.cluster.local:8080",
    "api-key",
    "project",
)
```

### With Ingress

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: kitaru-ingress
spec:
  rules:
  - host: kitaru.yourcompany.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: kitaru-service
            port:
              number: 8080
```

Connect with:

```go
client := kitaru.NewClient(
    "https://kitaru.yourcompany.com",
    "api-key",
    "project",
)
```

### With External LoadBalancer

```yaml
spec:
  type: LoadBalancer
  loadBalancerIP: 203.0.113.10
```

Connect with:

```go
client := kitaru.NewClient(
    "http://203.0.113.10:8080",
    "api-key",
    "project",
)
```

## Authentication

### API Key Authentication

```go
client := kitaru.NewClient(
    "http://kitaru-service:8080",
    "your-api-key",
    "your-project",
)
```

### OAuth2 (if configured)

```go
client := kitaru.NewClient(
    "http://kitaru-service:8080",
    "",
    "your-project",
    kitaru.WithOAuth2("client-id", "client-secret", "https://auth.example.com"),
)
```

## Connection Options

The SDK provides configuration options:

```go
client := kitaru.NewClient(
    baseURL,
    apiKey,
    project,
    kitaru.WithTimeout(30 * time.Second),      // Request timeout
    kitaru.WithRetry(3),                        // Retry attempts
    kitaru.WithDebug(true),                     // Debug logging
)
```

## Health Check

Verify connectivity:

```go
func checkConnection(client *kitaru.Client) error {
    ctx := context.Background()
    return client.Ping(ctx)
}

// Or check via HTTP
func checkHealth(client *kitaru.Client) error {
    resp, err := client.R().Get("/health")
    if err != nil {
        return fmt.Errorf("connection failed: %w", err)
    }
    if resp.StatusCode() != 200 {
        return fmt.Errorf("unhealthy: %d", resp.StatusCode())
    }
    return nil
}
```

## DNS Resolution

For local development, you can use `kubectl port-forward`:

```bash
kubectl port-forward -n default svc/kitaru-service 8080:8080
```

Then connect with:

```go
client := kitaru.NewClient(
    "http://localhost:8080",
    "api-key",
    "project",
)
```

## Network Policies

If using Kubernetes NetworkPolicies, ensure your pod can reach the Kitaru service:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-agent-to-kitaru
spec:
  podSelector:
    matchLabels:
      app: kitaru
  policyTypes:
  - Ingress
  ingress:
  - from:
    - podSelector:
        matchLabels:
          app: my-agent
    ports:
    - protocol: TCP
      port: 8080
```