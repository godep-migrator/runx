package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"os/user"
	"path"
	"time"
)

const sshdAddr = "127.0.0.1:2222"

// Starts an sshd process listening on a unix domain socket,
// and connects to it. Returns the connection.
func startSSHD(key []byte) (*exec.Cmd, net.Conn, error) {
	u, err := user.Current()
	if err != nil {
		return nil, nil, err
	}
	tmp, err := ioutil.TempDir("", "sshd")
	if err != nil {
		return nil, nil, err
	}
	hostKeyPath := path.Join(tmp, "id_host_rsa")
	authKeyPath := path.Join(tmp, "authorized_keys")
	keygen := exec.Command("ssh-keygen", "-trsa", "-qN", "", "-f"+hostKeyPath)
	keygen.Stdout = os.Stdout
	keygen.Stderr = os.Stderr
	err = keygen.Run()
	if err != nil {
		log.Println("ssh-keygen:", err)
		return nil, nil, fmt.Errorf("ssh-keygen: %s", err)
	}
	err = ioutil.WriteFile(authKeyPath, key, 0600)
	if err != nil {
		return nil, nil, err
	}

	// Assume there exists a reasonable set of files in /etc/ssh,
	// including sshd_config and host key files.
	sshd := exec.Command("/usr/sbin/sshd", "-D", "-e", "-f/dev/null",
		"-oProtocol 2",
		"-oAllowUsers "+u.Username+" dyno",
		"-oListenAddress "+sshdAddr,
		"-oPasswordAuthentication no",
		"-oChallengeResponseAuthentication no",
		"-oUsePAM no",
		"-oPermitRootLogin no",
		"-oLoginGraceTime 20",
		"-oLogLevel ERROR",
		"-oPrintLastLog no",
		"-oUsePrivilegeSeparation no",
		"-oPermitUserEnvironment yes",
		"-oHostKey "+hostKeyPath,
		"-oAuthorizedKeysFile "+authKeyPath,
		"-oPidFile /dev/null",
	)
	sshd.Stdout = os.Stdout
	sshd.Stderr = os.Stderr
	if err = sshd.Start(); err != nil {
		return nil, nil, err
	}
	time.Sleep(50 * time.Millisecond)
	c, err := net.Dial("tcp", sshdAddr)
	if err != nil {
		sshd.Process.Kill()
		return nil, nil, err
	}
	return sshd, c, err
}
