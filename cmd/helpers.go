package cmd

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/user"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

const (
	defaultPort = 22
)

func sshAgent() (ssh.AuthMethod, error) {
	sshAgent, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK"))
	if err != nil {
		return nil, err
	}
	return ssh.PublicKeysCallback(agent.NewClient(sshAgent).Signers), nil
}

func connectToHost(host *host) (*ssh.Client, error) {
	logger := log.WithFields(log.Fields{
		"hostname": host.hostName,
		"port":     host.port,
		"user":     host.user,
	})

	sa, err := sshAgent()
	if err != nil {
		return nil, err
	}

	cfg := &ssh.ClientConfig{
		User: host.user,
		Auth: []ssh.AuthMethod{
			sa,
		},
		Timeout:         time.Duration(timeout) * time.Second,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	logger.Debug("Dialing the host")
	sshCon, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", host.hostName, host.port), cfg)
	logger.Debug("Dialed to host successfully")
	if err != nil {
		return nil, errors.Wrapf(err, "could not ssh into server")
	}

	return sshCon, nil
}

type host struct {
	user     string
	port     int
	hostName string
}

func (h *host) String() string {
	return fmt.Sprintf("%v@%v:%v", h.user, h.hostName, h.port)
}

func split(hostList string) []string {
	return strings.FieldsFunc(hostList, func(r rune) bool {
		return unicode.IsSpace(r) || r == ','
	})
}

func parseHostsArg(hostsArg string) ([]*host, error) {
	hosts := make([]*host, 0)
	username := ""
	if u, err := user.Current(); err != nil {
		return hosts, err
	} else {
		username = u.Username
	}

	for _, hostArg := range split(hostsArg) {
		h := &host{
			user:     username,
			hostName: hostArg,
			port:     defaultPort,
		}

		if strings.Contains(h.hostName, "@") {
			parts := strings.Split(h.hostName, "@")
			h.user = parts[0]
			h.hostName = parts[1]
		}

		if strings.Contains(h.hostName, ":") {
			parts := strings.Split(h.hostName, ":")
			if p, err := strconv.ParseInt(parts[1], 10, 32); err == nil {
				h.hostName = parts[0]
				h.port = int(p)
			}
		}

		hosts = append(hosts, h)
	}
	return hosts, nil
}

func loadFileContents(file string) (string, error) {
	b, err := ioutil.ReadFile(file)
	if err != nil {
		return "", err
	}

	return string(b), nil
}
