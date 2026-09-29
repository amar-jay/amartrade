# amartrade

`amartrade` is a Go CLI for querying and analyzing international trade data.
It is designed for both humans and software agents: commands should be
scriptable, deterministic, and able to emit structured JSON.

The first data provider will be the JSON API used by the beta version of
[ITC Trade Map](https://www.trademap.org/). That API is currently undocumented,
so its integration is kept behind a dedicated internal package and must be
treated as an unstable dependency.

The reverse-engineered endpoint inventory is maintained in
[docs/trademap-api.md](docs/trademap-api.md).
The implementation roadmap and acceptance criteria are tracked in
[TODO.md](TODO.md).

## Status

The Cobra command framework and project boundaries are in place. Trade Map
requests and user-facing data commands have not been implemented yet.

## Requirements

- Go 1.27.1 or newer

## Development

Run the CLI directly:

```sh
go run . --help
go run . version
```

Run the tests:

```sh
go test ./...
```

Build a binary:

```sh
go build -o bin/amartrade .
```

## Project layout

```text
.
├── cmd/                         Cobra commands and CLI presentation
├── internal/
│   └── trademap/                Trade Map client and service composition
│       ├── reference/           Reference APIs, catalogs, and search
│       ├── timeseries/          Goods time-series queries and pagination
│       └── types/               Codes, selectors, enums, and periods
├── main.go                      Process entry point and build metadata
└── README.md
```

The dependency direction is intentionally one-way:

```text
main -> cmd -> internal/trademap
```

`cmd` owns flags, input validation, and output formatting. The `trademap`
packages own HTTP requests and provider-specific models. The root package
composes focused services such as `reference`; shared wire-safe values live in
`types`. None of these packages may depend on Cobra, write directly to standard
output, or terminate the process.

## Planned command surface

The initial API-backed commands will follow this shape:

```text
amartrade trademap goods time-series
amartrade trademap goods indicators
amartrade trademap services time-series
```

Commands intended for agents should support JSON output, write data only to
stdout, write diagnostics to stderr, and return a non-zero status for errors.
Human-readable table output can remain the default for interactive use.

## Trade Map API notes

The beta application currently uses this API root:

```text
https://www.trademap.org/api
```

Observed goods endpoints include:

```text
GET /goods/timeSeries/{yearly|quarterly|monthly}/{byProduct|byCountry|byPartner}
GET /goods/tradeIndicators/{byProduct|byCountry|byPartner}
```

Observed services endpoints include:

```text
GET /services/timeSeries/{yearly|quarterly}/{byService|byCountry|byPartner}
```

At the time of discovery, selected aggregate endpoints accepted anonymous
requests. That behavior is not a stability or access guarantee. The client
should therefore use conservative timeouts, bounded retries, rate limiting,
and explicit response validation. Authentication tokens, cookies, and personal
account data must never be committed or stored in test fixtures.

## Contribution conventions

- Keep provider code isolated under `internal/trademap`.
- Prefer typed request and response structures over untyped maps.
- Accept `context.Context` for every network operation.
- Inject an `http.Client` so tests never require the live service.
- Store only small, sanitized responses in `testdata`.
- Preserve raw provider codes in API models; resolve display labels separately.
- Add command tests for stdout, stderr, and invalid arguments.
