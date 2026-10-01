# amartrade HOWTO — for agents and humans

> Rule 0: **Chain, don't loop.** One `sh -c "... ; ... | grep ..."` call beats 5 separate calls.
> Fewer round-trips, less `429`, full context in one transcript.

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
`;` always runs next. `&&` stops on error. Add markers:
```sh
sh -c './bin/amartrade trademap goods imports --by country --from GHA --product TOTAL --years 2000:2024 --format csv; echo ---2710---; ./bin/amartrade trademap goods imports --by country --from GHA --product 2710 --years 2000:2024 --format csv'
```

### 1.3 Filter at source with Linux tools
`csv` + `grep/rg` avoids dumping 800x HS4 rows:

```sh
# only header + fuels/machinery/vehicles + total
sh -c './bin/amartrade trademap goods imports --by product --from GHA --years 2010:2024 --hs-level 2 --format csv | grep -E "^REPORTER|GHA,WORLD,(27,|84,|87,|TOTAL,)"'

# ripgrep equivalent (faster, same)
sh -c './bin/amartrade trademap goods imports --by product --from GHA --years 2024 --hs-level 4 --format csv | rg "^(REPORTER|GHA,WORLD,(2710|8703|8429))"'

# top-N with sort/cut/head (col 7 = single-year value)
sh -c './bin/amartrade trademap goods imports --by product --from GHA --years 2024 --hs-level 4 --format csv | sort -t, -k7 -nr | head -10'

# jsonl + jq: values map, pretty labels
sh -c './bin/amartrade trademap goods imports --by partner --from GHA --product 2710 --years 2024 --format jsonl | jq -s "sort_by(.values[\"2024\"]) | reverse | .[0:8]"'
sh -c './bin/amartrade trademap search 2710 --type product --format jsonl | jq .label'

# cut/cols, count rows, check blanks (missing years)
sh -c './bin/amartrade trademap goods imports --by country --from GHA --product TOTAL --years 2000:2024 --format csv | cut -d, -f1,7-10; echo rows:; ./bin/amartrade trademap goods imports --by product --from GHA --years 2024 --format csv | wc -l'
```

Useful set: `grep -E | rg | jq | sort | head/tail | cut -d, | wc -l | column -t -s, | tee out.csv`.

## 2. Output type by input (concise matrix)

### 2.1 `trademap search <query>`
Flags: `--type all|economy|economy-group|product|product-group` (default `all`), `--limit N` (default 20, `0`=all), `--format json|jsonl|csv` (default `jsonl`), `--children` (only matters on exact HS code).

| Input | Output row |
|---|---|
| `economy` | `{type:economy, selector:ISO3 e.g. GHA, native_code:288, label:Ghana}` |
| `economy-group` | `{type:economy-group, selector:NORMALIZED-LABEL e.g. EU27, native_code:42, label:EU 27}` |
| `product` exact e.g. `27` | `{type:product, selector:27, native_code:27, label:..., level:2/4/6, revisions:111111, hierarchy:[{code,label}...]}` + no children unless `--children` |
| `product` fuzzy e.g. `vehicles` | same shape, top `--limit` by exact>prefix>substring |
| `product-group` | `{type:product-group, selector:group:<id> e.g. group:18, native_code:18, label:Vehicles, level:6}` |
| `--format json` | pretty array `[...]` |
| `--format jsonl` | one object per line |
| `--format csv` | `TYPE,SELECTOR,NATIVE_CODE,LABEL,LEVEL,REVISIONS,HIERARCHY` (`HIERARCHY` = `code: label > ...`) |

`selector` is copy-pasteable into `--from/--to/--product`. `revisions` is opaque, keep it. `000`=WORLD.

### 2.2 `trademap goods imports|exports`
Required: `--years YYYY` or `YYYY:YYYY`. Flags: `--by country|partner|product` (default `country`), `--from` reporter (default `WORLD`), `--to` partner (default `WORLD`), `--product` HS/`ALL`/`group:<id>`/label (default `ALL`), `--hs-level 2|4|6|10` (default 2, **only used if `--by product`**), `--data direct|mirror|mixed` (default `direct` → `D|M|X`), `--format json|jsonl|csv` (default `jsonl`), `--raw` (**only with `--format json`**).

| `--by` | One row means | Example command | You get |
|---|---|---|---|
| `country` | one reporter x fixed partner/product x years | `--by country --from GHA --product 2710 --years 2020:2024` | `GHA,WORLD,2710` time series |
| `partner` | one partner x fixed reporter/product x years | `--by partner --from GHA --product 2710 --years 2024` | `GHA,GBR,2710 / GHA,ARE,2710 ... + WORLD,2710 aggregate` ranked suppliers |
| `product` | one HS x fixed reporter/partner x years | `--by product --from GHA --years 2024 --hs-level 4` | `GHA,WORLD,2710 / 8703 / 8429 ... + TOTAL aggregate` ranked products |

Format mapping (same row model):

* `csv`: wide `REPORTER,PARTNER,PRODUCT,AGGREGATE,CURRENCY,VALUE_SCALE,YYYY,...`. `REPORTER/PARTNER` = ISO3 (`GHA`) or `WORLD` or normalized group label; `PRODUCT` = HS code or normalized label; `AGGREGATE=true` = group/TOTAL total; blank year cell = no data (e.g. GHA `2020`, `2000,2002`). Values are exact USD (`provider thousands ×1000`).
* `jsonl` (default): one `{"reporter":"GHA","partner":"WORLD","product":"2710","aggregate":false,"currency":"USD","value_scale":"units","values":{"2024":4058860000}}` per line. No metadata. Use with `jq`.
* `json`: pretty array of the above. `json --raw`: full envelope `{"query":{flow,by,reporter:{kind,code,label},partner,product,period_from,period_to,direct_mirror,indicator:VAL,currency:USD,value_scale:units,hs_level?},"records":[{reporter,partner,product,aggregate,data:[{period,value,unit,flag}]}],"sources":[...],"fetched_at":...}`. `unit/flag/sources` preserved, labels joined locally, duplicates removed, NES kept.

Errors: `--years` missing/invalid → error; `--raw` without `json` → error; `--hs-level` ignored unless `--by product`; unknown `--from/--to/--product` → `unknown economy or group` (run `search` first); empty `2020` etc. = coverage gap, retry `--data mixed`.

## 3. Recipe: Ghana fuels (copy-paste)

```sh
sh -c './bin/amartrade trademap search Ghana --format jsonl; ./bin/amartrade trademap goods imports --by product --from GHA --years 2024 --hs-level 2 --format csv | sort -t, -k7 -nr | head -5; echo ---HS4---; ./bin/amartrade trademap goods imports --by product --from GHA --years 2024 --hs-level 4 --format csv | sort -t, -k7 -nr | head -5; echo ---PARTNERS-2710---; ./bin/amartrade trademap goods imports --by partner --from GHA --product 2710 --years 2024 --format csv | sort -t, -k7 -nr | head -10'
```
