package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/ochinchina/supervisord/types"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	log "github.com/sirupsen/logrus"
)

type NodeInfo struct {
	Name           string  `json:"name"`
	Status         string  `json:"status"`
	CpuPercent     float32 `json:"cpu_percent"`
	MemTotal       int     `json:"mem_total"`
	MemUsed        float32 `json:"mem_used"`
	MemUsedPercent float32 `json:"mem_used_percent"`
}

type NodeLoginInfo struct {
	name     string
	url      string
	user     string
	password string
}

func NewNodeLoginInfo(name, url, user, password string) *NodeLoginInfo {
	return &NodeLoginInfo{name: name, url: url, user: user, password: password}
}

type NodeLoginManager struct {
	sync.Mutex
	nodes map[string]*NodeLoginInfo
}

func NewNodeLoginManager() *NodeLoginManager {
	return &NodeLoginManager{nodes: make(map[string]*NodeLoginInfo)}
}

func (nlm *NodeLoginManager) AddNode(node *NodeLoginInfo) {
	nlm.Lock()
	defer nlm.Unlock()
	if _, exists := nlm.nodes[node.name]; !exists {
		log.WithFields(log.Fields{"node": node.name, "url": node.url}).Info("adding a new remote supervisor node")
	}
	nlm.nodes[node.name] = node
}

func (nlm *NodeLoginManager) GetNode(name string) (*NodeLoginInfo, bool) {
	nlm.Lock()
	defer nlm.Unlock()
	node, exists := nlm.nodes[name]
	return node, exists
}

func (nlm *NodeLoginManager) GetAllNode() map[string]*NodeLoginInfo {
	nlm.Lock()
	defer nlm.Unlock()
	nodesCopy := make(map[string]*NodeLoginInfo)
	for name, node := range nlm.nodes {
		nodesCopy[name] = node
	}
	return nodesCopy
}

func (nlm *NodeLoginManager) RemoveNode(name string) {
	nlm.Lock()
	defer nlm.Unlock()
	if node, exists := nlm.nodes[name]; exists {
		log.WithFields(log.Fields{"node": name, "url": node.url}).Info("removing remote supervisor node")
		delete(nlm.nodes, name)
	}
}

func (nlm *NodeLoginManager) GetAllNodeName() []string {
	nlm.Lock()
	defer nlm.Unlock()
	nodeNames := make([]string, 0, len(nlm.nodes))
	for name := range nlm.nodes {
		nodeNames = append(nodeNames, name)
	}
	sort.Strings(nodeNames)
	return nodeNames
}

// SupervisorRestful the restful interface to control the programs defined in configuration file
type SupervisorRestful struct {
	supervisor *Supervisor

	remoteSupervisors *NodeLoginManager
}

// NewSupervisorRestful create a new SupervisorRestful object
func NewSupervisorRestful(supervisor *Supervisor) *SupervisorRestful {
	return &SupervisorRestful{supervisor: supervisor, remoteSupervisors: NewNodeLoginManager()}
}

func (sr *SupervisorRestful) AddRemoteSupervisors(remoteSupervisors map[string]*NodeLoginInfo) *SupervisorRestful {
	for _, info := range remoteSupervisors {
		sr.remoteSupervisors.AddNode(info)
	}
	return sr
}

// CreateProgramHandler create http handler to process program related restful request
func (sr *SupervisorRestful) CreateProgramHandler() http.Handler {

	router := mux.NewRouter()
	router.HandleFunc("/program/list", sr.ListProgram).Methods("GET")
	router.HandleFunc("/program/info/{node}/{name}", sr.GetProgramInfo).Methods("GET")
	router.HandleFunc("/program/info/{name}", sr.GetProgramInfo).Methods("GET")
	router.HandleFunc("/program/start/{node}/{name}", sr.StartProgram).Methods("POST", "PUT")
	router.HandleFunc("/program/start/{name}", sr.StartProgram).Methods("POST", "PUT")
	router.HandleFunc("/program/stop/{node}/{name}", sr.StopProgram).Methods("POST", "PUT")
	router.HandleFunc("/program/stop/{name}", sr.StopProgram).Methods("POST", "PUT")
	router.HandleFunc("/program/restart/{node}/{name}", sr.RestartProgram).Methods("POST", "PUT")
	router.HandleFunc("/program/restart/{name}", sr.RestartProgram).Methods("POST", "PUT")
	router.HandleFunc("/program/log_stdout/{node}/{name}", sr.ReadStdoutLog).Methods("GET")
	router.HandleFunc("/program/log_stdout/{name}", sr.ReadStdoutLog).Methods("GET")
	router.HandleFunc("/program/log_stderr/{node}/{name}", sr.ReadStderrLog).Methods("GET")
	router.HandleFunc("/program/log_stderr/{name}", sr.ReadStderrLog).Methods("GET")
	router.HandleFunc("/program/startPrograms", sr.StartPrograms).Methods("POST", "PUT")
	router.HandleFunc("/program/stopPrograms", sr.StopPrograms).Methods("POST", "PUT")
	return router
}

// CreateSupervisorHandler create http rest interface to control supervisor itself
func (sr *SupervisorRestful) CreateSupervisorHandler() http.Handler {
	router := mux.NewRouter()
	router.HandleFunc("/supervisor/listNodes", sr.ListNodes).Methods("GET")
	router.HandleFunc("/supervisor/{node}/ping", sr.PingNode).Methods("GET")
	router.HandleFunc("/supervisor/shutdown", sr.Shutdown).Methods("PUT", "POST")
	router.HandleFunc("/supervisor/restart", sr.Restart).Methods("PUT", "POST")
	router.HandleFunc("/supervisor/{node}/restart", sr.Restart).Methods("PUT", "POST")
	router.HandleFunc("/supervisor/{node}/shutdown", sr.Shutdown).Methods("PUT", "POST")
	return router
}

func (sr *SupervisorRestful) httpGet(url, user, password string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if user != "" && password != "" {
		req.SetBasicAuth(user, password)
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	return client.Do(req)
}

func (sr *SupervisorRestful) httpPost(url, user, password string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	if user != "" && password != "" {
		req.SetBasicAuth(user, password)
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	return client.Do(req)
}

func (sr *SupervisorRestful) PingNode(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	params := mux.Vars(req)
	node := params["node"]

	_ = json.NewEncoder(w).Encode(map[string]bool{"success": sr.pingNode(node)})
}

func (sr *SupervisorRestful) pingNode(node string) bool {
	if node == "" || node == sr.supervisor.getNodeName() {
		return true
	}

	loginInfo, ok := sr.remoteSupervisors.GetNode(node)
	if !ok {
		return false
	}
	response, err := sr.httpGet(loginInfo.url+"/supervisor/"+node+"/ping", loginInfo.user, loginInfo.password)
	if err != nil {
		log.WithFields(log.Fields{"node": node}).Warn("failed to ping node: ", err)
		return false
	}
	defer response.Body.Close()
	result := struct{ Success bool }{false}

	log.WithFields(log.Fields{"node": node, "statusCode": response.StatusCode}).Info("ping node information")
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		log.WithFields(log.Fields{"node": node}).Warn("failed to decode ping node response: ", err)
		return false
	}
	log.WithFields(log.Fields{"node": node}).Info("ping node result: ", result.Success)
	return result.Success
}

// ListProgram list the status of all the programs
//
// json array to present the status of all programs
func (sr *SupervisorRestful) ListProgram(w http.ResponseWriter, req *http.Request) {
	result := struct{ AllProcessInfo []types.ProcessInfo }{make([]types.ProcessInfo, 0)}
	_ = sr.supervisor.GetAllProcessInfo(nil, nil, &result)
	remotePrograms := sr.listRemotePrograms()
	result.AllProcessInfo = append(result.AllProcessInfo, remotePrograms...)
	sort.Slice(result.AllProcessInfo, func(i, j int) bool {
		programWithNodeI := result.AllProcessInfo[i].Node + "/" + result.AllProcessInfo[i].Name
		programWithNodeJ := result.AllProcessInfo[j].Node + "/" + result.AllProcessInfo[j].Name
		return programWithNodeI < programWithNodeJ
	})
	w.WriteHeader(200)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result.AllProcessInfo)
}

func (sr *SupervisorRestful) ListNodes(w http.ResponseWriter, req *http.Request) {
	nodes := make(map[string]NodeInfo)
	cpuPercent, err := cpu.Percent(time.Second, false) // false = overall CPU usage
	if err != nil {
		log.Warn("failed to get CPU percent: ", err)
		cpuPercent = []float64{0}
	}
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		log.Warn("failed to get virtual memory: ", err)
		vmStat = &mem.VirtualMemoryStat{Total: 0, Used: 0, UsedPercent: 0}
	}

	nodes[sr.supervisor.getNodeName()] = NodeInfo{
		Name:           sr.supervisor.getNodeName(),
		Status:         "ONLINE",
		CpuPercent:     float32(cpuPercent[0]),
		MemTotal:       int(vmStat.Total),
		MemUsed:        float32(vmStat.Used),
		MemUsedPercent: float32(vmStat.UsedPercent),
	}

	for node, loginInfo := range sr.remoteSupervisors.GetAllNode() {
		nodeInfo, err := sr.listRemoteNodeInfo(loginInfo)
		if err != nil {
			log.WithFields(log.Fields{"node": loginInfo.name}).Warn("failed to get remote node info: ", err)

			nodes[node] = NodeInfo{
				Name:           node,
				Status:         "OFFLINE",
				CpuPercent:     0,
				MemTotal:       0,
				MemUsed:        0,
				MemUsedPercent: 0,
			}
		} else {
			for _, info := range nodeInfo {
				nodes[info.Name] = info
			}
		}

	}

	nodesCopy := make([]NodeInfo, 0, len(nodes))
	for _, info := range nodes {
		nodesCopy = append(nodesCopy, info)
	}

	sort.Slice(nodesCopy, func(i, j int) bool {
		return nodesCopy[i].Name < nodesCopy[j].Name
	})
	w.WriteHeader(200)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(nodesCopy)
}

func (sr *SupervisorRestful) listRemoteNodeInfo(loginInfo *NodeLoginInfo) ([]NodeInfo, error) {
	url := fmt.Sprintf("%s/supervisor/listNodes", loginInfo.url)
	response, err := sr.httpGet(url, loginInfo.user, loginInfo.password)
	if err != nil {
		log.WithFields(log.Fields{"node": loginInfo.name, "error": err}).Error("Fail to get node information")
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get remote node info: %s", response.Status)
	}
	var nodes []NodeInfo
	if err := json.NewDecoder(response.Body).Decode(&nodes); err != nil {
		return nil, err
	}

	nodeNameChanged := true

	for _, nodeInfo := range nodes {
		if nodeInfo.Name == loginInfo.name {
			nodeNameChanged = false
		}
		sr.remoteSupervisors.AddNode(NewNodeLoginInfo(nodeInfo.Name, loginInfo.url, loginInfo.user, loginInfo.password))
	}

	if nodeNameChanged {
		sr.remoteSupervisors.RemoveNode(loginInfo.name)
	}

	return nodes, nil
}

func (sr *SupervisorRestful) listRemotePrograms() []types.ProcessInfo {
	programs := make([]types.ProcessInfo, 0)

	for node, loginInfo := range sr.remoteSupervisors.GetAllNode() {
		url := fmt.Sprintf("%s/program/list", loginInfo.url)
		response, err := sr.httpGet(url, loginInfo.user, loginInfo.password)
		if err == nil && response.StatusCode >= 200 && response.StatusCode < 400 {
			var result []types.ProcessInfo

			defer response.Body.Close()
			if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
				log.WithFields(log.Fields{"node": node, "url": loginInfo.url}).Warn("failed to decode remote program list: ", err)
			} else {
				for _, program := range result {
					programs = append(programs, program)
				}
			}
		} else {
			log.WithFields(log.Fields{"node": node, "url": loginInfo.url, "error": err}).Warn("failed to list programs on remote node")
		}
	}
	return programs

}

func (sr *SupervisorRestful) GetProgramInfo(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	params := mux.Vars(req)
	node := params["node"]
	programName := params["name"]
	if node == "" || node == sr.supervisor.getNodeName() {
		result := struct{ ProcInfo types.ProcessInfo }{}
		procInfo := struct{ Name string }{Name: programName}
		err := sr.supervisor.GetProcessInfo(nil, &procInfo, &result)
		if err != nil {
			log.WithFields(log.Fields{"node": node, "program": programName}).Warn("failed to get program info: ", err)
			w.WriteHeader(500)
			w.Header().Set("Content-Type", "application/json")
			result := map[string]string{"error": err.Error()}
			_ = json.NewEncoder(w).Encode(&result)
		} else {
			w.WriteHeader(200)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(&result.ProcInfo)
		}
	} else {
		// get the program info from the remote supervisor
		procInfo, err := sr.getRemoteProgramInfo(node, programName)
		if err != nil {
			log.WithFields(log.Fields{"node": node, "program": programName}).Warn("failed to get program info from remote node: ", err)
			w.WriteHeader(500)
			w.Header().Set("Content-Type", "application/json")
			result := map[string]string{"error": "failed to get program info from remote node"}
			_ = json.NewEncoder(w).Encode(&result)
		} else {
			w.WriteHeader(200)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(&procInfo)
		}
	}
}

func (sr *SupervisorRestful) getRemoteProgramInfo(node, programName string) (*types.ProcessInfo, error) {
	loginInfo, ok := sr.remoteSupervisors.GetNode(node)
	if !ok {
		return nil, fmt.Errorf("failed to find remote supervisor for node: %s", node)
	}
	url := fmt.Sprintf("%s/program/info/%s/%s", loginInfo.url, node, programName)
	response, err := sr.httpGet(url, loginInfo.user, loginInfo.password)
	if err != nil {
		return nil, fmt.Errorf("failed to get program info from remote node: %v", err)
	}
	defer response.Body.Close()
	var result types.ProcessInfo
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response from remote node: %v", err)
	}
	return &result, nil
}

// StartProgram start the given program through restful interface
func (sr *SupervisorRestful) StartProgram(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	params := mux.Vars(req)
	success, err := sr._startProgram(params["node"], params["name"])
	r := map[string]bool{"success": err == nil && success}
	w.WriteHeader(200)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(&r)
}

func (sr *SupervisorRestful) _startProgram(node, program string) (bool, error) {
	log.WithFields(log.Fields{"node": node, "program": program}).Info("start program")
	startArgs := StartProcessArgs{Name: program, Wait: true}

	if node == "" || node == sr.supervisor.getNodeName() {
		result := struct{ Success bool }{false}
		err := sr.supervisor.StartProcess(nil, &startArgs, &result)
		if err != nil {
			log.WithFields(log.Fields{"node": node, "program": program}).Warn("failed to start program: ", err)
		}
		return result.Success, err
	} else {
		// start the program on the remote supervisor
		loginInfo, ok := sr.remoteSupervisors.GetNode(node)
		if !ok {
			log.WithFields(log.Fields{"node": node, "program": program}).Error("Fail to find node")
			return false, nil
		}
		url := fmt.Sprintf("%s/program/start/%s/%s", loginInfo.url, node, program)
		response, err := sr.httpPost(url, loginInfo.user, loginInfo.password, nil)
		if err != nil {
			log.WithFields(log.Fields{"node": node, "program": program}).Warn("failed to start program on remote node: ", err)
			return false, err
		}
		defer response.Body.Close()
		result := struct{ Success bool }{false}
		if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
			log.WithFields(log.Fields{"node": node, "program": program}).Warn("failed to decode response from remote node: ", err)
			return false, err
		}
		return result.Success, nil
	}
}

// StartPrograms start one or more programs through restful interface
func (sr *SupervisorRestful) StartPrograms(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()

	var programs []struct {
		Node    string `json:"node"`
		Program string `json:"program"`
	}
	if err := json.NewDecoder(req.Body).Decode(&programs); err != nil {
		w.WriteHeader(400)
		_, _ = w.Write([]byte("not a valid request"))
	} else {
		for _, program := range programs {
			if _, err := sr._startProgram(program.Node, program.Program); err != nil {
				log.WithField("program", program).Warn("failed to start program: ", err)
			}
		}
		_, _ = w.Write([]byte("Success to start the programs"))
	}
}

// StopProgram stop a program through the restful interface
func (sr *SupervisorRestful) StopProgram(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()

	params := mux.Vars(req)
	success, err := sr._stopProgram(params["node"], params["name"])
	r := map[string]bool{"success": err == nil && success}
	w.WriteHeader(200)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(&r)
}

func (sr *SupervisorRestful) _stopProgram(node, programName string) (bool, error) {
	log.WithFields(log.Fields{"node": node, "program": programName}).Info("stop program")
	stopArgs := StartProcessArgs{Name: programName, Wait: true}

	result := struct{ Success bool }{false}
	if node == "" || node == sr.supervisor.getNodeName() {
		err := sr.supervisor.StopProcess(nil, &stopArgs, &result)
		if err != nil {
			log.WithFields(log.Fields{"node": node, "program": programName}).Warn("failed to stop program: ", err)
		}
		return result.Success, err
	} else {
		// stop the program on the remote supervisor
		loginInfo, ok := sr.remoteSupervisors.GetNode(node)
		if !ok {
			log.WithFields(log.Fields{"node": node, "program": programName}).Warn("failed to find remote supervisor")
			return false, nil
		}
		url := fmt.Sprintf("%s/program/stop/%s/%s", loginInfo.url, node, programName)
		response, err := sr.httpPost(url, loginInfo.user, loginInfo.password, nil)
		if err != nil {
			log.WithFields(log.Fields{"node": node, "program": programName}).Warn("failed to stop program on remote node: ", err)
			return false, err
		}
		defer response.Body.Close()
		result := struct{ Success bool }{false}
		if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
			log.WithFields(log.Fields{"node": node, "program": programName}).Warn("failed to decode response from remote node: ", err)
			return false, err
		}
		return result.Success, nil
	}

}

func (sr *SupervisorRestful) RestartProgram(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	params := mux.Vars(req)
	node := params["node"]
	name := params["name"]

	if _, err := sr._stopProgram(node, name); err != nil {
		log.WithFields(log.Fields{"node": node, "program": name}).Warn("failed to stop program: ", err)
		w.WriteHeader(500)
		w.Header().Set("Content-Type", "application/json")
		result := map[string]bool{"success": false}
		_ = json.NewEncoder(w).Encode(&result)
		return
	}

	if _, err := sr._startProgram(node, name); err != nil {
		log.WithFields(log.Fields{"node": node, "program": name}).Warn("failed to start program: ", err)
		w.WriteHeader(500)
		w.Header().Set("Content-Type", "application/json")
		result := map[string]bool{"success": false}
		_ = json.NewEncoder(w).Encode(&result)
		return
	}

	w.WriteHeader(200)
	w.Header().Set("Content-Type", "application/json")
	result := map[string]bool{"success": true}
	_ = json.NewEncoder(w).Encode(&result)
}

// StopPrograms stop programs through the restful interface
func (sr *SupervisorRestful) StopPrograms(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()

	var programs []struct {
		Node    string `json:"node"`
		Program string `json:"program"`
	}

	if err := json.NewDecoder(req.Body).Decode(&programs); err != nil {
		w.WriteHeader(400)
		_, _ = w.Write([]byte("not a valid request"))
	} else {
		for _, program := range programs {
			if _, err := sr._stopProgram(program.Node, program.Program); err != nil {
				log.WithField("program", program).Warn("failed to stop program: ", err)
			}
		}
		_, _ = w.Write([]byte("Success to stop the programs"))
	}

}

func (sr *SupervisorRestful) readLog(node, programName, logType string) (string, error) {
	if node == "" || node == sr.supervisor.getNodeName() {
		readInfo := ProcessLogReadInfo{Name: programName, Offset: 0, Length: 0}
		reply := struct{ LogData string }{LogData: ""}
		var err error
		switch logType {
		case "stdout":
			err = sr.supervisor.ReadProcessStdoutLog(nil, &readInfo, &reply)
		case "stderr":
			err = sr.supervisor.ReadProcessStderrLog(nil, &readInfo, &reply)
		default:
			return "", fmt.Errorf("invalid log type: %s", logType)
		}
		if err != nil {
			return "", fmt.Errorf("failed to read %s log: %v", logType, err)
		}
		return reply.LogData, nil
	} else {
		return sr.readRemoteLog(node, programName, logType)
	}
}

// ReadStdoutLog read the stdout of given program
func (sr *SupervisorRestful) ReadStdoutLog(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	params := mux.Vars(req)
	node := params["node"]
	programName := params["name"]
	logData, err := sr.readLog(node, programName, "stdout")

	if err != nil {
		w.WriteHeader(500)
		_, _ = fmt.Fprintf(w, "failed to read log: %v", err)
	} else {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(logData))
	}

}

func (sr *SupervisorRestful) readRemoteLog(node, programName, logType string) (string, error) {
	loginInfo, ok := sr.remoteSupervisors.GetNode(node)
	if !ok {
		log.WithFields(log.Fields{"node": node, "program": programName, "log-type": logType}).Error("failed to find remote supervisor")
		return "", fmt.Errorf("not a valid node")
	}
	url := fmt.Sprintf("%s/program/log_stdout/%s/%s", loginInfo.url, node, programName)
	if logType == "stderr" {
		url = fmt.Sprintf("%s/program/log_stderr/%s/%s", loginInfo.url, node, programName)
	}

	resp, err := sr.httpGet(url, loginInfo.user, loginInfo.password)
	if err != nil {
		log.WithFields(log.Fields{"node": node, "url": url, "error": err, "program": programName, "log-type": logType}).Error("failed to read remote log")
		return "", fmt.Errorf("failed to read remote %s log: %v", logType, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.WithFields(log.Fields{"node": node, "url": url, "statusCode": resp.StatusCode, "program": programName, "log-type": logType}).Error("failed to read remote log: status code not OK")
		return "", fmt.Errorf("failed to read remote %s log: status code %d", logType, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.WithFields(log.Fields{"node": node, "url": url, "error": err, "program": programName, "log-type": logType}).Error("failed to read remote log")
		return "", fmt.Errorf("failed to read remote %s log: %v", logType, err)
	}
	return string(data), nil
}

func (sr *SupervisorRestful) ReadStderrLog(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	params := mux.Vars(req)
	node := params["node"]
	programName := params["name"]
	logData, err := sr.readLog(node, programName, "stderr")
	if err != nil {
		w.WriteHeader(500)
		_, _ = fmt.Fprintf(w, "failed to read log: %v", err)
	} else {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(logData))
	}
}

// Shutdown the supervisor itself
func (sr *SupervisorRestful) Shutdown(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()

	params := mux.Vars(req)
	node := params["node"]

	reply := struct{ Ret bool }{false}
	if node == "" || node == sr.supervisor.getNodeName() {
		if err := sr.supervisor.Shutdown(nil, nil, &reply); err != nil {
			log.Warn("shutdown error: ", err)
		}
		result := map[string]bool{"success": reply.Ret}
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(&result)
	} else {
		// shutdown the remote supervisor
		success, err := sr.shutdownRemote(node)
		if !success || err != nil {
			log.WithFields(log.Fields{"node": node}).Warn("failed to shutdown remote supervisor: ", err)
			return
		}
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json")
		r := map[string]bool{"success": success}
		_ = json.NewEncoder(w).Encode(&r)
		return
	}

}

func (sr *SupervisorRestful) shutdownRemote(node string) (bool, error) {
	loginInfo, ok := sr.remoteSupervisors.GetNode(node)
	if !ok {
		return false, fmt.Errorf("not a valid node")
	}
	url := fmt.Sprintf("%s/supervisor/%s/shutdown", loginInfo.url, node)
	response, err := sr.httpPost(url, loginInfo.user, loginInfo.password, nil)
	if err != nil {
		return false, fmt.Errorf("failed to shutdown remote supervisor: %v", err)
	}
	defer response.Body.Close()

	result := struct{ Success bool }{false}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return false, fmt.Errorf("failed to decode response from remote node: %v", err)
	}
	return result.Success, nil
}

// Restart the supervisor through rest interface
func (sr *SupervisorRestful) Restart(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()

	params := mux.Vars(req)
	node := params["node"]

	if node == "" || node == sr.supervisor.getNodeName() {
		log.Info("reload supervisor configuration")
		args := struct{}{}
		reply := struct{ Ret bool }{Ret: true}

		err := sr.supervisor.Restart(req, &args, &reply)
		if err != nil {
			log.Warn("restart error: ", err)
		}
		r := map[string]bool{"success": reply.Ret}
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(&r)
	} else {
		// reload the remote supervisor
		success, err := sr.restartRemote(node)
		if !success || err != nil {
			log.WithFields(log.Fields{"node": node}).Warn("failed to restart remote supervisor: ", err)
		}
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json")
		r := map[string]bool{"success": success}
		_ = json.NewEncoder(w).Encode(&r)
	}

}

func (sr *SupervisorRestful) restartRemote(node string) (bool, error) {

	loginInfo, ok := sr.remoteSupervisors.GetNode(node)
	if !ok {
		return false, fmt.Errorf("not a valid node")
	}
	url := fmt.Sprintf("%s/supervisor/%s/restart", loginInfo.url, node)
	response, err := sr.httpPost(url, loginInfo.user, loginInfo.password, nil)
	if err != nil {
		log.WithFields(log.Fields{"node": node, "url": loginInfo.url}).Warn("failed to restart remote supervisor: ", err)
		return false, fmt.Errorf("failed to restart remote supervisor: %v", err)
	}
	defer response.Body.Close()

	result := struct{ Success bool }{false}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		log.WithFields(log.Fields{"node": node, "url": loginInfo.url}).Warn("failed to decode response from remote node: ", err)
		return false, fmt.Errorf("failed to decode response from remote node: %v", err)
	}
	return result.Success, nil
}
