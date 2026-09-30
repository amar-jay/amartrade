# amartrade

`amartrade` is a Go CLI for querying and analyzing data.
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