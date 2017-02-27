// Package redistest privides temporary redis-server for testing.
//
// This is basic usage of redistest.
//
//	s, err := redistest.NewServer(true, nil)
//	if err != nil {
//		panic(err)
//	}
//	defer s.Stop()
//	conn, err := redis.Dial("unix", s.Config["unixsocket"])
//	if err != nil {
//		panic(err)
//	}
//	_, err = conn.Do("PING")
//	if err != nil {
//		panic(err)
//	}
package redistest

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Server is main struct of redistest.
type Server struct {
	Config  Config
	cmd     *exec.Cmd
	TempDir string
	TimeOut time.Duration
	reader  io.ReadCloser
	writer  io.WriteCloser
}

// Config is configuration of redis-server.
type Config map[string]string

func (config Config) write(wc io.Writer) error {
	for key, value := range config {
		if _, err := fmt.Fprintf(wc, "%s %s\n", key, value); err != nil {
			return err
		}
	}
	return nil
}

// Error is error while starting redis
type Error struct {
	Err error
	Log string
}

func (err *Error) Error() string {
	return err.Err.Error() + "\n" + err.Log
}

// Cause returns the underlying cause of the error.
// The error can be inspected by errors.Cause https://github.com/pkg/errors
func (err *Error) Cause() error {
	return err.Err
}

// NewServer create a new Server.
// If config is nil, redistest use default value. It means use unixsocket, dir is random.
func NewServer(autostart bool, config Config) (*Server, error) {
	server := new(Server)

	if config == nil {
		config = Config{}
	}

	dir, err := ioutil.TempDir("", "redistest")
	if err != nil {
		return nil, err
	}
	server.TempDir = dir

	server.TimeOut = time.Second * 3

	if _, ok := config["dir"]; !ok {
		config["dir"] = server.TempDir
	}

	if config["loglevel"] == "warning" {
		fmt.Println(`redistest does not support "loglevel warning", using "notice" instead.`)
		config["loglevel"] = "notice"
	}

	_, hasPort := config["port"]
	_, hasUnixSocket := config["unixsocket"]

	if !hasPort && !hasUnixSocket {
		config["port"] = "0"
		config["unixsocket"] = filepath.Join(server.TempDir, "redis.sock")
	}

	server.Config = config

	if autostart {
		if err := server.Start(); err != nil {
			return nil, err
		}
	}

	return server, nil
}

// Start start redis-server.
func (server *Server) Start() error {

	conffile, err := server.createConfigFile()
	if err != nil {
		return err
	}

	path, err := exec.LookPath("redis-server")
	if err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	server.reader, server.writer = io.Pipe()
	log := io.MultiWriter(buf, server.writer)
	cmd := exec.Command(path, conffile.Name())
	cmd.Stderr = log
	cmd.Stdout = log
	server.cmd = cmd

	// start
	if err := cmd.Start(); err != nil {
		return err
	}

	// check server is launced ?
	if err := server.checkLaunch(server.reader); err != nil {
		return &Error{Err: err, Log: buf.String()}
	}
	return nil
}

// Stop stop redis-server
func (server *Server) Stop() error {
	defer os.RemoveAll(server.TempDir)
	// kill process
	if err := server.killAndWait(); err != nil {
		return err
	}
	if server.writer != nil {
		if err := server.writer.Close(); err != nil {
			return err
		}
	}
	if server.reader != nil {
		if err := server.reader.Close(); err != nil {
			return err
		}
	}
	return nil
}

func (server *Server) killAndWait() error {
	if err := server.cmd.Process.Kill(); err != nil {
		return err
	}
	if _, err := server.cmd.Process.Wait(); err != nil {
		return err
	}
	return nil
}

func (server *Server) createConfigFile() (*os.File, error) {
	conffile, err := os.OpenFile(
		filepath.Join(server.TempDir, "redis.conf"),
		os.O_RDWR|os.O_CREATE|os.O_EXCL,
		0755,
	)
	defer conffile.Close()

	if err != nil {
		return nil, err
	}

	if err := server.Config.write(conffile); err != nil {
		return nil, err
	}

	return conffile, nil
}

func (server *Server) checkLaunch(r io.Reader) error {
	done := make(chan struct{})
	go func() {
		// wait until the server is ready
		s := bufio.NewScanner(r)
		for s.Scan() {
			idx := strings.Index(s.Text(), "The server is now ready to accept connections")
			if idx >= 0 {
				close(done)
				break
			}
		}

		// ignore other logs
		for s.Scan() {
		}

		type closer interface {
			CloseWithError(err error) error
		}
		if r, ok := r.(closer); ok {
			r.CloseWithError(s.Err())
		}
	}()

	select {
	case <-done:
		// The server is now ready to accept connections
	case <-time.After(server.TimeOut):
		// timeout
		if err := server.Stop(); err != nil {
			return err
		}
		return errors.New("*** failed to launch redis-server ***")
	}

	return nil
}
