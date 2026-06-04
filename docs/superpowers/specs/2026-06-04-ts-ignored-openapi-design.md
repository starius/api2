# Design: Fix `ts:"-"` Incorrectly Excluding Fields from OpenAPI Spec

**Date:** 2026-06-04
**Ticket:** DDR-29083

## Problem

Fields tagged `ts:"-"` are excluded from the generated `openapi.json` spec, even though they are present on the wire. This happens because `ParseStructTag` in `typegen/util.go` maps both `json:"-"` and `ts:"-"` to the same `Ignored` state, and `parse.go` drops `Ignored` fields before building the data model that both the TypeScript and Swagger printers consume.

The distinction that must be upheld:
- `json:"-"` — field is not on the wire; skip in all output formats.
- `ts:"-"` — field is on the wire but hidden from TypeScript clients; skip only in TypeScript output.

## Approach: New `TsIgnored` State (Option A)

Add a distinct `TsIgnored PropertyState` so the parser can thread `ts:"-"` fields through to printers, and each printer decides independently whether to render them.

## Changes

### `typegen/util.go`

Add `TsIgnored` to the `PropertyState` constants after `Ignored`:

```go
const (
    Auto PropertyState = iota
    Ignored    // json:"-": not on the wire, skip everywhere
    TsIgnored  // ts:"-":  on the wire, skip only in TS output
    Optional
    Null
    NotNull
    NoInfo
)
```

Change the branching in `ParseStructTag` from a combined `||` condition to separate branches:

```go
if jsonTagVal == "-" {
    result.State = Ignored
} else if tsTagVal == "-" {
    result.State = TsIgnored
}
```

### `typegen/parse.go`

The skip condition at line 144 must remain unchanged — `TsIgnored` is intentionally not listed, so those fields pass through into `record.Fields`:

```go
if parseResult.State == Ignored || (parseResult.State == NoInfo && !isEmbed) {
    continue
}
```

### `typegen/typescript.printer.go`

Filter `TsIgnored` fields at render time in `recordToString`, before executing the template:

```go
visible := make([]RecordField, 0, len(r.Fields))
for _, f := range r.Fields {
    if f.Tag.State != TsIgnored {
        visible = append(visible, f)
    }
}
// pass visible slice to the template instead of r.Fields
```

### `typegen/swagger.printer.go`

No changes required. Once `TsIgnored` fields are retained in `record.Fields` by the parser, the Swagger printer will include them automatically.

## Testing

Add a struct to `typegen/tests/types/types.go` with a `ts:"-"` field alongside a normal field. Assert:
- TypeScript output does **not** contain the `ts:"-"` field.
- Swagger/OpenAPI output **does** contain the `ts:"-"` field.

## Non-Goals

- No change to `json:"-"` behavior.
- No change to `ts:"CustomType"` (manual TS type override) behavior.
- No changes to the static or YAML client generators.
