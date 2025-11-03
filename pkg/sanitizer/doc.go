// Package sanitizer provides input normalization and sanitization functions for business data.
//
// All normalization functions are idempotent - applying them multiple times produces
// the same result. Functions handle invalid input gracefully, typically by returning
// empty strings or empty slices rather than errors.
//
// The package is designed to be used across microservices for consistent data
// normalization before validation and storage.
//
// Normalization includes:
//   - Phone numbers: Convert to E.164 format (+[country][number])
//   - URLs: Enforce HTTPS, lowercase domains, preserve paths and query parameters
//   - Strings: Collapse whitespace, trim leading/trailing spaces
//   - Labels: Lowercase for case-insensitive matching
//   - Slices: Remove duplicates and empty values after normalization
//   - Numbers: Clamp to valid ranges
package sanitizer
