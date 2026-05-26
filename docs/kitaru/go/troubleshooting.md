# Troubleshooting

Common issues when using kitaru-go with pedro-agentware.

## Connection Issues

### "connection refused" Error

**Symptom:**
```
dial tcp: connection refused
```

**Solutions:**
1. Verify Kitaru service is running:
   ```bash
   kubectl get pods -l app=kitaru
   ```

2. Check service DNS resolution:
   ```bash
   kubectl exec -it <your-pod> -- nslookup kitaru-service.default.svc.cluster.local
   ```

3. Verify port mapping:
   ```bash
   kubectl get svc kitaru-service
   ```

4. For local testing, use port-forward:
   ```bash
   kubectl port-forward -n default svc/kitaru-service 8080:8080
   ```

### "unauthorized" Error

**Symptom:**
```
401 Unauthorized
```

**Solutions:**
1. Verify API key is set correctly:
   ```go
   client := kitaru.NewClient(
       "http://kitaru-service:8080",
       "your-api-key",  // Check this is correct
       "your-project",
   )
   ```

2. Check Kubernetes secret exists:
   ```bash
   kubectl get secret kitaru-credentials
   ```

3. Verify secret contains correct key:
   ```bash
   kubectl get secret kitaru-credentials -o jsonpath='{.data.api-key}' | base64 -d
   ```

### "no such host" Error

**Symptom:**
```
dial tcp: no such host
```

**Solutions:**
1. Use full DNS name: `<service>.<namespace>.svc.cluster.local`
2. Check namespace matches your deployment
3. Verify CoreDNS is working:
   ```bash
   kubectl run dnsutils --image=tutum/dnsutils --restart=Never -- nslookup kubernetes.default
   ```

## Flow Execution Issues

### Flow Hangs or Times Out

**Symptom:**
```
context deadline exceeded
```

**Solutions:**
1. Increase timeout:
   ```go
   ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
   defer cancel()
   flow.RunWithWait(ctx, inputs, 5*time.Second)
   ```

2. Check Kitaru server logs:
   ```bash
   kubectl logs -l app=kitaru
   ```

3. Check for stuck flows in Kitaru UI

### "checkpoint failed" Warning

**Symptom:**
```
Warning: Failed to save checkpoint: checkpoint creation failed
```

**Solutions:**
1. Check Kitaru server is healthy
2. Verify flow execution ID exists
3. This is a warning, not a fatal error - flow continues

### Replay Fails

**Symptom:**
```
failed to replay: no such checkpoint
```

**Solutions:**
1. Ensure checkpoint name exists:
   ```go
   // List available checkpoints
   checkpoints, _ := flow.ListCheckpoints()
   ```
2. Check checkpoint was saved before the failure
3. Verify execution ID is correct

## Middleware Integration Issues

### Tool Not Found

**Symptom:**
```
unknown tool: some_tool
```

**Solutions:**
1. Verify tool is in ListTools():
   ```go
   tools := executor.ListTools()
   for _, t := range tools {
       fmt.Println(t.Name)
   }
   ```

2. Check toolMap in KitaruToolExecutor has correct mappings

### Policy Not Applied

**Symptom:**
Tool calls succeed but should be denied

**Solutions:**
1. Verify policy file loads correctly:
   ```go
   policy, err := middleware.LoadPolicyFromFile("policy.yaml")
   if err != nil {
       log.Fatalf("Policy load failed: %v", err)
   }
   ```

2. Check policy syntax (use `yamllint`)

3. Verify rule matching - tools use glob patterns:
   ```yaml
   tools:
     - "delete_*"  # Matches delete_database, delete_file, etc.
     - "web_search"  # Exact match
   ```

### Rate Limit Not Working

**Symptom:**
More calls allowed than rate limit specifies

**Solutions:**
1. Check max_rate config:
   ```yaml
   max_rate:
     count: 10    # Max calls
     window: 60   # In seconds
   ```

2. Rate limits are per policy instance - ensure middleware is singleton

### Audit Log Empty

**Symptom:**
No entries in audit log

**Solutions:**
1. Enable audit in policy:
   ```yaml
   rules:
     - name: "audit-all"
       tools:
         - "*"
       action: "allow"
       audit: true  # Must be true
   ```

2. Check auditor is configured:
   ```go
   auditor := audit.NewInMemoryAuditor()
   mw := middleware.New(executor, policy, middleware.WithAuditor(auditor))
   ```

## Kubernetes-Specific Issues

### Pod Cannot Reach Kitaru Service

**Symptom:**
```
dial tcp: i/o timeout
```

**Solutions:**
1. Check NetworkPolicy (if enabled):
   ```bash
   kubectl get networkpolicy
   ```

2. Add network policy to allow traffic:
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

3. Check security contexts and SELinux/AppArmor

### Secret Not Mounted

**Symptom:**
API key is empty

**Solutions:**
1. Verify secret exists:
   ```bash
   kubectl get secrets | grep kitaru
   ```

2. Check pod spec references correct secret:
   ```yaml
   env:
   - name: KITARU_API_KEY
     valueFrom:
       secretKeyRef:
         name: kitaru-credentials  # Must match secret name
         key: api-key              # Must match key name
   ```

3. Check secret data:
   ```bash
   kubectl describe secret kitaru-credentials
   ```

### Out of Memory

**Symptom:**
Pod is killed due to OOM

**Solutions:**
1. Increase memory limits:
   ```yaml
   resources:
     limits:
       memory: "512Mi"
     requests:
       memory: "256Mi"
   ```

2. Check for memory leaks in long-running flows
3. Use checkpoint frequently to release memory

## Debugging Tips

### Enable Debug Logging

```go
client := kitaru.NewClient(
    url,
    apiKey,
    project,
    kitaru.WithDebug(true),
)
```

### Check Middleware Decisions

```go
// Get last decision
auditor := mw.GetAuditor()
entries := auditor.GetLog()
if len(entries) > 0 {
    last := entries[len(entries)-1]
    fmt.Printf("Tool: %s, Decision: %s, Rule: %s\n",
        last.ToolCall.ToolName,
        last.Decision.Action,
        last.Decision.Rule,
    )
}
```

### Verify Policy Loading

```go
fmt.Printf("Policy loaded: %d rules\n", len(policy.Rules))
for _, rule := range policy.Rules {
    fmt.Printf("  - %s: %s\n", rule.Name, rule.Action)
}
```

## Getting Help

- Kitaru docs: https://kitaru.ai/docs
- pedro-agentware: https://github.com/Soypete/pedro-agentware
- Report issues at respective GitHub repos