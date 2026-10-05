# num

`github.com/trippwill/num` supplies exact, error-carrying decimal values for
financial and other precision-sensitive Go programs.

It wraps `github.com/govalues/decimal` with fixed process-scale values,
chainable arithmetic, JSON/XML/text encoding, and `database/sql` support.

```sh
go get github.com/trippwill/num@v0.1.0
```

## Status

The initial `v0.1.0` release preserves the extracted API. Its process-global
scale can be set at link time:

```sh
go build -ldflags "-X github.com/trippwill/num.scaleOverride=4"
```

The module is licensed under MPL-2.0. Its extraction provenance is recorded in
[`NOTICE`](NOTICE).
