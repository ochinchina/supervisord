package process

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
)

// FileExists returns true if the file exists at the given path.
func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	if errors.Is(err, os.ErrNotExist) {
		return false
	}
	return err == nil
}

type ScriptExecutor struct {
	script string
}

func NewScriptExecutor(script string) *ScriptExecutor {
	return &ScriptExecutor{script: script}
}

// Execute the script in local or remote machine
// @return error if the script execution failed
func (se *ScriptExecutor) Execute() error {
	if strings.HasPrefix(se.script, "http://") || strings.HasPrefix(se.script, "https://") {
		return se.executeHTTP()
	} else if strings.HasPrefix(se.script, "tcp://") {
		return se.executeTCP()
	} else {
		return se.executeLocal()
	}
}

// Execute the script in remote machine via HTTP request
// @return error if the script execution failed
func (se *ScriptExecutor) executeHTTP() error {

	fields := strings.Fields(se.script)
	var url string = fields[0]
	if len(fields) > 1 {
		options, _, headers := se.parseHttpParameters(fields[1:])
		key_file, cert_file, ca_file := options["key"], options["cert"], options["ca"]
		data := se.loadData(options)

		return se.executeHttpRequest(url, key_file, cert_file, ca_file, data, headers)
	}
	return nil
}

// loadData load the data from options["data"] or options["d"]
// If the data starts with "@", it will be treated as a file path and the content of the file will be loaded as data.
// @return the data as []byte, or nil if no data is provided or the file does not exist
func (se *ScriptExecutor) loadData(options map[string]string) []byte {
	data := options["data"]
	if data == "" {
		data = options["d"]
	}
	if data == "" {
		return make([]byte, 0)
	}
	if strings.HasPrefix(data, "@") {
		dataFile := strings.TrimPrefix(data, "@")
		if fileExists(dataFile) {
			content, err := os.ReadFile(dataFile)
			if err == nil {
				return content
			} else {
				return make([]byte, 0)
			}
		} else {
			return make([]byte, 0)
		}
	} else {
		return []byte(data)
	}
}

// Execute the script in remote machine via HTTP request
// @return error if the script execution failed
func (se *ScriptExecutor) executeHttpRequest(url, key_file, cert_file, ca_file string, data []byte, headers map[string]string) error {
	tlsConfig, _ := se.createTlsConfig(key_file, cert_file, ca_file)

	var client *http.Client = nil
	if tlsConfig != nil {
		transport := &http.Transport{
			TLSClientConfig: tlsConfig,
		}
		client = &http.Client{Transport: transport}
	} else {
		client = &http.Client{}
	}

	var req *http.Request = nil

	if len(data) > 0 {
		req, _ = http.NewRequest("POST", url, bytes.NewReader(data))
	} else {
		req, _ = http.NewRequest("GET", url, nil)
	}

	if req == nil {
		return errors.New("failed to create HTTP request")
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return nil
}

func (se *ScriptExecutor) createTlsConfig(key_file, cert_file, ca_file string) (*tls.Config, error) {
	var caCertPool *x509.CertPool = nil
	if fileExists(ca_file) {
		caCert, err := os.ReadFile(ca_file)
		if err == nil {
			caCertPool = x509.NewCertPool()
			caCertPool.AppendCertsFromPEM(caCert)
		}
	}
	var clientCert *tls.Certificate = nil
	if fileExists(cert_file) && fileExists(key_file) {
		cert, err := tls.LoadX509KeyPair(cert_file, key_file)
		if err == nil {
			clientCert = &cert
		}
	}

	var tlsConfig *tls.Config = nil
	if clientCert != nil && caCertPool != nil {
		tlsConfig = &tls.Config{
			Certificates: []tls.Certificate{*clientCert},
			RootCAs:      caCertPool,
		}
	} else if clientCert != nil {
		tlsConfig = &tls.Config{
			Certificates: []tls.Certificate{*clientCert},
		}
	} else if caCertPool != nil {
		tlsConfig = &tls.Config{
			RootCAs: caCertPool,
		}
	}

	if tlsConfig != nil {
		return tlsConfig, nil
	} else {
		return nil, errors.New("failed to create TLS config")
	}

}

// Execute the script in local machine
// @return error if the script execution failed
func (se *ScriptExecutor) executeLocal() error {
	cmd := strings.TrimPrefix(se.script, "script://")
	_, err := executeCommand(cmd)
	return err
}

// Execute the script in remote machine via TCP connection
// @return error if the script execution failed
func (se *ScriptExecutor) executeTCP() error {
	hostPort := strings.TrimPrefix(se.script, "tcp://")
	_, err := net.Dial("tcp", hostPort)
	return err
}

// parseHttpParameters parse the parameters for HTTP request
// @return options, paramsList, headers
func (se *ScriptExecutor) parseHttpParameters(params []string) (map[string]string, []string, map[string]string) {
	index := 0
	options := make(map[string]string)
	paramsList := make([]string, 0)
	headers := make(map[string]string)
	for index < len(params) {
		// parse options in the form of -key value or --key value
		if strings.HasPrefix(params[index], "-") && index+1 < len(params) {
			if params[index] == "-H" || params[index] == "--header" {
				header := params[index+1]
				kv := strings.SplitN(header, ":", 2)
				if len(kv) == 2 {
					headers[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
				}
			} else {
				name := ""
				if strings.HasPrefix(params[index], "--") {
					name = params[index][2:]
				} else {
					name = params[index][1:]
				}
				options[name] = params[index+1]
			}
			index += 2
		} else { // parse parameters in the form of value
			paramsList = append(paramsList, params[index])
			index++
		}
	}

	return options, paramsList, headers

}
