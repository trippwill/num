# num

`go.trippwill.dev/num` supplies exact, error-carrying decimal values for
financial and other precision-sensitive Go programs.

It wraps `github.com/govalues/decimal` with fixed process-scale values,
chainable arithmetic, JSON/XML/text encoding, and `database/sql` support.

After a new vanity-path release is tagged:

```sh
go get go.trippwill.dev/num@latest
```

## Status

The initial `v0.1.0` release was published as `github.com/trippwill/num`.
The vanity path requires a new release tag before `@latest` is available.
The process-global scale can be set at link time:

```sh
go build -ldflags "-X go.trippwill.dev/num.scaleOverride=4"
```

The module is licensed under MPL-2.0. Its extraction provenance is recorded in
[`NOTICE`](NOTICE).
