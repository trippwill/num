# Release notes

## v0.3.0 — prepared, unreleased

This release replaces the fixed six-place `govalues/decimal` wrapper with dynamic
decimals backed privately by `github.com/cockroachdb/apd/v3` v3.2.3. The module
path is unchanged. The minimum Go version is now 1.26.9, following the library
policy of the latest stable release series minus one. Development uses the
Go 1.26.9 toolchain pinned in mise; Go 1.27 is also supported.

### Arithmetic and presentation

- Parsing retains finite decimal value and fractional trailing zeros. Exact
  addition/subtraction retain the greater scale; multiplication retains the sum
  of operand scales. These operations no longer round each result to six places.
- Division rounds half-even at
  `max(34, meaningfulDigits(left), meaningfulDigits(right))` significant digits,
  then removes unnecessary trailing zeros. Trailing coefficient zeros do not
  increase the chosen precision. Repeating results succeed.
- `Quantize(scale)` explicitly rounds half-even or pads zeros to that scale.
- `fmt.Formatter` supports fixed-point `%f`/`%F`, with six fractional places by
  default and an explicit precision for display scaling. Width, dynamic
  width/precision, and numeric sign/alignment/zero-padding flags are supported.
  Formatting never changes the value. `%v`/`%s` preserve its representation; `%q`
  quotes it. Unsupported requests produce bounded diagnostics.
- Stored values are limited to 4,096 coefficient digits, 4,096 integer digits,
  and fractional scale 0–4,096. Numeric input is limited to 16,384 decoded bytes.
  Exact results exceeding capacity fail instead of silently rounding to fit.
- Rendering independently permits 4,096 fractional places, width 16,384, and
  16,384 output bytes.
- Values behave immutably when copied or used concurrently for reading.
  `Num{}` remains invalid; `Zero()` constructs valid scale-zero zero.

### JSON compatibility

JSON output changes from quoted decimal strings to bare exact numeric tokens:

```json
{"before":"123.450000","after":123.4500}
```

Input continues accepting quoted and bare numeric values. Direct decoding into
`Num` preserves precision; generic Go decoding uses `float64` unless configured
with `Decoder.UseNumber()`, and JavaScript numbers also use binary floating point.
Those consumers can lose digits. Output always uses the same numeric JSON type,
regardless of value size. Update API schemas and consumers expecting strings.

Text/XML and SQL continue encoding exact decimal text; SQL `Value()` remains
string-based. JSON numeric parse errors continue to be carried in `Num.Err`,
while text/SQL decoding returns errors. Failed decoding clears stale values.

### Migration

- Replace `From(govalues.Decimal)` with a backend-neutral constructor. To preserve
  an existing decimal's value and scale, use `FromString(d.String())` and check
  `Ok()`. No backend-specific constructor replaces `From`.
- Remove `Scale()` and the linker flag
  `-X go.trippwill.dev/num.scaleOverride=...`. Scale is now per value.
- Replace presentation assumptions such as `n.String() == "1.230000"` with the
  appropriate format, for example `fmt.Sprintf("%.6f", n)`.
- Use `n.Quantize(6)` when six places are part of the numeric or serialization
  contract. `fmt.Sprintf("%.6f", n)` changes presentation only.
- Remove checks for the obsolete `ErrIntegerOverflow`; int64 values fit the new
  backend. Handle documented input, initialization, scale, capacity, and
  division-by-zero errors instead.
- Keep initialization explicit: use `Zero()` rather than `Num{}` in arithmetic.
  Check `Ok()` after decoding and calculations, and before comparisons.
- Use `errors.Is` with `ErrUninitialized`, `ErrInvalidScale`,
  `ErrLimitExceeded`, and `ErrDivisionByZero` for the new error policies.
  `ErrInvalidNumericString`, `ErrUnsupportedType`, and `ErrMarshalErrorState`
  remain available; serialization failures also wrap the underlying error.
- Audit downstream fixed-scale behavior.

### Performance

Small coefficients use checked integer arithmetic and inline numeric storage.
Exact short quotients avoid building a padded 34-digit coefficient; larger or
rounded division results use apd with the same documented precision policy.
Big coefficients are reduced in blocks of decimal zeros. Constructors and JSON
codecs avoid allocating or validating the same short decimal text twice.

The original nine package benchmarks meet the v0.2.0 baseline: timing medians
are equal within ordinary measurement variation or lower, and allocation counts
match in every case. The arithmetic chain allocates zero times, compared with
six allocations in the initial all-apd implementation. Tests retain this
zero-allocation behavior for the common constructors and arithmetic chain.

Both versions use the identical retained-result `benchmark_test.go` harness.
The baseline is the actual `v0.2.0` tag at `c24f9b8`; the comparison is the
v0.3.0 arithmetic implementation at `51a903e`, before the Go version update.
The arithmetic assertion compares numeric values rather
than their different stored scales. The new arithmetic, capacity, and numeric
JSON policies remain in force; these are migration benchmarks with changed
contracts, rather than interchangeable implementations.

Recorded on 2026-10-17 with Go 1.25.7, Linux/amd64, Intel Core Ultra 5 228V.
Both test binaries were pinned to CPU 0 with one benchmark processor. Ten
200ms samples per version ran sequentially, alternating version order each
round. Times are sample medians; allocation counts were identical across all
ten samples for each case. Each cell lists **ns/op / B/op / allocs/op**.

| Benchmark | v0.2.0 | v0.3.0 |
|---|---:|---:|
| FromString | 22.32 / 0 / 0 | 15.96 / 0 / 0 |
| FromInt64 | 9.46 / 0 / 0 | 7.16 / 0 / 0 |
| FromFloat64 | 52.55 / 0 / 0 | 42.61 / 0 / 0 |
| ArithmeticChain | 73.60 / 0 / 0 | 73.04 / 0 / 0 |
| MarshalJSON | 150.70 / 96 / 3 | 150.05 / 96 / 3 |
| UnmarshalJSON bare | 31.26 / 0 / 0 | 21.58 / 0 / 0 |
| UnmarshalJSON quoted | 30.84 / 0 / 0 | 22.50 / 0 / 0 |
| Scan | 23.78 / 0 / 0 | 22.61 / 0 / 0 |
| Value | 36.64 / 32 / 2 | 34.94 / 32 / 2 |

These measurements cover the original package benchmark inputs. The separate
`BenchmarkDivision` cases exercise repeating quotients, terminating quotients
that require rounding, large operands, retained padding, and the maximum scale.
`BenchmarkAPDArithmeticChain` uses the same exact arithmetic and 34-digit
half-even division policy with the unmodified backend's reduction, without
Num's compact arithmetic paths. `BenchmarkQuantize` and `BenchmarkFormatFixed`
cover the new APIs.

To reproduce the recorded comparison, archive `v0.2.0` and `51a903e` into temporary
directories and copy `51a903e`'s `benchmark_test.go` over the baseline's original
benchmark file. That shared file builds against both APIs; the new API and backend
benchmarks live separately in `benchmark_apd_test.go`. Both snapshots retain
Go 1.25.7 compatibility. Build both test binaries with Go 1.25.7, then alternate
them using CPU affinity and the same flags. The equivalent primary suite command
for either version is:

```sh
taskset -c 0 go test -run '^$' -bench '^(BenchmarkFromString|BenchmarkFromInt64|BenchmarkFromFloat64|BenchmarkArithmeticChain|BenchmarkMarshalJSON|BenchmarkUnmarshalJSON_Bare|BenchmarkUnmarshalJSON_Quoted|BenchmarkScan|BenchmarkValue)$' -benchmem -benchtime=200ms -count=10 -cpu=1
```

### Publication

Publication uses an annotated `v0.3.0` tag on the validated release commit and a
check that `go.trippwill.dev/num@v0.3.0` resolves through the vanity path. These
release notes do not indicate that publication has occurred.
