package xmlrpcclient

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/ochinchina/gorilla-xmlrpc/xml"
	"github.com/ochinchina/supervisord/types"
	"net"
	"net/http"
	"net/url"
)

type XmlRPCClient struct {
	serverurl string
	user      string
	password  string
}

type VersionReply struct {
	Value string
}

type StartStopReply struct {
	Value bool
}

type ShutdownReply StartStopReply

type AllProcessInfoReply struct {
	Value []types.ProcessInfo
}

func NewXmlRPCClient(serverurl string) *XmlRPCClient {
	return &XmlRPCClient{serverurl: serverurl}
}

func (r *XmlRPCClient) SetUser(user string) {
	r.user = user
}

func (r *XmlRPCClient) SetPassword(password string) {
	r.password = password
}

func (r *XmlRPCClient) Url() string {
	return fmt.Sprintf("%s/RPC2", r.serverurl)
}

func (r *XmlRPCClient) post(method string, data interface{}) (*http.Response, error) {
	buf, _ := xml.EncodeClientRequest(method, data)
	url, err := url.Parse(r.serverurl)
	if err != nil {
		return nil, err
	}
	var resp *http.Response = nil
	if url.Scheme == "http" || url.Scheme == "https" {
		req, err := http.NewRequest("POST", r.Url(), bytes.NewBuffer(buf))
		if err != nil {
			fmt.Println("Fail to create request:", err)
			return nil, err
		}
		if len(r.user) > 0 && len(r.password) > 0 {
			req.SetBasicAuth(r.user, r.password)
		}
		req.Header.Set("Content-Type", "text/xml")
		resp, err = http.DefaultClient.Do(req)
		//resp, err = http.Post(r.Url(), "text/xml", bytes.NewBuffer(buf))
		if err != nil {
			fmt.Println("Fail to send request to supervisord:", err)
			return nil, err
		}
	} else if url.Scheme == "unix" {
		conn, err := net.Dial("unix", url.Path)
		if err != nil {
			fmt.Printf("Fail to connect unix socket path: %s\n", r.serverurl)
			return nil, err
		}
		req, err := http.NewRequest("POST", "/RPC2", bytes.NewBuffer(buf))
		if err != nil {
			fmt.Printf("Fail to create a http request")
			return nil, err
		}
		if len(r.user) > 0 && len(r.password) > 0 {
			req.SetBasicAuth(r.user, r.password)
		}
		req.Header.Set("Content-Type", "text/xml")
		err = req.Write(conn)
		if err != nil {
			fmt.Printf("Fail to write to unix socket %s\n", r.serverurl)
			return nil, err
		}
		resp, err = http.ReadResponse(bufio.NewReader(conn), req)
		if err != nil {
			fmt.Printf("Fail to read response %s\n", err)
			return nil, err
		}
	}

	if resp.StatusCode/100 != 2 {
		fmt.Println("Bad Response:", resp.Status)
		resp.Body.Close()
		return nil, fmt.Errorf("Response code is NOT 2xx")
	}
	return resp, nil
}

func (r *XmlRPCClient) GetVersion() (reply VersionReply, err error) {
	ins := struct{}{}
	resp, err := r.post("supervisor.getVersion", &ins)

	if err != nil {
		return
	}
	defer resp.Body.Close()

	err = xml.DecodeClientResponse(resp.Body, &reply)

	return
}

func (r *XmlRPCClient) GetAllProcessInfo() (reply AllProcessInfoReply, err error) {
	ins := struct{}{}
	resp, err := r.post("supervisor.getAllProcessInfo", &ins)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	err = xml.DecodeClientResponse(resp.Body, &reply)

	return
}

func (r *XmlRPCClient) ChangeProcessState(change string, processName string) (reply StartStopReply, err error) {
	if !(change == "start" || change == "stop") {
		err = fmt.Errorf("Incorrect required state")
		return
	}

	ins := struct{ Value string }{processName}
	resp, err := r.post(fmt.Sprintf("supervisor.%sProcess", change), &ins)

	if err != nil {
		return
	}
	defer resp.Body.Close()

	err = xml.DecodeClientResponse(resp.Body, &reply)

	return
}

func (r *XmlRPCClient) ChangeAllProcessState(change string) (reply AllProcessInfoReply, err error) {
	if !(change == "start" || change == "stop") {
		err = fmt.Errorf("Incorrect required state")
		return
	}
	ins := struct{ Wait bool }{true}
	resp, err := r.post(fmt.Sprintf("supervisor.%sAllProcesses", change), &ins)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	err = xml.DecodeClientResponse(resp.Body, &reply)
	return
}

func (r *XmlRPCClient) Shutdown() (reply ShutdownReply, err error) {
	ins := struct{}{}
	resp, err := r.post("supervisor.shutdown", &ins)

	if err != nil {
		return
	}
	defer resp.Body.Close()

	err = xml.DecodeClientResponse(resp.Body, &reply)

	return
}

func (r *XmlRPCClient) ReloadConfig() (reply types.ReloadConfigResult, err error) {
	ins := struct{}{}
	resp, err := r.post("supervisor.reloadConfig", &ins)

	if err != nil {
		return
	}

	defer resp.Body.Close()
	xmlProcMgr := NewXmlProcessorManager()
	reply.AddedGroup = make([]string, 0)
	reply.ChangedGroup = make([]string, 0)
	reply.RemovedGroup = make([]string, 0)
	i := -1
	has_value := false
	xmlProcMgr.AddNonLeafProcessor("methodResponse/params/param/value/array/data", func() {
		if has_value {
			has_value = false
		} else {
			i++
		}
	})
	xmlProcMgr.AddLeafProcessor("methodResponse/params/param/value/array/data/value", func(value string) {
		has_value = true
		i++
		switch i {
		case 0:
			reply.AddedGroup = append(reply.AddedGroup, value)
		case 1:
			reply.ChangedGroup = append(reply.ChangedGroup, value)
		case 2:
			reply.RemovedGroup = append(reply.RemovedGroup, value)
		}
	})
	xmlProcMgr.ProcessXml(resp.Body)
	return
}

func (r *XmlRPCClient) SignalProcess(signal string, name string) (reply types.BooleanReply, err error) {
	ins := types.ProcessSignal{Name: name, Signal: signal}
	resp, err := r.post("supervisor.signalProcess", &ins)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	err = xml.DecodeClientResponse(resp.Body, &reply)
	return
}

func (r *XmlRPCClient) SignalAll(signal string) (reply AllProcessInfoReply, err error) {
	ins := struct{ Signal string }{signal}
	resp, err := r.post("supervisor.signalProcess", &ins)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	err = xml.DecodeClientResponse(resp.Body, &reply)

	return
}
