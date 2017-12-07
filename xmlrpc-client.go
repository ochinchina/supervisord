package main

import (
	"bytes"
	goxml "encoding/xml"
	"fmt"
	"net/http"

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

type XmlPath struct {
	ElemNames []string
}

func NewXmlPath() *XmlPath {
	return &XmlPath{ElemNames: make([]string, 0)}
}

func (xp *XmlPath) AddChildren(names ...string) {
	for _, name := range names {
		xp.ElemNames = append(xp.ElemNames, name)
	}
}
func (xp *XmlPath) AddChild(elemName string) {
	xp.ElemNames = append(xp.ElemNames, elemName)
}

func (xp *XmlPath) RemoveLast() {
	if len(xp.ElemNames) > 0 {
		xp.ElemNames = xp.ElemNames[0 : len(xp.ElemNames)-1]
	}
}

func (xp *XmlPath) Equals(other *XmlPath) bool {
	if len(xp.ElemNames) != len(other.ElemNames) {
		return false
	}

	for i := len(xp.ElemNames) - 1; i >= 0; i -= 1 {
		if xp.ElemNames[i] != other.ElemNames[i] {
			return false
		}
	}
	return true
}

type ShutdownReply StartStopReply

type AllProcessInfoReply struct {
	Value []ProcessInfo
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

func (r *XmlRPCClient) ReloadConfig() (reply ReloadConfigResult, err error) {
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
		arrayStartPath := NewXmlPath()
		arrayStartPath.AddChildren("methodResponse", "params", "param", "value", "array", "data")
		ValueArrayPath := NewXmlPath()
		ValueArrayPath.AddChildren("methodResponse", "params", "param", "value", "array", "data", "value")
		curPath := NewXmlPath()

		decoder := goxml.NewDecoder(resp.Body)
		var curArray []string = make([]string, 0)
		var curData goxml.CharData
		i := -1
		for {
			tk, err := decoder.Token()
			if err != nil {
				break
			}
			switch tk.(type) {
			case goxml.StartElement:
				startElem, _ := tk.(goxml.StartElement)
				curPath.AddChild(startElem.Name.Local)
				if curPath.Equals(arrayStartPath) {
					curArray = make([]string, 0)
					i += 1
				}
			case goxml.CharData:
				data, _ := tk.(goxml.CharData)
				curData = data.Copy()
			case goxml.EndElement:
				if curPath.Equals(ValueArrayPath) {
					curArray = append(curArray, string(curData))
				} else if curPath.Equals(arrayStartPath) {
					switch i {
					case 0:
						reply.AddedGroup = curArray
					case 1:
						reply.ChangedGroup = curArray
					case 2:
						reply.RemovedGroup = curArray
					}
				}

				curPath.RemoveLast()
			}
		}
	}
	return
}
