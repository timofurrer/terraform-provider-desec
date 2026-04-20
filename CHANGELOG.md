## 0.6.2 (2026-04-20)

### BUG FIXES (1 change)

- `desec_records` fix import regression

## 0.6.1 (2026-04-15)

### BUG FIXES (1 change)

- `desec_records` with `exclusive = false` no longer deletes unmanaged RRsets dropped from
  config or inherited via import

## 0.6.0 (2026-03-21)

### BREAKING CHANGES (1 change)

- Rename `*_record` to `*_rrset`

### IMPROVEMENTS (1 change)

- Add `openpgpkey_dane` function to support `OPENPGPKEY` record type

## 0.5.0 (2026-03-17)

### IMPROVEMENTS (1 change)

- Add functions to work with DNSSEC material

## 0.4.2 (2026-03-17)

### IMPROVEMENTS (1 change)

- Reorder guides in docs

## 0.4.1 (2026-03-17)

### IMPROVEMENTS (1 change)

- Add some guides to the docs

## 0.4.0 (2026-03-17)

### IMPROVEMENTS (1 change)

- Add new `desec_records` domain to manage a rrsets of a domain via zone file or rrsets. Optionally, it can be exclusive.
## 0.4.0 (2026-03-17)

### IMPROVEMENTS (1 change)

- Add new `desec_records` domain to manage a rrsets of a domain via zone file or rrsets. Optionally, it can be exclusive.

## 0.3.1 (2026-03-16)

### IMPROVEMENTS (1 change)

- docs: add examples for DNSSEC relevant outputs / attributes

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
