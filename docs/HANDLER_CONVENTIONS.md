# HTTP Handler Conventions

**Last Updated:** 2025-01-20

This document outlines standard patterns and conventions for writing HTTP handlers in the TelHawk Stack.

---

## Table of Contents

- [Overview](#overview)
- [Required Utilities](#required-utilities)
- [Response Patterns](#response-patterns)
- [Request Parsing](#request-parsing)
- [Error Handling](#error-handling)
- [Examples](#examples)
- [Anti-Patterns](#anti-patterns)

---

## Overview

All HTTP handlers must follow these conventions to ensure consistency, maintainability, and security across services.

**Core Principles:**
1. ✅ **Always use `httputil` helpers** - Never implement utilities yourself
2. ✅ **Consistent error responses** - Use standard JSON:API error format
3. ✅ **Proper request context** - Pass `r.Context()` to all service calls
4. ✅ **Security first** - Use `GetClientIP()` for audit trails

---

## Required Utilities

### Import Statement

```go
import (
    "github.com/telhawk-systems/telhawk-stack/common/httputil"
)
```

### Available Utilities

| Function | Purpose | Usage |
|----------|---------|-------|
| `httputil.WriteJSON()` | Write JSON response | Standard API responses |
| `httputil.WriteJSONAPI()` | Write JSON:API response | REST API resources |
| `httputil.WriteJSONAPIResource()` | Write single resource | User, token, case, etc. |
| `httputil.WriteJSONAPICollection()` | Write collection | List endpoints |
| `httputil.WriteJSONAPIError()` | Write JSON:API error | All errors |
| `httputil.GetClientIP()` | Extract client IP | Audit trails, rate limiting |
| `httputil.ParseIntParam()` | Parse int query param | Page, limit, IDs |
| `httputil.ParsePagination()` | Parse pagination | List endpoints |

---

## Response Patterns

### ✅ DO: Use httputil Helpers

**Standard JSON Response:**
```go
func (h *Handler) GetHealth(w http.ResponseWriter, r *http.Request) {
    httputil.WriteJSON(w, http.StatusOK, map[string]string{
        "status": "healthy",
    })
}
```

**JSON:API Single Resource:**
```go
func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
    user, err := h.service.GetUser(r.Context(), userID)
    if err != nil {
        httputil.WriteJSONAPINotFoundError(w, "user", userID)
        return
    }

    // Prepare attributes (whatever fields you want to expose)
    attributes := map[string]interface{}{
        "username":   user.Username,
        "email":      user.Email,
        "created_at": user.CreatedAt,
    }

    httputil.WriteJSONAPIResource(w, http.StatusOK, "user", user.ID, attributes)
}
```

**JSON:API Collection:**
```go
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
    pagination := httputil.ParsePagination(r, 50, 1000) // default=50, max=1000

    users, total, err := h.service.ListUsers(r.Context(), pagination.Limit, pagination.Offset())
    if err != nil {
        httputil.WriteJSONAPIInternalError(w, "Failed to list users")
        return
    }

    // Convert to JSON:API format
    items := make([]map[string]interface{}, len(users))
    for i, user := range users {
        items[i] = map[string]interface{}{
            "id": user.ID,
            "attributes": map[string]interface{}{
                "username":   user.Username,
                "email":      user.Email,
                "created_at": user.CreatedAt,
            },
        }
    }

    pagination.Total = total
    httputil.WriteJSONAPICollection(w, http.StatusOK, "user", items, &pagination)
}
```

### ❌ DON'T: Manual JSON Encoding

```go
// ❌ BAD: No error handling
w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(response)

// ❌ BAD: Inconsistent with the rest of the codebase
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(http.StatusOK)
if err := json.NewEncoder(w).Encode(response); err != nil {
    log.Printf("encode error: %v", err)
}

// ✅ GOOD: Use httputil
httputil.WriteJSON(w, http.StatusOK, response)
```

---

## Request Parsing

### Client IP Extraction

**✅ DO: Use httputil.GetClientIP()**
```go
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
    clientIP := httputil.GetClientIP(r)  // Handles X-Forwarded-For correctly
    userAgent := r.Header.Get("User-Agent")

    // Pass to service for audit logging
    token, err := h.service.Login(r.Context(), username, password, clientIP, userAgent)
    // ...
}
```

**❌ DON'T: Reimplement GetClientIP()**
```go
// ❌ BAD: Inconsistent implementations across services
func getClientIP(r *http.Request) string {
    if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
        return xff  // WRONG: May contain multiple IPs!
    }
    return r.RemoteAddr
}
```

### Integer Query Parameters

**✅ DO: Use httputil.ParseIntParam()**
```go
func (h *Handler) GetItem(w http.ResponseWriter, r *http.Request) {
    itemID := httputil.ParseIntParam(r.URL.Query().Get("id"), 0)
    if itemID == 0 {
        httputil.WriteJSONAPIValidationError(w, "Item ID is required")
        return
    }
    // ...
}
```

**❌ DON'T: Manual parsing**
```go
// ❌ BAD: Duplicated code
func parseInt(s string, defaultVal int) int {
    if s == "" {
        return defaultVal
    }
    // ...
}
```

### Pagination Parameters

**✅ DO: Use httputil.ParsePagination()**
```go
func (h *Handler) ListItems(w http.ResponseWriter, r *http.Request) {
    pagination := httputil.ParsePagination(r, 50, 1000)
    // pagination.Page - current page (minimum 1)
    // pagination.Limit - items per page (capped at 1000)
    // pagination.Offset() - database offset

    items, total, err := h.service.ListItems(r.Context(), pagination.Limit, pagination.Offset())
    // ...
}
```

**❌ DON'T: Manual pagination parsing**
```go
// ❌ BAD: Repeated code in every handler
page := parseInt(r.URL.Query().Get("page"), 1)
limit := parseInt(r.URL.Query().Get("limit"), 50)
if limit > 1000 {
    limit = 1000
}
```

---

## Error Handling

### Standard Error Responses

Use the convenience functions for common errors:

```go
// 400 Bad Request - Validation error
httputil.WriteJSONAPIValidationError(w, "Username is required")

// 401 Unauthorized
httputil.WriteJSONAPIUnauthorizedError(w, "Invalid or expired token")

// 403 Forbidden
httputil.WriteJSONAPIForbiddenError(w, "Admin role required")

// 404 Not Found
httputil.WriteJSONAPINotFoundError(w, "user", userID)

// 500 Internal Server Error
log.Printf("Database error: %v", err)  // Always log first!
httputil.WriteJSONAPIInternalError(w, "An internal error occurred")
```

### Custom Errors

For errors that don't fit the convenience functions:

```go
httputil.WriteJSONAPIError(w, http.StatusConflict, "duplicate_entry",
    "Duplicate Entry", "A user with this email already exists")
```

### Multiple Errors

```go
errors := []httputil.JSONAPIErrorObject{
    httputil.NewJSONAPIError(400, "validation_error", "Validation Failed", "Username is required"),
    httputil.NewJSONAPIError(400, "validation_error", "Validation Failed", "Email is invalid"),
}
httputil.WriteJSONAPIErrorResponse(w, http.StatusBadRequest, errors)
```

---

## Examples

### Complete CRUD Handler

```go
package handlers

import (
    "encoding/json"
    "net/http"
    "github.com/telhawk-systems/telhawk-stack/common/httputil"
)

type UserHandler struct {
    service UserService
}

// Create
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
    var req CreateUserRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        httputil.WriteJSONAPIValidationError(w, "Invalid request body")
        return
    }

    clientIP := httputil.GetClientIP(r)
    userAgent := r.Header.Get("User-Agent")

    user, err := h.service.CreateUser(r.Context(), &req, clientIP, userAgent)
    if err != nil {
        httputil.WriteJSONAPIInternalError(w, "Failed to create user")
        return
    }

    attributes := map[string]interface{}{
        "username":   user.Username,
        "email":      user.Email,
        "created_at": user.CreatedAt,
    }

    httputil.WriteJSONAPIResource(w, http.StatusCreated, "user", user.ID, attributes)
}

// Read
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
    userID := r.URL.Query().Get("id")
    if userID == "" {
        httputil.WriteJSONAPIValidationError(w, "User ID is required")
        return
    }

    user, err := h.service.GetUser(r.Context(), userID)
    if err != nil {
        httputil.WriteJSONAPINotFoundError(w, "user", userID)
        return
    }

    attributes := map[string]interface{}{
        "username":   user.Username,
        "email":      user.Email,
        "created_at": user.CreatedAt,
    }

    httputil.WriteJSONAPIResource(w, http.StatusOK, "user", user.ID, attributes)
}

// List
func (h *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
    pagination := httputil.ParsePagination(r, 50, 1000)

    users, total, err := h.service.ListUsers(r.Context(), pagination.Limit, pagination.Offset())
    if err != nil {
        httputil.WriteJSONAPIInternalError(w, "Failed to list users")
        return
    }

    items := make([]map[string]interface{}, len(users))
    for i, user := range users {
        items[i] = map[string]interface{}{
            "id": user.ID,
            "attributes": map[string]interface{}{
                "username":   user.Username,
                "email":      user.Email,
                "created_at": user.CreatedAt,
            },
        }
    }

    pagination.Total = total
    httputil.WriteJSONAPICollection(w, http.StatusOK, "user", items, &pagination)
}

// Update
func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
    userID := r.URL.Query().Get("id")
    if userID == "" {
        httputil.WriteJSONAPIValidationError(w, "User ID is required")
        return
    }

    var req UpdateUserRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        httputil.WriteJSONAPIValidationError(w, "Invalid request body")
        return
    }

    clientIP := httputil.GetClientIP(r)
    userAgent := r.Header.Get("User-Agent")

    user, err := h.service.UpdateUser(r.Context(), userID, &req, clientIP, userAgent)
    if err != nil {
        httputil.WriteJSONAPIInternalError(w, "Failed to update user")
        return
    }

    attributes := map[string]interface{}{
        "username":   user.Username,
        "email":      user.Email,
        "updated_at": user.UpdatedAt,
    }

    httputil.WriteJSONAPIResource(w, http.StatusOK, "user", user.ID, attributes)
}

// Delete
func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
    userID := r.URL.Query().Get("id")
    if userID == "" {
        httputil.WriteJSONAPIValidationError(w, "User ID is required")
        return
    }

    if err := h.service.DeleteUser(r.Context(), userID); err != nil {
        httputil.WriteJSONAPIInternalError(w, "Failed to delete user")
        return
    }

    w.WriteHeader(http.StatusNoContent)
}
```

---

## Anti-Patterns

### ❌ DON'T: Reimplementing Utilities

```go
// ❌ BAD: Don't do this
func getClientIP(r *http.Request) string { /* ... */ }
func parseInt(s string, def int) int { /* ... */ }
```

### ❌ DON'T: Ignoring Errors

```go
// ❌ BAD: Silent failure
json.NewEncoder(w).Encode(response)

// ✅ GOOD: Error handling
httputil.WriteJSON(w, http.StatusOK, response)
```

### ❌ DON'T: Inconsistent Content Types

```go
// ❌ BAD: Mixing content types
w.Header().Set("Content-Type", "application/json")  // Sometimes
// (no content type set other times)

// ✅ GOOD: httputil sets content type consistently
httputil.WriteJSON(w, http.StatusOK, response)
httputil.WriteJSONAPI(w, http.StatusOK, response)
```

### ❌ DON'T: Manual Status Code Setting

```go
// ❌ BAD: Easy to forget
w.WriteHeader(http.StatusOK)
json.NewEncoder(w).Encode(response)

// ✅ GOOD: Status and encoding in one call
httputil.WriteJSON(w, http.StatusOK, response)
```

---

## Testing

When writing tests for handlers, use `httptest` package:

```go
func TestCreateUser(t *testing.T) {
    req := httptest.NewRequest("POST", "/users", strings.NewReader(`{"username":"alice"}`))
    w := httptest.NewRecorder()

    handler.CreateUser(w, req)

    if w.Code != http.StatusCreated {
        t.Errorf("Status = %d, want %d", w.Code, http.StatusCreated)
    }

    var response map[string]interface{}
    json.NewDecoder(w.Body).Decode(&response)
    // Assert on response structure
}
```

---

## References

- [JSON:API Specification](https://jsonapi.org/format/)
- [Go net/http Documentation](https://pkg.go.dev/net/http)
- `common/httputil` package documentation

---

## Changelog

- **2025-01-20:** Initial version with httputil utilities
