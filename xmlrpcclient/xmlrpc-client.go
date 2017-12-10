package xmlclient

import (
	"bytes"
	"fmt"
	"net/http"
    "github.com/ochinchina/supervisord/types"

	"github.com/ochinchina/gorilla-xmlrpc/xml"
)

type XmlRPCClient struct {
	serverurl string
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

func (r *XmlRPCClient) Url() string {
	return fmt.Sprintf("%s/RPC2", r.serverurl)
}

func (r *XmlRPCClient) GetVersion() (reply VersionReply, err error) {
	ins := struct{}{}
	buf, _ := xml.EncodeClientRequest("supervisor.getVersion", &ins)

	resp, err := http.Post(r.Url(), "text/xml", bytes.NewBuffer(buf))
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		fmt.Println("Bad Response:", resp.Status)
		err = fmt.Errorf("Response code is NOT 2xx")
		return
	}

	err = xml.DecodeClientResponse(resp.Body, &reply)

	return
}

func (r *XmlRPCClient) GetAllProcessInfo() (reply AllProcessInfoReply, err error) {
	ins := struct{}{}
	buf, _ := xml.EncodeClientRequest("supervisor.getAllProcessInfo", &ins)

	resp, err := http.Post(r.Url(), "text/xml", bytes.NewBuffer(buf))
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		fmt.Println("Bad Response:", resp.Status)
		err = fmt.Errorf("Response code is NOT 2xx")
		return
	}

	err = xml.DecodeClientResponse(resp.Body, &reply)

	return
}

func (r *XmlRPCClient) ChangeProcessState(change string, processName string) (reply StartStopReply, err error) {
	if !(change == "start" || change == "stop") {
		err = fmt.Errorf("Incorrect required state")
		return
	}

	ins := struct{ Value string }{processName}
	buf, _ := xml.EncodeClientRequest(fmt.Sprintf("supervisor.%sProcess", change), &ins)

	resp, err := http.Post(r.Url(), "text/xml", bytes.NewBuffer(buf))
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		fmt.Println("Bad Response:", resp.Status)
		err = fmt.Errorf("Response code is NOT 2xx")
		return
	}

	err = xml.DecodeClientResponse(resp.Body, &reply)

	return
}

func (r *XmlRPCClient) Shutdown() (reply ShutdownReply, err error) {
	ins := struct{}{}
	buf, _ := xml.EncodeClientRequest("supervisor.shutdown", &ins)

	resp, err := http.Post(r.Url(), "text/xml", bytes.NewBuffer(buf))
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		fmt.Println("Bad Response:", resp.Status)
		err = fmt.Errorf("Response code is NOT 2xx")
		return
	}

	err = xml.DecodeClientResponse(resp.Body, &reply)

	return
}

func (r *XmlRPCClient) ReloadConfig() (reply types.ReloadConfigResult, err error) {
	ins := struct{}{}
	buf, _ := xml.EncodeClientRequest("supervisor.reloadConfig", &ins)
	resp, err := http.Post(r.Url(), "text/xml", bytes.NewBuffer(buf))
	if err != nil {
		return
	}

	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		fmt.Println("Bad Response:", resp.Status)
		err = fmt.Errorf("Response code is NOT 2xx")
	} else {
        xmlProcMgr := NewXmlProcessorManager() 
        reply.AddedGroup = make( []string, 0 )
        reply.ChangedGroup = make( []string, 0 )
        reply.RemovedGroup = make( []string, 0 )
        i := -1
        has_value := false
        xmlProcMgr.AddNonLeafProcessor( "methodResponse/params/param/value/array/data", func () {
            if has_value {
                has_value = false
            } else {
                i ++
            }
        })
        xmlProcMgr.AddLeafProcessor( "methodResponse/params/param/value/array/data/value", func (value string) {
            has_value = true
            i ++
            switch i {
            case 0:
                reply.AddedGroup = append( reply.AddedGroup, value )
            case 1:
                reply.ChangedGroup = append( reply.ChangedGroup, value )
            case 2:
                reply.RemovedGroup = append( reply.RemovedGroup, value )
            }
        })
        xmlProcMgr.ProcessXml( resp.Body )
	}
	return
}
