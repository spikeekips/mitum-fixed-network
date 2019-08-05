# contest: consensus tester of ISAAC+


## Run

```
$ go run -race *.go run contest-config.yml --cpuprofile /tmp/contest-cpup.prof --exit-after 50s --number-of-nodes 3 2>&1 | tee /tmp/contest.log
```
