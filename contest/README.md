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

This is example design file, `contest-example.yml`
```
nodes:
    - address: n0
    - address: n1
    - address: n2
```

* `design.yml` is the configuration file of shape of testing.

```
$ bash ./contest/build-contest.sh /tmp/contest
$ bash ./contest/build-runner.sh /tmp/runner

$ /tmp/contest \
    --log-level debug --log-color --verbose \
    start \
        --output /tmp/contest-shared \
        ./contest-example.yml \
        /tmp/runner
```
