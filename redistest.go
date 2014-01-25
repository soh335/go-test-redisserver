package redistest

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"time"
)

type RedisServer struct {
	Config  *RedisServerConfig
	Cmd     *exec.Cmd
	TempDir string
}

type RedisServerConfig struct {
	TimeOut    time.Duration
	AutoStart  bool
	Port       string
	Dir        string
	LogLevel   string
	UnixSocket string
}

func NewRedisServerConfig() *RedisServerConfig {
	return &RedisServerConfig{AutoStart: true, TimeOut: 3 * time.Second}
}

func (config *RedisServerConfig) Bytes() []byte {
	var buf bytes.Buffer
	v := reflect.ValueOf(config).Elem()
	for i := 0; i < v.NumField(); i++ {
		keyField := v.Type().Field(i)
		valueField := v.Field(i)
		switch keyField.Name {
		case "Port", "Dir", "LogLevel", "UnixSocket":
			// empty string
			if valueField.Len() > 0 {
				buf.WriteString(fmt.Sprintf("%s %s\n", bytes.ToLower([]byte(keyField.Name)), valueField.String()))
			}
		}
	}
	return buf.Bytes()
}

func NewRedisServer(config *RedisServerConfig) (*RedisServer, error) {
	redisServer := new(RedisServer)
	if config == nil {
		config = NewRedisServerConfig()
	}
	redisServer.Config = config

	dir, err := ioutil.TempDir("", "redistest")
	if err != nil {
		return nil, err
	}
	redisServer.TempDir = dir

	if config.Dir == "" {
		config.Dir = redisServer.TempDir
	}

	if config.LogLevel == "warning" {
		fmt.Println(`redistest does not support "loglevel warning", using "notice" instead.`)
		config.LogLevel = "notice"
	}

	if config.Port == "" && config.UnixSocket == "" {
		config.UnixSocket = filepath.Join(redisServer.TempDir, "redis.sock")
		config.Port = "0"
	}

	if config.AutoStart {
		if err := redisServer.Start(); err != nil {
			return nil, err
		}
	}

	return redisServer, nil
}

func (server *RedisServer) Start() error {
	conffile, err := os.OpenFile(
		filepath.Join(server.TempDir, "redis.conf"),
		os.O_RDWR|os.O_CREATE|os.O_EXCL,
		0755,
	)
	defer conffile.Close()

	if err != nil {
		return err
	}

	if _, err := conffile.Write(server.Config.Bytes()); err != nil {
		return err
	}

	logfile, err := os.OpenFile(
		filepath.Join(server.TempDir, "redis-server.log"),
		os.O_RDWR|os.O_CREATE|os.O_EXCL,
		0755,
	)
	defer logfile.Close()

	if err != nil {
		return err
	}

	path, err := exec.LookPath("redis-server")
	if err != nil {
		return err
	}

	cmd := exec.Command(path, conffile.Name())
	server.Cmd = cmd

	//append to log stdout, stderr
	appendLog := func(pipe io.Reader) {
		_, err := io.Copy(logfile, pipe)
		if err != nil {
			fmt.Println("err", err)
		}
	}
	if stdout, err := cmd.StdoutPipe(); err == nil {
		go appendLog(stdout)
	} else {
		return err
	}

	if stderr, err := cmd.StderrPipe(); err == nil {
		go appendLog(stderr)
	} else {
		return err
	}

	// start
	if err := cmd.Start(); err != nil {
		return err
	}

	// check server is launced ?
	timer := time.After(server.Config.TimeOut)
	r := regexp.MustCompile("The server is now ready to accept connections")
	ready := false
OuterLoop:
	for {
		select {
		case <-timer:
			break OuterLoop
		default:
			byt, err := ioutil.ReadFile(logfile.Name())
			if err != nil {
				return err
			}
			if r.Match(byt) {
				ready = true
				break OuterLoop
			}
			time.Sleep(time.Millisecond * 100)
		}
	}

	if !ready {
		if err := server.killAndStop(); err != nil {
			return err
		}
		byt, err := ioutil.ReadFile(logfile.Name())
		if err != nil {
			return err
		}
		return errors.New(
			fmt.Sprintf("%s\n%s", "*** failed to launch redis-server ***", string(byt)),
		)
	}

	return nil
}

func (server *RedisServer) Stop() error {
	// kill process
	if err := server.killAndStop(); err != nil {
		return err
	}
	return os.RemoveAll(server.TempDir)
}

func (server *RedisServer) killAndStop() error {
	if err := server.Cmd.Process.Kill(); err != nil {
		return err
	}
	if _, err := server.Cmd.Process.Wait(); err != nil {
		return err
	}
	return nil
}
