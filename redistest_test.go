package redistest

import (
	"github.com/garyburd/redigo/redis"
	"net"
	"testing"
)

func TestConfig(t *testing.T) {
	config := NewRedisServerConfig()
	if string(config.Bytes()) != "" {
		t.Error("should be empty config")
	}

	config.Dir = "/path/to/example"
	expect := "dir /path/to/example\n"
	if string(config.Bytes()) != expect {
		t.Error("config should be:", expect)
	}
}

func TestConnectRedisViaUnixScoket(t *testing.T) {
	s, err := NewRedisServer(nil)
	if err != nil {
		t.Error("NewRedisServer is err:", err.Error())
	}
	defer s.Stop()
	conn, err := redis.Dial("unix", s.Config.UnixSocket)
	if err != nil {
		t.Error("failed to connect to redis via unixscoket:", err.Error())
	}
	_, err = conn.Do("PING")
	if err != nil {
		t.Error("failed to execute command:", err)
	}
}

func TestConnectRedisViaTCP(t *testing.T) {
	config := NewRedisServerConfig()
	config.Port = "6789"
	s, err := NewRedisServer(config)
	if err != nil {
		t.Error("NewRedisServer is err:", err.Error())
	}
	defer s.Stop()
	conn, err := redis.Dial("tcp", net.JoinHostPort("0.0.0.0", s.Config.Port))
	if err != nil {
		t.Error("failed to connect to redis via tcp:", err.Error())
	}
	_, err = conn.Do("PING")
	if err != nil {
		t.Error("failed to execute command:", err)
	}
}
