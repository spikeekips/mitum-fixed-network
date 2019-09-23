# mitum

Prepare for winter.

[![Documentation](https://readthedocs.org/projects/mitum-doc/badge/?version=proto2)](https://mitum-doc.readthedocs.io/en/latest/?badge=proto2)
[![GoDoc](https://godoc.org/github.com/golang/gddo?status.svg)](https://godoc.org/github.com/spikeekips/mitum) 
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fspikeekips%2Fmitum.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2Fspikeekips%2Fmitum?ref=badge_shield)
[![Go Report Card](https://goreportcard.com/badge/github.com/spikeekips/mitum)](https://goreportcard.com/report/github.com/spikeekips/mitum)
[![](https://tokei.rs/b1/github/spikeekips/mitum?category=lines)](https://github.com/spikeekips/mitum)

This is the second prototype for MITUM. The previous prototype, `proto` can be found at [`proto1` branch](https://github.com/spikeekips/mitum/tree/proto1). The detailed operations of this prototype is writing at [MITUM Documentation](https://mitum-doc.readthedocs.io/en/proto2/).

## Test

```
$ go test -race -tags test ./... -v
```

```
$ golangci-lint run
$ for i in *; do [ -d "$i" ] || continue; nargs ./$i...; done
```
