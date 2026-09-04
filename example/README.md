# example/

A 30-second walkthrough of the product: a curated record, a lookup, a rewrite
that kills the anchor, `whence check` exiting 1.

This is not capture. The record is written with `whence add` so the per-line
hashes exist. Hand-dropping JSON does not.

## Run it

From the repo root (needs Go 1.22+ and git):

```console
$ bash example/demo.sh
```

That command should **exit 0**. Internally:

| step | command | expected |
|---|---|---|
| build | `go build` to a temp binary, never `$PATH` | — |
| seed | `whence add src/auth/session.go:<span>` in a throwaway git repo | writes `.whence/records.jsonl` |
| lookup | `whence src/auth/session.go` | intact, exact range, exit 0 |
| mutate | replace the three `CHECKOUT_*` `Set` calls with `persistNamespaced(s)` | working tree only |
| lookup | same | `ORPHANED — anchor lost` |
| gate | `whence check --base HEAD` | **exit 1**, one damaged decision |

If `demo.sh` itself exits 2, the gate did not fire and the demo is wrong.

`WHENCE=/path/to/whence bash example/demo.sh` skips the build (CI can pass the
binary it just compiled).

## What is in this directory

| path | role |
|---|---|
| `demo.sh` | the walkthrough |
| `src/auth/session.go` | the file it copies (`//go:build ignore` so `go test ./...` does not compile it) |
| `records.json` | the same decision in the **legacy** array shape — still read by the binary, never written. Not a runnable store. |

The throwaway repo lives under `$TMPDIR` and is deleted on exit.
