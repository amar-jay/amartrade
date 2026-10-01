# amartrade HOWTO — for agents and humans

> Rule 0: **Chain related queries.** One `sh -c "... ; ... | grep ..."` tool call is easier to inspect than five separate tool calls.
> This reduces agent/tool round-trips, not Trade Map HTTP requests; it does not by itself prevent `429` responses.

Examples assume `./bin/amartrade`, `rg`, and `jq` exist. Prefer `rg` for text filtering and `jq` for JSON/JSONL processing. The examples also use standard Unix tools such as `sort`, `head`, `cut`, `wc`, and `awk`; `column` is only a display convenience.

`run_command` does NOT interpret shell. To chain, invoke the shell explicitly:

```sh
sh -c './bin/amartrade trademap search Ghana --format jsonl; echo ---; ./bin/amartrade trademap goods imports --by product --from GHA --years 2024 --hs-level 4 --format csv | grep -E "^REPORTER|GHA,WORLD,(2710,|8703,|8429,)"'
# command=sh args=[-c, <script>] cwd=.
```

## 1. Chain patterns

### 1.1 Discover then query (2-in-1)
```sh
sh -c './bin/amartrade trademap search Ghana --format jsonl; ./bin/amartrade trademap search 2710 --type product --format jsonl'
```

### 1.2 Separators for parsing
`;` always runs next. `&&` stops on error. For pipelines, use `set -o pipefail` so an upstream failure is not hidden. Add markers:
```sh
sh -c 'set -o pipefail; ./bin/amartrade trademap goods imports --by country --from GHA --product TOTAL --years 2000:2024 --format csv; echo ---2710---; ./bin/amartrade trademap goods imports --by country --from GHA --product 2710 --years 2000:2024 --format csv'
```

Quote the whole script with single quotes. Inside it, use double quotes where needed (for example, a `jq` expression). Never interpolate untrusted input into the script.

### 1.3 Filter at source with Linux tools
`csv` + `grep/rg` avoids dumping 800x HS4 rows:

```sh
# only header + fuels/machinery/vehicles + total
sh -c './bin/amartrade trademap goods imports --by product --from GHA --years 2010:2024 --hs-level 2 --format csv | grep -E "^REPORTER|GHA,WORLD,(27,|84,|87,|TOTAL,)"'

# preferred ripgrep form
sh -c './bin/amartrade trademap goods imports --by product --from GHA --years 2024 --hs-level 4 --format csv | rg "^(REPORTER|GHA,WORLD,(2710|8703|8429))"'

# top-N non-aggregate products, retaining the header (col 7 = single-year value)
sh -c 'set -o pipefail; ./bin/amartrade trademap goods imports --by product --from GHA --years 2024 --hs-level 4 --format csv | awk -F, '\''NR == 1 || $4 == "false"'\'' | { IFS= read -r header; printf "%s\n" "$header"; sort -t, -k7,7nr | head -10; }'

# jsonl + jq: values map, pretty labels
sh -c './bin/amartrade trademap goods imports --by partner --from GHA --product 2710 --years 2024 --format jsonl | jq -s "sort_by(.values[\"2024\"]) | reverse | .[0:8]"'
sh -c './bin/amartrade trademap search 2710 --type product --format jsonl | jq .label'

# cut/cols, count rows, check blanks (missing years)
sh -c './bin/amartrade trademap goods imports --by country --from GHA --product TOTAL --years 2000:2024 --format csv | cut -d, -f1,7-10; echo rows:; ./bin/amartrade trademap goods imports --by product --from GHA --years 2024 --format csv | wc -l'
```

Preferred set: `rg | jq | sort | head/tail | cut -d, | wc -l | awk | tee out.csv`; use `grep -E` when portability matters and `column -t -s,` for optional display formatting.

These filters treat the CLI's CSV as simple comma-separated fields. Use a CSV-aware tool if filtering columns that can contain quoted commas (notably search labels).

## 2. Output type by input (concise matrix)

### 2.1 `trademap search <query>`
Flags: `--type all|economy|economy-group|product|product-group` (default `all`), `--limit N` (default 20, `0`=all), `--format json|jsonl|csv` (default `jsonl`), `--children` (only matters on exact HS code).

| Input | Output row |
|---|---|
| `economy` | `{type:economy, selector:ISO3 e.g. GHA, native_code:288, label:Ghana}` |
| `economy-group` | `{type:economy-group, selector:NORMALIZED-LABEL e.g. EU27, native_code:42, label:EU 27}` |
| `product` exact e.g. `27` | `{type:product, selector:27, native_code:27, label:..., level:2/4/6, revisions:111111, hierarchy:[{code,label}...]}` + no descendants unless `--children` |
| `product` fuzzy e.g. `vehicles` | same shape, top `--limit` by exact>prefix>substring |
| `product-group` | `{type:product-group, selector:group:<id> e.g. group:18, native_code:18, label:Vehicles, level:6}` |
| `--format json` | pretty array `[...]` |
| `--format jsonl` | one object per line |
| `--format csv` | `TYPE,SELECTOR,NATIVE_CODE,LABEL,LEVEL,REVISIONS,HIERARCHY` (`HIERARCHY` = `code: label > ...`) |

`selector` is copy-pasteable into `--from/--to/--product`. `revisions` is opaque, keep it. `000`=WORLD.

An exact product search with `--children` can return current and historical HS codes. Do not sum every returned child blindly: use `revisions` and the queried year's HS classification to avoid double counting. No matches produce `[]` in JSON, no lines in JSONL, or only the header in CSV.

### 2.2 `trademap goods imports|exports`
Required: `--years YYYY` or `YYYY:YYYY`. Flags: `--by country|partner|product` (default `country`), `--from` reporter (default `WORLD`), `--to` partner (default `WORLD`), `--product` HS/`ALL`/`group:<id>`/label (default `ALL`), `--hs-level 2|4|6|10` (default 2, **only used if `--by product`**), `--data direct|mirror|mixed` (default `direct` → `D|M|X`), `--format json|jsonl|csv` (default `jsonl`), `--raw` (**only with `--format json`**).

Selector inputs: `--from/--to` accept `WORLD`, ISO3 codes, numeric economy codes, exact economy labels, normalized group labels, or `group:<id>`. `--product` accepts `ALL`/`TOTAL`, an HS code, exact product label, normalized product-group label, or `group:<id>`. Prefer the value returned by `search`.

| `--by` | One row means | Example command | You get |
|---|---|---|---|
| `country` | one reporter x fixed partner/product x years | `--by country --from GHA --product 2710 --years 2020:2024` | `GHA,WORLD,2710` time series |
| `partner` | one partner x fixed reporter/product x years | `--by partner --from GHA --product 2710 --years 2024` | `GHA,GBR,2710 / GHA,ARE,2710 ... + WORLD,2710 aggregate` ranked suppliers |
| `product` | one HS x fixed reporter/partner x years | `--by product --from GHA --years 2024 --hs-level 4` | `GHA,WORLD,2710 / 8703 / 8429 ... + TOTAL aggregate` ranked products |

Format mapping (same row model):

* `csv`: wide `REPORTER,PARTNER,PRODUCT,AGGREGATE,CURRENCY,VALUE_SCALE,YYYY,...`. `REPORTER/PARTNER` = ISO3 (`GHA`) or `WORLD` or normalized group label; `PRODUCT` = HS code or normalized label; `AGGREGATE=true` = group/TOTAL total; blank year cell = no reported value. Values are exact USD (`provider thousands ×1000`).
* `jsonl` (default): one `{"reporter":"GHA","partner":"WORLD","product":"2710","aggregate":false,"currency":"USD","value_scale":"units","values":{"2024":4058860000}}` per line. No metadata. Years with null values are omitted from `values`. Use with `jq`.
* `json`: pretty array of the above. `json --raw`: full envelope `{"query":{flow,by,reporter:{kind,code,label},partner,product,period_from,period_to,direct_mirror,indicator:VAL,currency:USD,value_scale:units,hs_level?},"records":[{reporter,partner,product,aggregate,data:[{period,value,unit,flag}]}],"sources":[...],"fetched_at":...}`. `unit/flag/sources` preserved, labels joined locally, duplicates removed, NES kept.

`--data direct` uses reporter-submitted data, `mirror` uses partner-reported data, and `mixed` asks the provider for mixed data. They are different observations, not interchangeable estimates. A blank/null can mean no reported value; it is not automatically zero. `mixed` is not a guaranteed fallback and some otherwise valid queries return HTTP 400, so report the gap if both direct and a separately checked mirror query fail.

Errors: nonzero exit with a message on stderr. Common cases are missing/invalid `--years`, invalid enum values, `--raw` without JSON, unknown selectors, HTTP errors (`400`, `429`, `5xx`), DNS/network failure, or malformed provider output. `--hs-level` is ignored unless `--by product`. An empty successful result is distinct from an error.

## 3. Recipe: Ghana fuels (copy-paste)

```sh
sh -c 'set -o pipefail; ./bin/amartrade trademap search Ghana --format jsonl; echo ---HS2---; ./bin/amartrade trademap goods imports --by product --from GHA --years 2024 --hs-level 2 --format csv | awk -F, '\''NR == 1 || $4 == "false"'\'' | { IFS= read -r h; printf "%s\n" "$h"; sort -t, -k7,7nr | head -5; }; echo ---HS4---; ./bin/amartrade trademap goods imports --by product --from GHA --years 2024 --hs-level 4 --format csv | awk -F, '\''NR == 1 || $4 == "false"'\'' | { IFS= read -r h; printf "%s\n" "$h"; sort -t, -k7,7nr | head -5; }; echo ---PARTNERS-2710---; ./bin/amartrade trademap goods imports --by partner --from GHA --product 2710 --years 2024 --format csv | awk -F, '\''NR == 1 || $4 == "false"'\'' | { IFS= read -r h; printf "%s\n" "$h"; sort -t, -k7,7nr | head -10; }'
```
