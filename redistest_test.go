package redistest

import (
	"bytes"
	"net"
	"testing"

	"github.com/gomodule/redigo/redis"
)

func TestConfig(t *testing.T) {

	{
		var b bytes.Buffer
		config := Config{}
		config.write(&b)
		expect := ""
		if b.String() != expect {
			t.Errorf("config should be: %s but got %s", expect, b.String())
		}
	}

	{
		var b bytes.Buffer
		config := Config{"dir": "/path/to/example", "port": "0"}
		config.write(&b)
		expect := "dir /path/to/example\nport 0\n"
		if b.String() != expect {
			t.Errorf("config should be: %s but got %s", expect, b.String())
		}
	}
}

func TestConnectRedisViaUnixScoket(t *testing.T) {
	s, err := NewServer(true, nil)
	if err != nil {
		t.Fatal("NewServer is err:", err.Error())
	}
	defer func() {
		if err := s.Stop(); err != nil {
			t.Fatal("failed to stop", err)
		}
	}()

	t.Log("unixsocket:", s.Config["unixsocket"])
	conn, err := redis.Dial("unix", s.Config["unixsocket"])
	if err != nil {
		t.Fatal("failed to connect to redis via unixscoket:", err.Error())
	}
	_, err = conn.Do("PING")
	if err != nil {
		t.Fatal("failed to execute command:", err)
	}
}

func TestConnectRedisViaTCP(t *testing.T) {
	// empty port
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal("failed to listen", err)
	}

	if err := l.Close(); err != nil {
		t.Fatal("failed to close", err)
	}

	_, port, err := net.SplitHostPort(l.Addr().String())
	if err != nil {
		t.Fatal("err", err)
	}

	t.Log("empty port:", port)
	s, err := NewServer(true, Config{
		"port": port,
	})

	if err != nil {
		t.Fatal("NewServer is err:", err.Error())
	}
	defer func() {
		if err := s.Stop(); err != nil {
			t.Fatal("failed to stop", err)
		}
	}()

	conn, err := redis.Dial("tcp", net.JoinHostPort("127.0.0.1", port))
	if err != nil {
		t.Fatal("failed to connect to redis via tcp:", err.Error())
		return
	}
	_, err = conn.Do("PING")
	if err != nil {
		t.Error("failed to execute command:", err)
		return
	}
}

func TestAutoStart(t *testing.T) {
	s, err := NewServer(false, nil)
	if err != nil {
		t.Fatal("NewServer is err:", err.Error())
	}

	t.Log("unixsocket:", s.Config["unixsocket"])
	_, err = redis.Dial("unix", s.Config["unixsocket"])

	if err == nil {
		t.Fatal("should not connect to redis server. because redis server is not runing yet")
	}

	t.Log("start redis server immediately")
	if err := s.Start(); err != nil {
		t.Fatal("failed to start", err)
	}
	defer func() {
		if err := s.Stop(); err != nil {
			t.Fatal("failed to stop", err)
		}
	}()

	conn, err := redis.Dial("unix", s.Config["unixsocket"])
	if err != nil {
		t.Fatal("failed to connect to redis via unixscoket:", err.Error())
		return
	}
	_, err = conn.Do("PING")
	if err != nil {
		t.Error("failed to execute command:", err)
		return
	}
}
