# mitum

Ready to go to next winter.

[![CircleCI](https://img.shields.io/circleci/project/github/spikeekips/mitum/proto3.svg?style=flat-square&logo=circleci&label=circleci&cacheSeconds=60)](https://circleci.com/gh/spikeekips/mitum/tree/proto3)
[![Documentation](https://readthedocs.org/projects/mitum-doc/badge/?version=proto3)](https://mitum-doc.readthedocs.io/en/latest/?badge=proto3)
[![GoDoc](https://godoc.org/github.com/golang/gddo?status.svg)](https://pkg.go.dev/github.com/spikeekips/mitum?tab=overview)
[![Go Report Card](https://goreportcard.com/badge/github.com/spikeekips/mitum)](https://goreportcard.com/report/github.com/spikeekips/mitum)
[![codecov](https://codecov.io/gh/spikeekips/mitum/branch/proto3/graph/badge.svg)](https://codecov.io/gh/spikeekips/mitum)
[![](https://tokei.rs/b1/github/spikeekips/mitum?category=lines)](https://github.com/spikeekips/mitum)

This is the third prototype for MITUM. The previous prototype, `proto2` can be found at [`proto2` branch](https://github.com/spikeekips/mitum/tree/proto1). The detailed operations of this prototype is writing at [MITUM Documentation](https://mitum-doc.readthedocs.io/en/proto3/). After

[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fspikeekips%2Fmitum.svg?type=large)](https://app.fossa.com/projects/git%2Bgithub.com%2Fspikeekips%2Fmitum?ref=badge_large)

## Test

```sh
$ go clean -testcache; go test -timeout 10s -tags test -race -v ./... -run .
```
