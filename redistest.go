package redistest

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"time"
)

type Server struct {
	Config  Config
	Cmd     *exec.Cmd
	TempDir string
	TimeOut time.Duration
}

type Config map[string]string

func (config Config) Write(wc io.Writer) error {
	for key, value := range config {
		if _, err := fmt.Fprintf(wc, "%s %s\n", key, value); err != nil {
			return err
		}
	}
	return nil
}

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

func (server *Server) Start() error {

	conffile, err := server.createConfigFile()
	if err != nil {
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
	timer := time.After(server.TimeOut)
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
		if err := server.killAndWait(); err != nil {
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

func (server *Server) Stop() error {
	defer os.RemoveAll(server.TempDir)
	// kill process
	if err := server.killAndWait(); err != nil {
		return err
	}
	return nil
}

func (server *Server) killAndWait() error {
	if err := server.Cmd.Process.Kill(); err != nil {
		return err
	}
	if _, err := server.Cmd.Process.Wait(); err != nil {
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

	if err := server.Config.Write(conffile); err != nil {
		return nil, err
	}

	return conffile, nil
}
