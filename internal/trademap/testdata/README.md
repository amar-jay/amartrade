# Trade Map test fixtures

Place small, sanitized API responses used by package tests in this directory.

Fixtures must not contain access tokens, cookies, email addresses, account IDs,
or other personal information. Prefer the smallest response that exercises the
behavior under test and record the endpoint shape and capture date in the test
that consumes it.
