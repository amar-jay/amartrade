// Package trademap provides a client for ITC Trade Map's public data API.
//
// Trade Map's API is not formally documented and may change. Higher-level
// services in this package should build on Client rather than issuing HTTP
// requests directly so timeout, retry, and error behavior stays consistent.
package trademap
