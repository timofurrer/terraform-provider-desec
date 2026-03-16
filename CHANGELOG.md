## 0.3.0 (2026-03-16)

### IMPROVEMENTS (2 changes)

- resource/desec_domain: Reject bare unicode domains during plan
- functions: Add functions to convert IDN domains to ascii and to unicode (punycode)

## 0.2.2 (2026-03-12)

### IMPROVEMENTS (2 changes)

- api: Emit structured HTTP request/response trace logs via `TF_LOG_PROVIDER_DESEC=TRACE`
- docs: Improve `desec_zonefile` data source documentation

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
