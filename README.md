## Run Unit Tests
```
Brians-MacBook-Pro:healthcheck brian$ cd healthcheck/
Brians-MacBook-Pro:healthcheck brian$ go test
PASS
ok      main/healthcheck        2.321s
```

## Build
```
Brians-MacBook-Pro:healthcheck brian$ cd ..
Brians-MacBook-Pro:healthcheck brian$ go build
```

## Run (After Building)
```
Brians-MacBook-Pro:healthcheck brian$ ./main test.yml 
fetch.com has 67% availability percentage
www.fetchrewards.com has 100% availability percentage
fetch.com has 67% availability percentage
www.fetchrewards.com has 100% availability percentage
```
