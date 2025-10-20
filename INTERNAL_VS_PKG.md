# Go Project Layout: Why `internal/` Instead of `pkg/`?

## The Change

**Before**: All packages were in `pkg/` folder  
**After**: All packages moved to `internal/` folder

## Why This Matters

### The `internal/` Directory

The `internal/` directory is a **special directory recognized by the Go compiler**. It enforces a visibility rule:

> Code in an `internal/` directory can only be imported by code in the directory tree **rooted at the parent** of `internal/`.

**Example:**
```
event-pipeline/
├── internal/
│   ├── config/      # Can only be imported by event-pipeline
│   ├── models/      # Cannot be imported by external projects
│   └── database/    # Private to this application
└── cmd/
    └── consumer/    # Can import internal packages
```

### The `pkg/` Directory

The `pkg/` directory is a **convention** (not enforced by Go) for:
- Library code that's **safe to import** by external applications
- Public APIs meant to be consumed by other projects
- Reusable packages

**Example:**
```
mylibrary/
└── pkg/
    ├── validator/   # Can be imported by anyone
    └── utils/       # Public utility functions
```

## Why We Changed

### Our Application Context

The event-pipeline is a **standalone application**, not a library. The packages are:

1. **`config`** - Application-specific configuration
2. **`models`** - Event definitions specific to this system
3. **`database`** - Database layer tied to our schema
4. **`consumer`** - Kafka consumer implementation
5. **`producer`** - Kafka producer implementation
6. **`api`** - REST API handlers
7. **`dlq`** - Dead letter queue implementation
8. **`logger`** - Logging configuration
9. **`metrics`** - Prometheus metrics

**None of these should be imported by external projects!** They're tightly coupled to this specific application's business logic.

## Go Project Layout Standards

According to the [golang-standards/project-layout](https://github.com/golang-standards/project-layout):

### Use `internal/` when:
- ✅ Code is **private to your application**
- ✅ You don't want others to import it
- ✅ It's application-specific business logic
- ✅ It's tightly coupled to your system

### Use `pkg/` when:
- ✅ Building a **library or framework**
- ✅ Code is meant to be **imported by others**
- ✅ Providing a **public API**
- ✅ Reusable across projects

## Real-World Examples

### ✅ Correct: `internal/` for Applications

```
# Kubernetes
kubernetes/
├── cmd/
└── internal/              # Application code
    ├── kubelet/
    ├── scheduler/
    └── proxy/

# Docker
docker/
├── cmd/
└── internal/              # Application code
    ├── daemon/
    ├── container/
    └── network/
```

### ✅ Correct: `pkg/` for Libraries

```
# Prometheus Client Library
prometheus/client_golang/
├── prometheus/            # Public API (can import)
└── internal/             # Private implementation

# gRPC
grpc-go/
├── grpc.go               # Public API
├── credentials/          # Public packages
└── internal/             # Private implementation
```

## Benefits of Using `internal/`

1. **Compiler Enforcement**: Go compiler prevents external imports
2. **Clear Intent**: Signals "this is private application code"
3. **Refactoring Freedom**: Change internals without breaking external dependencies
4. **Best Practice**: Follows Go community standards
5. **Security**: Prevents accidental exposure of internal logic

## Migration Impact

### Changes Made
- ✅ Moved all packages from `pkg/` to `internal/`
- ✅ Updated all import paths
- ✅ Updated documentation
- ✅ Verified builds and tests pass

### No Functional Changes
- ✅ Application works exactly the same
- ✅ Docker images unchanged
- ✅ API endpoints unchanged
- ✅ All tests pass

### Import Path Changes
```go
// Before
import "event-pipeline/pkg/models"
import "event-pipeline/pkg/config"

// After
import "event-pipeline/internal/models"
import "event-pipeline/internal/config"
```

## When Would We Use `pkg/`?

If we were to **extract reusable components** that other projects could use:

```
event-pipeline/
├── cmd/                   # Applications
├── internal/              # Private app code
│   ├── api/
│   └── consumer/
└── pkg/                   # Public libraries (if needed)
    ├── kafkautil/        # Reusable Kafka utilities
    └── validator/        # Reusable validation package
```

But in our case, **everything is application-specific**, so it all belongs in `internal/`.

## Summary

| Directory | Purpose | Visibility | Our Usage |
|-----------|---------|------------|-----------|
| `cmd/` | Application entry points | Executables | ✅ Correct |
| `internal/` | Private application code | This project only | ✅ **Now correct** |
| `pkg/` | Public library code | External projects | ❌ Was incorrect |

**Bottom Line**: We moved from `pkg/` to `internal/` because our code is **private application logic**, not a public library. This follows Go best practices and prevents unintended external usage.

## References

- [Go Documentation: Internal Packages](https://go.dev/doc/go1.4#internalpackages)
- [golang-standards/project-layout](https://github.com/golang-standards/project-layout)
- [Effective Go: Package Names](https://go.dev/doc/effective_go#names)
