package session

import (
  "github.com/fsouza/go-dockerclient"
  "io"
  "os"
)

type Session struct {
  client *docker.Client
  container *docker.Container
}

type Options struct {
  Image string
}

type Cmd struct {
  Path string
  Args []string
  Stdin io.Reader
  Stdout io.Writer
  Stderr io.Writer
}

func NewSession(sopts Options) (*Session, error) {

  host := os.Getenv("DOCKER_HOST")
  if host == "" {
    host = "unix:///var/run/docker.sock"
  }

  client, err := docker.NewClient(host)
  if err != nil {
    return nil, err
  }

  copts := docker.CreateContainerOptions{
    Config: &docker.Config{
      Image: sopts.Image,
      Cmd: []string{"/usr/bin/tail", "-f", "/dev/null"},
      AttachStdin: true,
    },
  }

  container, err := client.CreateContainer(copts)
  if err != nil {
    return nil, err
  }

  s := &Session{client, container}

  err = s.start()
  if err != nil {
    s.Close()
    return nil, err
  }

  return s, nil

}

func (s *Session) start() error {
  return s.client.StartContainer(s.container.ID, nil)
}

func (s *Session) Run(c Cmd) error {
  cmd := append([]string{c.Path}, c.Args...)
  exec, err := s.client.CreateExec(docker.CreateExecOptions{
    Container: s.container.ID,
    Cmd: cmd,
    AttachStdin: c.Stdin != nil,
    AttachStdout: c.Stdout != nil,
    AttachStderr: c.Stderr != nil,
  })
  if err != nil {
    return err
  }
  return s.client.StartExec(exec.ID, docker.StartExecOptions{
    InputStream: c.Stdin,
    OutputStream: c.Stdout,
    ErrorStream: c.Stderr,
  })
}

func (s *Session) Save() (string, error) {
  img, err := s.client.CommitContainer(docker.CommitContainerOptions{
    Container: s.container.ID,
  })
  if err != nil {
    return "", err
  }
  return img.ID, nil
}

func (s *Session) Close() error {
  opts := docker.RemoveContainerOptions{
    ID: s.container.ID,
    Force: true,
  }
  return s.client.RemoveContainer(opts)
}
