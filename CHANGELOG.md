## 0.2.1 (2026-03-12)

### IMPROVEMENTS (1 change)

- docs: Add provider-level example showing how to create a domain and retrieve its deSEC nameservers via `data "desec_record"` for use in registrar settings

### BUG FIXES (1 change)

- fake server: Seed apex NS records (`ns1.desec.io.`, `ns2.desec.org.`) on domain creation, matching real deSEC API behaviour

## 0.2.0 (2026-03-12)

### IMPROVEMENTS (2 changes)

- Various bug fixes
- Provider attribute to serialize per-domain requests
- Improved rate limiting in API client

## 0.1.0 (2026-03-10)

Initial Release
