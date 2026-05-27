# StatefulSet Registry -- Setup Guide & Risk Notes

This registry derives the node ID from the pod ordinal in the hostname.
No external coordination is needed for basic use, but production deployments
require specific Kubernetes configuration to reduce (not eliminate) the risks
described below.

---

## Required StatefulSet Configuration

### 1. Pod management policy

```yaml
spec:
  podManagementPolicy: OrderedReady  # default -- do not change to Parallel
```

`Parallel` starts all pods at the same time. Two pods with the same ordinal
will race to generate IDs. Never use `Parallel` if ID uniqueness matters.

### 2. Rolling update strategy

```yaml
spec:
  updateStrategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1  # default
```

This ensures pods are replaced one at a time. It does not close the overlap
window (see warning below), but it prevents multiple simultaneous replacements.

### 3. Graceful shutdown -- preStop hook

Add a preStop sleep so the old pod has time to finish in-flight work before
the new pod starts generating IDs with the same ordinal:

```yaml
spec:
  template:
    spec:
      containers:
        - name: your-app
          lifecycle:
            preStop:
              exec:
                command: ["sh", "-c", "sleep 5"]
      terminationGracePeriodSeconds: 30  # must be > preStop sleep + shutdown time
```

The sleep value should be >= your application's max in-flight request duration.
5s is a reasonable default for most services.

### 4. Avoid forced deletes in runbooks

```
# DO NOT run this unless you accept a duplicate-ID window:
kubectl delete pod <name> --force --grace-period=0
```

Document this explicitly in your incident runbooks. Forced deletes are common
during node drain and incident response. Treat them as a known risk, not an
accident.

### 5. Ordinal reuse after scale-down

After scaling down, do not scale back up and assign the same ordinal to a
different workload or a fresh pod until the ID retention window of your system
has passed (e.g. if IDs are stored for 90 days, wait 90 days before treating
that ordinal as safe to reuse).

---

## Risks That Cannot Be Fully Resolved Without an External Registry

The following issues are inherent to this registry. The configuration above
reduces exposure but does not eliminate them.

### WARNING: Rolling update overlap

During a rolling update, K8s terminates the old pod and starts a new pod with
the same ordinal. Both pods are alive during `terminationGracePeriodSeconds`.
The preStop sleep shrinks this window but cannot close it -- if the old pod is
still processing requests at the end of the sleep, it will continue to generate
IDs after the new pod has started.

**There is no built-in fence. This is unresolvable without Redis/etcd.**

### WARNING: Forced delete

`kubectl delete pod --force --grace-period=0` bypasses the termination grace
period entirely. The old process may still be running on the node while K8s
has already created a new pod with the same ordinal. Both processes will
generate IDs with the same node ID simultaneously.

**There is no mitigation at the registry level. Avoid forced deletes.**

### WARNING: Clock drift on restart

`lastMs` (the last issued timestamp) is in-memory only and resets to 0 on
every process start. If the wall clock has drifted backward since the last
run -- due to NTP correction, VM live migration, or host clock adjustment --
the generator will reissue timestamps it already used during the previous run.

The generator's `MaxClockDrift` guard only covers backward drift detected
within a single process lifetime. It does not protect across restarts.

**There is no mitigation at the registry level without persisting lastMs
externally (e.g. Redis) and reading it back on startup.**

### WARNING: Split-brain

If a network partition occurs after startup, two pods with the same ordinal
can operate independently and generate colliding IDs. This is indistinguishable
from normal operation at the registry level.

**There is no mitigation without a consensus-based lease (etcd, Redis with
Redlock, or similar).**

---

## When to Upgrade to the Redis Registry

Use the Redis registry instead if any of the following apply:

- Your system treats duplicate IDs as a hard error (financial transactions,
  audit logs, deduplication keys)
- You perform forced deletes as part of normal operations or incident response
- You run in an environment with frequent NTP corrections or VM migrations
- You need cross-cluster or multi-datacenter uniqueness guarantees
