# Code Duplication Analysis

**Date:** 2025-01-20
**Severity:** Medium (Technical Debt)
**Impact:** Maintenance burden, inconsistent behavior, bug propagation

---

## Executive Summary

Analysis of the TelHawk Stack codebase has identified several categories of code duplication across services. While the `common/` package has been successfully used for some shared functionality (httputil, logging), there are still utility functions and patterns being reimplemented in multiple services.

**Key Findings:**
- ✅ Good centralization: `httputil.WriteJSON`, `logging` package
- ⚠️ Duplicated utilities: `parseInt`, `getClientIP` (with inconsistent implementations!)
- ⚠️ Repeated patterns: JSON decoding, method checking, pagination parsing
- ⚠️ Inconsistent error handling: Mix of `httputil.WriteJSON` and `json.NewEncoder`

---

## Detailed Findings

### 1. ⚠️ CRITICAL: `getClientIP()` - Inconsistent Implementations

**Location:**
- `auth/internal/handlers/auth_handler.go:100-108`
- `ingest/internal/handlers/hec_handler.go:317-326`

**Problem:** The implementations are **different**, which could lead to security issues!

**Auth Version:**
```go
func getClientIP(r *http.Request) string {
    if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
        return xff  // ⚠️ Returns FULL header (may contain multiple IPs)
    }
    if xri := r.Header.Get("X-Real-IP"); xri != "" {
        return xri
    }
    return r.RemoteAddr
}
```

**Ingest Version:**
```go
func getClientIP(r *http.Request) string {
    if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
        parts := strings.Split(xff, ",")
        return strings.TrimSpace(parts[0])  // ✅ Correctly extracts first IP
    }
    if xri := r.Header.Get("X-Real-IP"); xri != "" {
        return xri
    }
    return r.RemoteAddr
}
```

**Security Impact:**
- Auth service logs may contain multiple IPs in XFF header: `"192.168.1.1, 10.0.0.1, 172.16.0.1"`
- Ingest service correctly extracts first (client) IP: `"192.168.1.1"`
- This affects audit trails, rate limiting, and security event correlation

**Recommendation:** Move to `common/httputil` with the **correct implementation** (ingest version).

---

### 2. ⚠️ `parseInt()` - Duplicated Utility

**Location:**
- `alerting/internal/handlers/handlers.go:435-443`
- `rules/internal/handlers/handlers.go:586-594`

**Implementations:** Identical (9 lines each)

```go
func parseInt(s string, defaultVal int) int {
    if s == "" {
        return defaultVal
    }
    if v, err := strconv.Atoi(s); err == nil {
        return v
    }
    return defaultVal
}
```

**Usage:** Parsing pagination parameters from query strings:
```go
page := parseInt(r.URL.Query().Get("page"), 1)
limit := parseInt(r.URL.Query().Get("limit"), 50)
```

**Recommendation:** Move to `common/httputil` as `httputil.ParseIntParam()`.

---

### 3. ⚠️ Pagination Parsing - Repeated Pattern

**Locations:**
- `alerting/internal/handlers/handlers.go:98-105` (ListCases)
- `alerting/internal/handlers/handlers.go:274-278` (ListAlerts)
- `rules/internal/handlers/handlers.go:384-393` (ListSchemas)
- `query/internal/handlers/saved_searches.go` (likely similar pattern)

**Pattern:**
```go
req := &models.ListCasesRequest{
    Page:     parseInt(r.URL.Query().Get("page"), 1),
    Limit:    parseInt(r.URL.Query().Get("limit"), 50),
    Status:   r.URL.Query().Get("status"),
    Severity: r.URL.Query().Get("severity"),
    // ... more filters
}
```

**Recommendation:** Create reusable pagination parser in `common/httputil`:
```go
// httputil.ParsePagination extracts page/limit from query params
func ParsePagination(r *http.Request, defaultLimit int) (page, limit int)
```

---

### 4. ⚠️ Method Checking - Repeated Pattern

**Problem:** Every handler manually checks HTTP method:
```go
if r.Method != http.MethodPost {
    http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
    return
}
```

**Locations:** 100+ occurrences across all handler files

**Why This Matters:**
- Verbose boilerplate
- Inconsistent error messages
- Some use `http.Error`, some use `httputil.WriteJSON`
- Already documented in CLEANLINESS_TODO.md as router library consideration

**Recommendation:**
- Short-term: Use stdlib route patterns (Go 1.22+): `mux.HandleFunc("POST /api/v1/users", h.CreateUser)`
- Long-term: Consider `chi` or `gorilla/mux` for cleaner routing

---

### 5. ⚠️ JSON:API Response Building - Duplicated Structure

**Locations:**
- `auth/internal/handlers/auth_handler.go:58-73` (CreateUser)
- `auth/internal/handlers/auth_handler.go:226-244` (ListUsers)
- `rules/internal/handlers/handlers.go` (multiple functions)
- `query/internal/handlers/saved_searches.go` (likely similar)

**Pattern:**
```go
response := map[string]interface{}{
    "data": map[string]interface{}{
        "type": "user",
        "id":   resp.ID,
        "attributes": map[string]interface{}{
            "username":   resp.Username,
            "email":      resp.Email,
            "roles":      resp.Roles,
            // ...
        },
    },
}
w.Header().Set("Content-Type", "application/vnd.api+json")
json.NewEncoder(w).Encode(response)
```

**Problems:**
- Manual construction of JSON:API envelope
- Inconsistent use of `httputil.WriteJSON` vs `json.NewEncoder`
- No type safety
- Verbose and error-prone

**Recommendation:** Create JSON:API helpers in `common/httputil`:
```go
// httputil.WriteJSONAPIResource writes a single resource
func WriteJSONAPIResource(w http.ResponseWriter, status int, resourceType, id string, attributes interface{})

// httputil.WriteJSONAPICollection writes a collection with pagination
func WriteJSONAPICollection(w http.ResponseWriter, status int, resourceType string, items []interface{}, pagination *Pagination)
```

---

### 6. ⚠️ Inconsistent Error Handling in Handlers

**Problem:** Mix of error response patterns:

**Pattern 1 (Recommended):**
```go
httputil.WriteJSON(w, http.StatusCreated, response)
```

**Pattern 2 (Inconsistent):**
```go
json.NewEncoder(w).Encode(response)  // No error handling!
```

**Pattern 3 (Manual):**
```go
w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(response)  // No status code set explicitly
```

**Locations:**
- Auth handlers: Mix of all 3 patterns
- Storage handlers: Lines 66-72, 112-118 (Pattern 3)
- Alerting handlers: Lines 86-88, 115-117, 259-264 (Pattern 2)
- Rules handlers: Lines 298-305, 372-374, 436-438 (Pattern 2)

**Recommendation:**
- Standardize on `httputil.WriteJSON` or `httputil.WriteJSONAPI` everywhere
- Remove direct `json.Encoder` usage
- Already tracked in CLEANLINESS_TODO.md issue #7

---

## Proposed Solutions

### Solution 1: Expand `common/httputil` Package

Create centralized HTTP utilities:

```go
// common/httputil/request.go

// ParseIntParam parses an integer query parameter with a default value
func ParseIntParam(s string, defaultVal int) int {
    if s == "" {
        return defaultVal
    }
    if v, err := strconv.Atoi(s); err == nil {
        return v
    }
    return defaultVal
}

// GetClientIP extracts the real client IP from request headers
// Handles X-Forwarded-For (proxy chain), X-Real-IP, and RemoteAddr
func GetClientIP(r *http.Request) string {
    if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
        // X-Forwarded-For can be: "client, proxy1, proxy2"
        // We want the first (client) IP
        parts := strings.Split(xff, ",")
        return strings.TrimSpace(parts[0])
    }
    if xri := r.Header.Get("X-Real-IP"); xri != "" {
        return xri
    }
    return r.RemoteAddr
}

// Pagination represents common pagination parameters
type Pagination struct {
    Page  int `json:"page"`
    Limit int `json:"limit"`
    Total int `json:"total,omitempty"`
}

// ParsePagination extracts pagination params from query string
func ParsePagination(r *http.Request, defaultLimit, maxLimit int) Pagination {
    page := ParseIntParam(r.URL.Query().Get("page"), 1)
    limit := ParseIntParam(r.URL.Query().Get("limit"), defaultLimit)

    // Enforce maximum limit
    if limit > maxLimit {
        limit = maxLimit
    }

    // Ensure page is at least 1
    if page < 1 {
        page = 1
    }

    return Pagination{Page: page, Limit: limit}
}
```

```go
// common/httputil/jsonapi.go

// WriteJSONAPIResource writes a single JSON:API resource response
func WriteJSONAPIResource(w http.ResponseWriter, status int, resourceType, id string, attributes interface{}) {
    response := map[string]interface{}{
        "data": map[string]interface{}{
            "type":       resourceType,
            "id":         id,
            "attributes": attributes,
        },
    }
    w.Header().Set("Content-Type", "application/vnd.api+json")
    WriteJSON(w, status, response)
}

// WriteJSONAPICollection writes a JSON:API collection response with pagination
func WriteJSONAPICollection(w http.ResponseWriter, status int, resourceType string, items []interface{}, pagination *Pagination) {
    data := make([]map[string]interface{}, len(items))
    for i, item := range items {
        // Assuming items have ID and Attributes fields
        data[i] = map[string]interface{}{
            "type":       resourceType,
            "id":         item["id"],
            "attributes": item,
        }
    }

    response := map[string]interface{}{
        "data": data,
    }

    if pagination != nil {
        response["meta"] = map[string]interface{}{
            "page":  pagination.Page,
            "limit": pagination.Limit,
            "total": pagination.Total,
        }
    }

    w.Header().Set("Content-Type", "application/vnd.api+json")
    WriteJSON(w, status, response)
}

// WriteJSONAPIError writes a JSON:API error response
func WriteJSONAPIError(w http.ResponseWriter, status int, code, title, detail string) {
    response := map[string]interface{}{
        "errors": []map[string]interface{}{
            {
                "status": strconv.Itoa(status),
                "code":   code,
                "title":  title,
                "detail": detail,
            },
        },
    }
    w.Header().Set("Content-Type", "application/vnd.api+json")
    WriteJSON(w, status, response)
}
```

---

### Solution 2: Standardize Handler Patterns

Create a handler template/guidelines document:

**File:** `docs/HANDLER_CONVENTIONS.md`

```markdown
# HTTP Handler Conventions

## Always Use httputil Helpers

✅ DO:
```go
httputil.WriteJSON(w, http.StatusOK, response)
httputil.WriteJSONAPI(w, http.StatusCreated, response)
httputil.WriteJSONAPIError(w, http.StatusBadRequest, "invalid_input", "Invalid Input", err.Error())
```

❌ DON'T:
```go
w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(response)  // No error handling!
```

## Use httputil Utilities

✅ DO:
```go
clientIP := httputil.GetClientIP(r)
pagination := httputil.ParsePagination(r, 50, 1000)
```

❌ DON'T:
```go
// Don't reimplement these in every handler!
func getClientIP(r *http.Request) string { ... }
```
```

---

## Migration Plan

### Phase 1: Add Utilities to `common/httputil` (Low Risk)
- [x] Review existing `common/httputil/response.go`
- [ ] Add `GetClientIP()` function
- [ ] Add `ParseIntParam()` function
- [ ] Add `Pagination` struct and `ParsePagination()` function
- [ ] Add JSON:API helpers (`WriteJSONAPIResource`, `WriteJSONAPICollection`, `WriteJSONAPIError`)
- [ ] Write unit tests for all new utilities
- [ ] Update `docs/CODE_STYLE_AND_CONVENTIONS.md`

**Estimated Effort:** 2-3 hours

### Phase 2: Migrate Auth Service (Medium Risk)
- [ ] Replace `getClientIP()` with `httputil.GetClientIP()`
- [ ] Standardize JSON responses to use `httputil.WriteJSON` everywhere
- [ ] Update imports
- [ ] Run tests to verify no regressions

**Estimated Effort:** 1-2 hours

### Phase 3: Migrate Ingest Service (Low Risk)
- [ ] Replace `getClientIP()` with `httputil.GetClientIP()`
- [ ] Remove duplicate function
- [ ] Run tests

**Estimated Effort:** 30 minutes

### Phase 4: Migrate Alerting Service (Medium Risk)
- [ ] Replace `parseInt()` with `httputil.ParseIntParam()`
- [ ] Refactor pagination parsing to use `httputil.ParsePagination()`
- [ ] Standardize JSON responses
- [ ] Run tests

**Estimated Effort:** 2 hours

### Phase 5: Migrate Rules Service (Medium Risk)
- [ ] Replace `parseInt()` with `httputil.ParseIntParam()`
- [ ] Refactor pagination parsing
- [ ] Migrate JSON:API responses to use helpers
- [ ] Standardize error responses
- [ ] Run tests

**Estimated Effort:** 2-3 hours

### Phase 6: Migrate Remaining Services (Low-Medium Risk)
- [ ] Storage service
- [ ] Query service
- [ ] Core service
- [ ] Web backend

**Estimated Effort:** 3-4 hours

### Phase 7: Documentation & Linting (Low Risk)
- [ ] Create `docs/HANDLER_CONVENTIONS.md`
- [ ] Update `docs/CODE_STYLE_AND_CONVENTIONS.md`
- [ ] Add to `docs/CLEANLINESS_TODO.md` completion tracking
- [ ] Consider adding linter rules to prevent future duplication

**Estimated Effort:** 1 hour

---

## Metrics

**Current State:**
- Duplicated functions: 4 (`getClientIP` x2, `parseInt` x2)
- Inconsistent error handling: ~50+ handlers
- Manual JSON:API construction: ~20+ handlers
- Lines of duplicate code: ~100 lines

**Target State:**
- Duplicated functions: 0
- Standardized error handling: 100%
- Centralized JSON:API construction: 100%
- Lines of duplicate code: 0
- Code reuse: `common/httputil` used by all 10 services

**Benefits:**
- Reduced maintenance burden
- Consistent security behavior (client IP extraction)
- Easier to add features (e.g., request ID tracking)
- Better testability (test utilities once)
- Improved code review efficiency

---

## Priority

**Priority:** Medium-High

**Rationale:**
- **Security:** Inconsistent `getClientIP()` affects audit trails and rate limiting
- **Maintainability:** Duplication makes bug fixes harder (must update N places)
- **Code Quality:** Aligns with Go best practices (DRY principle)
- **Developer Experience:** Reduces cognitive load for new contributors

**Recommended Timeline:**
- Week 1: Phase 1 (Add utilities)
- Week 2: Phases 2-3 (Auth + Ingest migration)
- Week 3: Phases 4-5 (Alerting + Rules migration)
- Week 4: Phases 6-7 (Remaining services + docs)

---

## References

- [Effective Go - Functions](https://go.dev/doc/effective_go#functions)
- [Go Code Review Comments - Package Names](https://go.dev/wiki/CodeReviewComments#package-names)
- [JSON:API Specification](https://jsonapi.org/format/)
- `docs/CLEANLINESS_TODO.md` - Issue #7 (Inconsistent Error Handling)
- `docs/CLEANLINESS_TODO.md` - Issue #11 (Router Path Parsing)

---

## Notes

- This analysis was performed on 2025-01-20
- Review and update quarterly as codebase evolves
- Track completion in `docs/CLEANLINESS_TODO.md`
