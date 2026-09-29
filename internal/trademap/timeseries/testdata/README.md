# Goods time-series fixtures

The yearly and quarterly JSON fixtures were recorded from the public Trade Map
API on 2026-09-30 with `page=1` and `pageSize=2`. Source objects retain public
English attribution fields; redundant translations were removed.

The grouped fixture is `yearly_by_country.json`, recorded with
`countryGrp=42` (EU 27). It preserves member records separately from the group
aggregate record.

The monthly API currently responds with HTTP 401 and the message `Access to
monthly data requires login.` for anonymous requests. The three monthly JSON
files are therefore minimal synthetic successful responses used to test
decoding and route construction. Replace them with sanitized recordings when
an explicitly authorized fixture-recording credential is available. No token,
cookie, or account data may be committed.
