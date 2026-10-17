# num

`go.trippwill.dev/num` provides error-carrying decimals for financial and other
precision-sensitive Go programs. Arithmetic policies are fixed by the package;
applications write calculations without managing decimal contexts.

```go
pnl := exitPrice.Sub(entryPrice).Mul(quantity)
averageCost := totalCost.Div(quantity)
settlement := pnl.Quantize(2)
if !settlement.Ok() {
    return settlement.Err
}
```

v0.3.0 replaces `govalues/decimal` with private
[`apd`](https://pkg.go.dev/github.com/cockroachdb/apd/v3) arithmetic, with an
allocation-free inline path for small coefficients. It is a breaking release.
The complete contract is in the
[package documentation](https://pkg.go.dev/go.trippwill.dev/num) and the
[v0.3.0 migration notes](CHANGELOG.md).

The library follows a latest-minus-one Go release policy, currently requiring
Go 1.26.9 or later within the supported Go 1.26 and 1.27 series. Development and
the main CI checks use the Go 1.26.9 toolchain pinned in mise.

## Decimal policies

| Operation | Policy |
|---|---|
| Construction | Preserve finite decimal value and fractional trailing zeros |
| `Add`, `Sub` | Exact, retaining the greater operand scale |
| `Mul` | Exact, retaining the sum of operand scales |
| `Div` | Automatic significant-digit precision; half-even rounding |
| `Quantize(scale)` | Half-even rounding or padding to exactly that scale |
| Comparison | Numeric value, independent of stored scale |
| Serialization | Stored decimal, with no additional rounding |

Division precision is `max(34, meaningfulDigits(left), meaningfulDigits(right))`.
Meaningful digits count coefficient digits with trailing zeros removed; zero
counts as one digit. The 34-digit floor follows the decimal128 convention, and
longer operands can increase it. `1.23` and `1.2300` select the same precision.
Repeating results succeed, and unnecessary result zeros are removed. Division
rounds even a terminating result if it exceeds the selected precision.

Half-even (banker's rounding) is the only rounding mode. Exact arithmetic is
bounded to 4,096 coefficient digits, 4,096 integer digits, and fractional scale
0–4,096. Input is limited to 16,384 decoded bytes. Exceeding capacity returns an
error rather than silently rounding an exact result.

Values behave immutably, including when copied. `Num{}` is uninitialized and
fails `Ok()` even though its `Err` is nil; use `num.Zero()` for numeric zero.
Check `Ok()` before comparing or using a final calculation. `Num` is deliberately
not comparable with `==`.

## Presentation with fmt

Display rounding does not change the stored value:

```go
n := num.FromString("123.456789")
fmt.Printf("%.2f\n", n)         // 123.46
fmt.Printf("%10.2f\n", n)       //     123.46
fmt.Printf("%.*f\n", 4, n)      // 123.4568
fmt.Println(n)                 // 123.456789
fmt.Println(n.Quantize(2))      // 123.46, a new value with scale 2
```

`%f` and `%F` default to six fractional places; explicit precision selects the
display scale. Numeric output supports `+`, space, `-`, and `0` flags. `%v` and
`%s` preserve the complete representation; `%q` quotes it. Textual forms support
width and alignment but reject precision. Scientific and general verbs such as
`%e` and `%g` are unsupported. Unsupported or excessive requests produce bounded
formatting diagnostics.

Rendering limits are 4,096 fractional places, field width 16,384, and 16,384 output
bytes. Rendering uses decimal arithmetic directly, without `float64` conversion.

## Exact JSON, text, and SQL

JSON output uses bare decimal numbers. Input accepts bare numbers and strings:

```go
n := num.FromString("12345678901234567890.1200")
data, err := json.Marshal(n)
// data is 12345678901234567890.1200

var decoded num.Num
err = json.Unmarshal(data, &decoded)
// Check err for JSON syntax errors and decoded.Ok() for numeric errors.
```

Direct decoding into `Num` preserves precision and stored trailing zeros. A
consumer that decodes into JavaScript numbers or generic Go `float64` values can
lose precision. Use `Decoder.UseNumber()` when handling generic JSON in Go:

```go
decoder := json.NewDecoder(bytes.NewReader(data))
decoder.UseNumber()
var raw any
if err := decoder.Decode(&raw); err != nil {
    return err
}
exact := num.FromString(raw.(json.Number).String())
```

Numeric parse failures in `Num.UnmarshalJSON` are carried in `Num.Err` so other
struct fields can decode. Text/XML and SQL decoding return errors. SQL `Value()`
returns exact decimal text as a string. `NullNum` tracks optional presence: JSON
null, SQL NULL, and empty text mean absent; a malformed present JSON numeric value
remains present with a carried error.

## Development and release

```sh
mise run verify
mise run precommit
go test -run '^$' -bench . -benchmem ./...
```

The module path remains `go.trippwill.dev/num`. The v0.3.0 changes are intended
for a separately reviewed release; a version tag is not implied by these files.
After publication, install that version with:

```sh
go get go.trippwill.dev/num@v0.3.0
```

The module is licensed under MPL-2.0. Extraction provenance is recorded in
[`NOTICE`](NOTICE).
