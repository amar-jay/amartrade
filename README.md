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

The first working slice supports annual goods imports and exports:

```sh
amartrade trademap goods exports \
  --by country \
  --from DEU \
  --years 2020:2024 \
  --format csv
```

Economies accept ISO alpha-3 codes (`DEU`, `TUR`), Trade Map's native
three-digit codes, labels, `WORLD`, and group labels such as `EU27` or
`ASEAN`. Native group IDs use `group:<id>` when they could be ambiguous.

Use `--to` for the partner, `--product` for an HS code or product group,
and `--by country|partner|product` to choose the expanded dimension. Product
groups can be selected by label or `group:<id>`. JSON is the default output;
CSV produces a wide table with one column per year, and JSONL emits one compact
record per line. Normal JSON and JSONL use the same row model, with yearly
values under `values`. Pass `--raw` with JSON to include query metadata, labels,
sources, units, and provider flags; raw mode is not available with JSONL.

```text
REPORTER,PARTNER,PRODUCT,AGGREGATE,CURRENCY,VALUE_SCALE,2024
DEU,WORLD,TOTAL,false,USD,units,1677078371000
```

Trade Map reports monetary values in thousands of USD. `amartrade` converts
them to exact USD units. Meaningful group aggregates are retained and marked;
provider duplicates of an individual record are removed.

Use `search` to discover copy-ready economy, group, and product selectors:

```sh
amartrade trademap search EU27
amartrade trademap search "live animals" --type product --format jsonl
amartrade trademap search vehicles --type product-group --format csv
```

`--type` accepts `all`, `economy`, `economy-group`, `product`, or
`product-group`. Results include the selector accepted by query commands, the
native Trade Map code, label, HS level, and the provider's opaque HS revision
marker where relevant. The default limit is 20; pass `--limit 0` for all
matches. Exact HS-code searches include the full classification hierarchy and
return only that definition by default; add `--children` to include descendant
codes.
