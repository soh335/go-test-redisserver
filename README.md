# go-test-redisserver

[![Build Status](https://travis-ci.org/soh335/go-test-redisserver.png?branch=master)](https://travis-ci.org/soh335/go-test-redisserver)
[![GoDoc](https://godoc.org/github.com/soh335/go-test-redisserver?status.svg)](https://godoc.org/github.com/soh335/go-test-redisserver)

redis-server runner for tests. ```go-test-redisserver``` is a port of [Test::RedisServer](https://github.com/typester/Test-RedisServer).

## USAGE

```go
package main

import (
        "github.com/soh335/go-test-redisserver"
        "github.com/garyburd/redigo/redis"
)

func main() {
        s, err := redistest.NewServer(true, nil)
        if err != nil {
                panic(err)
        }
        defer s.Stop()
        conn, err := redis.Dial("unix", s.Config["unixsocket"])
        if err != nil {
                panic(err)
        }
        _, err = conn.Do("PING")
        if err != nil {
                panic(err)
        }
}
```

## LICENSE

* MIT

## SEE ALSO

* [Test::RedisServer](https://github.com/typester/Test-RedisServer)
* [go-test-mysqld](https://github.com/lestrrat/go-test-mysqld)
