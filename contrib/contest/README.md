# contest: consensus tester of ISAAC+

## Installation

```
$ go get github.com/spikeekips/mitum/contrib/contest
```


## Run

```
./contest run config.yml \
    --log /tmp/contest-log \
    --cpuprofile /tmp/contest-log/cpu.prof \
    --memprofile /tmp/contest-log/mem.prof \
    --trace /tmp/contest-log/trace.out \
    --exit-after 10s \
    --number-of-nodes 4 \
    2> /tmp/contest-log/stderr.log \
    | tee /tmp/contest-log/stdout.log
```
