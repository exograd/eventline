package ssh

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"strconv"
	"time"

	cgeneric "github.com/exograd/eventline/pkg/connectors/generic"
	"golang.org/x/crypto/ssh"
)

func (r *Runner) authMethod() (ssh.AuthMethod, error) {
	identity := r.runner.RunnerIdentity
	if identity == nil {
		return nil, fmt.Errorf("missing runner identity for authentication")
	}

	var method ssh.AuthMethod

	switch i := identity.Data.(type) {
	case *cgeneric.PasswordIdentity:
		method = ssh.Password(i.Password)

	case *cgeneric.SSHKeyIdentity:
		signer, err := ssh.ParsePrivateKey([]byte(i.PrivateKey))
		if err != nil {
			return nil, fmt.Errorf("cannot parse private key of identity "+
				"%q: %w", identity.Name, err)
		}

		method = ssh.PublicKeys(signer)

	default:
		return nil, fmt.Errorf("identity %q cannot be used for ssh "+
			"authentication", identity.Name)
	}

	return method, nil
}

func (r *Runner) connect(ctx context.Context) (*ssh.Client, error) {
	je := r.runner.JobExecution
	params := je.JobSpec.Runner.Parameters.(*RunnerParameters)

	// Prepare connection data
	address := net.JoinHostPort(params.Host, strconv.Itoa(params.Port))

	authMethod, err := r.authMethod()
	if err != nil {
		return nil, err
	}

	clientCfg := ssh.ClientConfig{
		User: params.User,
		Auth: []ssh.AuthMethod{authMethod},

		HostKeyCallback: ssh.InsecureIgnoreHostKey(),

		Timeout: 30 * time.Second,
	}

	if params.HostKey != nil {
		hostKey, err := ssh.ParsePublicKey(params.HostKey)
		if err != nil {
			return nil, fmt.Errorf("cannot parse host key: %w", err)
		}

		clientCfg.HostKeyCallback = ssh.FixedHostKey(hostKey)
		clientCfg.HostKeyAlgorithms = []string{params.HostKeyAlgorithm}
	}

	// Connect to the remote host
	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to %q: %w", address, err)
	}

	// Initialize the SSH connection itself
	sshConn, chans, reqs, err := ssh.NewClientConn(conn, address, &clientCfg)
	if err != nil {
		return nil, fmt.Errorf("cannot initialize ssh connection to %q: %w",
			address, err)
	}

	client := ssh.NewClient(sshConn, chans, reqs)

	return client, nil
}

func (r *Runner) uploadFileSet(ctx context.Context) error {
	// Directories
	dirPaths := make(map[string]struct{})
	for fp := range r.runner.FileSet.Files {
		dirPaths[path.Dir(path.Join(r.rootPath, fp))] = struct{}{}
	}

	for dirPath := range dirPaths {
		if err := r.createDirectory(ctx, dirPath, 0700); err != nil {
			return fmt.Errorf("cannot create directory %q: %w", dirPath, err)
		}
	}

	// Files
	openFlags := os.O_WRONLY | os.O_CREATE | os.O_TRUNC

	for fp, f := range r.runner.FileSet.Files {
		filePath := path.Join(r.rootPath, fp)

		// The sftp package does not support setting permissions when opening
		// the file. See https://github.com/pkg/sftp/issues/335 for more
		// information.
		file, err := r.sftpClient.OpenFile(filePath, openFlags)
		if err != nil {
			return fmt.Errorf("cannot open %q: %w", filePath, err)
		}

		if err := file.Chmod(f.Mode.Perm()); err != nil {
			return fmt.Errorf("cannot change permissions of %q: %w",
				filePath, err)
		}

		if _, err := io.Copy(file, bytes.NewReader(f.Content)); err != nil {
			return fmt.Errorf("cannot write %q: %w", filePath, err)
		}

		if err := file.Close(); err != nil {
			return fmt.Errorf("cannot close %q: %w", filePath, err)
		}
	}

	return nil
}

func (r *Runner) createDirectory(ctx context.Context, dirPath string, mode os.FileMode) error {
	// We do not try to chmod if the permissions are already correct. This is
	// annoying because it is an extra operation which is not useful most of
	// the times. But if a directory already exists, we may not be allowed to
	// chmod it; this is relevant for the root directory which may already
	// exist with permissions 0777 but be owned by another user.

	if err := r.sftpClient.MkdirAll(dirPath); err != nil {
		return err
	}

	info, err := r.sftpClient.Stat(dirPath)
	if err != nil {
		return fmt.Errorf("cannot stat directory: %w", err)
	}

	if info.Mode().Perm() == mode {
		return nil
	}

	if err := r.sftpClient.Chmod(dirPath, mode); err != nil {
		return fmt.Errorf("cannot chmod directory: %w", err)
	}

	return nil
}

func (r *Runner) deleteDirectoryContent(dirPath string) error {
	var deletePath func(string) error

	deletePath = func(currPath string) error {
		info, err := r.sftpClient.Lstat(currPath)
		if err != nil {
			return fmt.Errorf("cannot stat %q: %w", currPath, err)
		}

		if info.IsDir() {
			children, err := r.sftpClient.ReadDir(currPath)
			if err != nil {
				return fmt.Errorf("cannot list directory %q: %w",
					currPath, err)
			}

			for _, child := range children {
				err := deletePath(path.Join(currPath, child.Name()))
				if err != nil {
					return err
				}
			}

			if currPath != dirPath {
				if err := r.sftpClient.RemoveDirectory(currPath); err != nil {
					return fmt.Errorf("cannot delete directory %q: %w",
						currPath, err)
				}
			}
		} else {
			if err := r.sftpClient.Remove(currPath); err != nil {
				return fmt.Errorf("cannot delete %q: %w", currPath, err)
			}
		}

		return nil
	}

	return deletePath(dirPath)
}
