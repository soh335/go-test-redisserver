package redistest

import (
	"bytes"
	"github.com/garyburd/redigo/redis"
	"net"
	"testing"
)

func TestConfig(t *testing.T) {

	{
		var b bytes.Buffer
		config := Config{}
		config.Write(&b)
		expect := ""
		if b.String() != expect {
			t.Error("config should be:", expect)
		}
	}

	{
		var b bytes.Buffer
		config := Config{"dir": "/path/to/example", "port": "0"}
		config.Write(&b)
		expect := "dir /path/to/example\nport 0\n"
		if b.String() != expect {
			t.Error("config should be:", expect)
		}
	}
}

func TestConnectRedisViaUnixScoket(t *testing.T) {
	s, err := NewServer(true, nil)
	if err != nil {
		t.Error("NewServer is err:", err.Error())
	}
	defer s.Stop()

	t.Log("unixsocket:", s.Config["unixsocket"])
	conn, err := redis.Dial("unix", s.Config["unixsocket"])
	if err != nil {
		t.Error("failed to connect to redis via unixscoket:", err.Error())
	}
	_, err = conn.Do("PING")
	if err != nil {
		t.Error("failed to execute command:", err)
	}
}

func TestConnectRedisViaTCP(t *testing.T) {
	// empty port
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Error("failed to listen", err)
		return
	}

	if err := l.Close(); err != nil {
		t.Error("failed to close", err)
		return
	}

	_, port, err := net.SplitHostPort(l.Addr().String())
	if err != nil {
		t.Error("err", err)
		return
	}

	t.Log("empty port:", port)
	s, err := NewServer(true, Config{
		"port": port,
	})

	if err != nil {
		t.Error("NewServer is err:", err.Error())
	}
	defer s.Stop()

	conn, err := redis.Dial("tcp", net.JoinHostPort("127.0.0.1", port))
	if err != nil {
		t.Error("failed to connect to redis via tcp:", err.Error())
	}
	_, err = conn.Do("PING")
	if err != nil {
		t.Error("failed to execute command:", err)
	}
}
