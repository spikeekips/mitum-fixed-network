# contest: Testing consensus

To test the ISAAC+ and it's process, contest supports the proper tool for it.
contest provides,

- mitum node for testing: each node run in isolated environment.
- event logs analyzer: the logs from node are analyzed.

## Requirement

* docker host: each nodes run under docker.
* enough cpu power and memory


## Installation

```
$ go get -u github.com/spikeekips/contest
```

## Deployment

```
$ contest start design.yml
```

* `design.yml` is the configuration file of shape of testing.
