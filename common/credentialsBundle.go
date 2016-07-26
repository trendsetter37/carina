package common

import (
	"crypto/tls"
	"crypto/x509"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/url"
	"path/filepath"
	"strings"
)

type CredentialsBundle struct {
	Files map[string][]byte
}

// NewCredentialsBundle loads a credentials bundle from the filesystem
func NewCredentialsBundle(credentialsPath string) (CredentialsBundle, error) {
	var creds CredentialsBundle

	files, err := ioutil.ReadDir(credentialsPath)
	if err != nil {
		return creds, errors.Wrap(err, "Invalid credentials bundle. Cannot list files in "+credentialsPath)
	}

	creds.Files = make(map[string][]byte)
	for _, file := range files {
		filePath := filepath.Join(credentialsPath, file.Name())
		fileContents, err := ioutil.ReadFile(filePath)
		if err != nil {
			return creds, errors.Wrap(err, "Invalid credentials bundle. Cannot read "+filePath)
		}
		creds.Files[file.Name()] = fileContents
	}

	return creds, nil
}

func (creds CredentialsBundle) GetCA() []byte {
	return creds.Files["ca.pem"]
}

func (creds CredentialsBundle) GetCAKey() []byte {
	return creds.Files["ca-key.pem"]
}

func (creds CredentialsBundle) GetCert() []byte {
	return creds.Files["cert.pem"]
}

func (creds CredentialsBundle) GetKey() []byte {
	return creds.Files["key.pem"]
}

func (creds CredentialsBundle) GetDockerEnv() []byte {
	return creds.Files["docker.env"]
}

func (creds CredentialsBundle) Verify() error {
	tlsConfig, err := creds.getTLSConfig()
	if err != nil {
		return err
	}

	// Lookup the Docker host from docker.env
	dockerEnv := string(creds.GetDockerEnv()[:])
	var dockerHost string
	sourceLines := strings.Split(dockerEnv, "\n")
	for _, line := range sourceLines {
		if strings.Index(line, "export ") == 0 {
			varDecl := strings.TrimRight(line[7:], "\n")
			eqLocation := strings.Index(varDecl, "=")

			varName := varDecl[:eqLocation]
			varValue := varDecl[eqLocation+1:]

			switch varName {
			case "DOCKER_HOST":
				dockerHost = varValue
			}

		}
	}

	dockerHostUrl, err := url.Parse(dockerHost)
	if err != nil {
		return errors.Wrap(err, "Invalid credentials bundle. Bad DOCKER_HOST URL.")
	}

	conn, err := tls.Dial("tcp", dockerHostUrl.Host, tlsConfig)
	if err != nil {
		return errors.Wrap(err, "Invalid credentials bundle. Unable to connect to the Docker host.")
	}
	conn.Close()

	return nil
}

func (creds CredentialsBundle) getTLSConfig() (*tls.Config, error) {
	var tlsConfig tls.Config
	tlsConfig.InsecureSkipVerify = true
	certPool := x509.NewCertPool()

	certPool.AppendCertsFromPEM(creds.GetCA())
	tlsConfig.RootCAs = certPool
	keypair, err := tls.X509KeyPair(creds.GetCert(), creds.GetKey())
	if err != nil {
		return &tlsConfig, errors.Wrap(err, "Invalid credentials bundle. Keypair mis-match.")
	}
	tlsConfig.Certificates = []tls.Certificate{keypair}

	return &tlsConfig, nil
}