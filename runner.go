package watch

// from https://github.com/codegangsta/gin

import (
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"runtime"
	"time"
)

type runner struct {
	dir       string
	bin       string
	args      []string
	writer    io.Writer
	command   *exec.Cmd
	starttime time.Time
}

func newRunner(bin string, args ...string) *runner {
	return &runner{
		bin:       bin,
		args:      args,
		writer:    ioutil.Discard,
		starttime: time.Now(),
	}
}

func (r *runner) Run(wait bool) error {
	if r.needsRefresh() {
		r.Kill()
	}

	if r.command == nil || r.Exited() {
		err := r.runBin(wait)
		if err != nil {
			log.Print("Error running: ", err)
		}
		time.Sleep(250 * time.Millisecond)
		return err
	}

	return nil
}

func (r *runner) Info() (os.FileInfo, error) {
	return os.Stat(r.bin)
}

func (r *runner) SetDir(dir string) {
	r.dir = dir
}

func (r *runner) SetWriter(writer io.Writer) {
	r.writer = writer
}

func (r *runner) Kill() error {
	if r.command != nil && r.command.Process != nil {
		done := make(chan error)
		go func() {
			r.command.Wait()
			close(done)
		}()

		if runtime.GOOS == "windows" {
			if err := r.command.Process.Kill(); err != nil {
				return err
			}
		} else if err := r.command.Process.Signal(os.Interrupt); err != nil {
			return err
		}

		select {
		case <-time.After(3 * time.Second):
			if err := r.command.Process.Kill(); err != nil {
				log.Println("failed to kill: ", err)
			}
		case <-done:
		}
		r.command = nil
	}

	return nil
}

func (r *runner) Exited() bool {
	return r.command != nil && r.command.ProcessState != nil && r.command.ProcessState.Exited()
}

func (r *runner) runBin(wait bool) error {
	r.command = exec.Command(r.bin, r.args...)
	r.command.Dir = r.dir

	stdout, err := r.command.StdoutPipe()
	if err != nil {
		return err
	}

	stderr, err := r.command.StderrPipe()
	if err != nil {
		return err
	}

	err = r.command.Start()
	if err != nil {
		return err
	}

	r.starttime = time.Now()

	go io.Copy(r.writer, stdout)
	go io.Copy(r.writer, stderr)

	waitFunc := func() error {
		defer stdout.Close()
		defer stderr.Close()
		return r.command.Wait()
	}

	if wait {
		return waitFunc()
	}

	go waitFunc()
	return nil
}

func (r *runner) needsRefresh() bool {
	info, err := r.Info()
	if err != nil {
		return false
	} else {
		return info.ModTime().After(r.starttime)
	}
}
