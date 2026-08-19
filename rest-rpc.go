package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
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

// SupervisorRestful the restful interface to control the programs defined in configuration file
type SupervisorRestful struct {
	router     *mux.Router
	supervisor *Supervisor
	// key: node name, value: supervisor URL
	remoteSupervisors map[string]string
}

// NewSupervisorRestful create a new SupervisorRestful object
func NewSupervisorRestful(supervisor *Supervisor) *SupervisorRestful {
	return &SupervisorRestful{router: mux.NewRouter(), supervisor: supervisor, remoteSupervisors: make(map[string]string)}
}

func (sr *SupervisorRestful) AddRemoteSupervisors(remoteSupervisors map[string]string) *SupervisorRestful {
	for node, url := range remoteSupervisors {
		sr.remoteSupervisors[node] = url
	}
	return sr
}

// CreateProgramHandler create http handler to process program related restful request
func (sr *SupervisorRestful) CreateProgramHandler() http.Handler {
	sr.router.HandleFunc("/program/list", sr.ListProgram).Methods("GET")
	sr.router.HandleFunc("/program/info/{node}/{name}", sr.GetProgramInfo).Methods("GET")
	sr.router.HandleFunc("/program/start/{node}/{name}", sr.StartProgram).Methods("POST", "PUT")
	sr.router.HandleFunc("/program/stop/{node}/{name}", sr.StopProgram).Methods("POST", "PUT")
	sr.router.HandleFunc("/program/restart/{node}/{name}", sr.RestartProgram).Methods("POST", "PUT")
	sr.router.HandleFunc("/program/log/{node}/{name}/stdout", sr.ReadStdoutLog).Methods("GET")
	sr.router.HandleFunc("/program/log/{node}/{name}/stderr", sr.ReadStderrLog).Methods("GET")
	sr.router.HandleFunc("/program/startPrograms", sr.StartPrograms).Methods("POST", "PUT")
	sr.router.HandleFunc("/program/stopPrograms", sr.StopPrograms).Methods("POST", "PUT")
	return sr.router
}

// CreateSupervisorHandler create http rest interface to control supervisor itself
func (sr *SupervisorRestful) CreateSupervisorHandler() http.Handler {
	sr.router.HandleFunc("/supervisor/listNodes", sr.ListNodes).Methods("GET")
	sr.router.HandleFunc("/supervisor/{node}/ping", sr.PingNode).Methods("GET")
	sr.router.HandleFunc("/supervisor/shutdown", sr.Shutdown).Methods("PUT", "POST")
	sr.router.HandleFunc("/supervisor/reload", sr.Reload).Methods("PUT", "POST")
	sr.router.HandleFunc("/supervisor/{node}/reload", sr.Reload).Methods("PUT", "POST")
	sr.router.HandleFunc("/supervisor/{node}/shutdown", sr.Shutdown).Methods("PUT", "POST")
	return sr.router
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

	url, ok := sr.remoteSupervisors[node]
	if !ok {
		return false
	}
	response, err := http.Get(url + "/supervisor/" + node + "/ping")
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
	nodes := make([]NodeInfo, 0)
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

	nodes = append(nodes, NodeInfo{
		Name:           sr.supervisor.getNodeName(),
		Status:         "ONLINE",
		CpuPercent:     float32(cpuPercent[0]),
		MemTotal:       int(vmStat.Total),
		MemUsed:        float32(vmStat.Used),
		MemUsedPercent: float32(vmStat.UsedPercent),
	})

	for node, url := range sr.remoteSupervisors {
		nodeInfo, err := sr.listRemoteNodeInfo(node, url)
		if err != nil {
			log.WithFields(log.Fields{"node": node}).Warn("failed to get remote node info: ", err)
			nodes = append(nodes, NodeInfo{
				Name:           node,
				Status:         "OFFLINE",
				CpuPercent:     0,
				MemTotal:       0,
				MemUsed:        0,
				MemUsedPercent: 0,
			})
		} else {
			nodes = append(nodes, *nodeInfo)
		}

	}

	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Name < nodes[j].Name
	})
	w.WriteHeader(200)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(nodes)
}

func (sr *SupervisorRestful) listRemoteNodeInfo(node, url string) (*NodeInfo, error) {
	response, err := http.Get(url + "/supervisor/listNodes")
	if err != nil {
		log.WithFields(log.Fields{"node": node, "error": err}).Error("Fail to get node information")
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
	for _, nodeInfo := range nodes {
		if nodeInfo.Name == node {
			return &nodeInfo, nil
		}
	}
	return nil, fmt.Errorf("node not found")
}

func (sr *SupervisorRestful) listRemotePrograms() []types.ProcessInfo {
	programs := make([]types.ProcessInfo, 0)

	for node, url := range sr.remoteSupervisors {
		response, err := http.Get(url + "/program/list")
		if err == nil && response.StatusCode >= 200 && response.StatusCode < 400 {
			var result []types.ProcessInfo

			defer response.Body.Close()
			if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
				log.WithFields(log.Fields{"node": node, "url": url}).Warn("failed to decode remote program list: ", err)
			} else {
				for _, program := range result {
					program.Node = node
					programs = append(programs, program)
				}
			}
		} else {
			log.WithFields(log.Fields{"node": node, "url": url}).Warn("failed to list programs on remote node: ", err)
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
	url, ok := sr.remoteSupervisors[node]
	if !ok {
		return nil, fmt.Errorf("failed to find remote supervisor for node: %s", node)
	}
	response, err := http.Get(url + "/program/info/" + node + "/" + programName)
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
		url, ok := sr.remoteSupervisors[node]
		if !ok {
			log.WithFields(log.Fields{"node": node, "program": program}).Error("Fail to find node")
			return false, nil
		}
		response, err := http.Post(url+"/program/start/"+node+"/"+program, "application/json", nil)
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
		url, ok := sr.remoteSupervisors[node]
		if !ok {
			log.WithFields(log.Fields{"node": node, "program": programName}).Warn("failed to find remote supervisor")
			return false, nil
		}
		response, err := http.Post(url+"/program/stop/"+node+"/"+programName, "application/json", nil)
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

// ReadStdoutLog read the stdout of given program
func (sr *SupervisorRestful) ReadStdoutLog(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	params := mux.Vars(req)
	node := params["node"]
	programName := params["name"]
	if node == "" || node == sr.supervisor.getNodeName() {
		readInfo := ProcessLogReadInfo{Name: programName, Offset: 0, Length: 0}
		reply := struct{ LogData string }{LogData: ""}
		err := sr.supervisor.ReadProcessStdoutLog(req, &readInfo, &reply)
		if err != nil {
			log.WithFields(log.Fields{"node": node, "program": programName}).Warn("failed to read stdout log: ", err)
		}
		w.WriteHeader(200)
		w.Write([]byte(reply.LogData))
	} else {
		// read the stdout log from the remote supervisor
		logData, err := sr.readRemoteLog(node, programName, "stdout")
		if err != nil {
			log.WithFields(log.Fields{"node": node, "program": programName}).Warn("failed to read remote stdout log: ", err)
		}
		w.WriteHeader(200)
		w.Write([]byte(logData))
	}
}

func (sr *SupervisorRestful) readRemoteLog(node, programName, logType string) (string, error) {
	url, ok := sr.remoteSupervisors[node]
	if !ok {
		return "", fmt.Errorf("not a valid node")
	}
	resp, err := http.Get(fmt.Sprintf("%s/program/log/%s/%s/%s", url, node, programName, logType))
	if err != nil {
		return "", fmt.Errorf("failed to read remote %s log: %v", logType, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to read remote %s log: status code %d", logType, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read remote %s log: %v", logType, err)
	}
	return string(data), nil
}

func (sr *SupervisorRestful) ReadStderrLog(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	params := mux.Vars(req)
	node := params["node"]
	programName := params["name"]
	if node == "" || node == sr.supervisor.getNodeName() {
		readInfo := ProcessLogReadInfo{Name: programName, Offset: 0, Length: 0}
		reply := struct{ LogData string }{LogData: ""}
		err := sr.supervisor.ReadProcessStderrLog(req, &readInfo, &reply)
		if err != nil {
			log.WithFields(log.Fields{"node": node, "program": programName}).Warn("failed to read stderr log: ", err)
		}
		w.WriteHeader(200)
		w.Write([]byte(reply.LogData))
	} else {
		// read the stderr log from the remote supervisor
		logData, err := sr.readRemoteLog(node, programName, "stderr")
		if err != nil {
			log.WithFields(log.Fields{"node": node, "program": programName}).Warn("failed to read remote stderr log: ", err)
		}
		w.WriteHeader(200)
		w.Write([]byte(logData))
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
	url, ok := sr.remoteSupervisors[node]
	if !ok {
		return false, fmt.Errorf("not a valid node")
	}
	response, err := http.Post(url+"/supervisor/shutdown", "application/json", nil)
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

// Reload the supervisor configuration file through rest interface
func (sr *SupervisorRestful) Reload(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()

	params := mux.Vars(req)
	node := params["node"]

	if node == "" || node == sr.supervisor.getNodeName() {
		log.Info("reload supervisor configuration")
		_, _, _, err := sr.supervisor.Reload(false)
		if err != nil {
			log.Warn("reload error: ", err)
		}
		r := map[string]bool{"success": err == nil}
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(&r)
	} else {
		// reload the remote supervisor
		success, err := sr.reloadRemote(node)
		if !success || err != nil {
			log.WithFields(log.Fields{"node": node}).Warn("failed to reload remote supervisor: ", err)
		}
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json")
		r := map[string]bool{"success": success}
		_ = json.NewEncoder(w).Encode(&r)
	}

}

func (sr *SupervisorRestful) reloadRemote(node string) (bool, error) {

	url, ok := sr.remoteSupervisors[node]
	if !ok {
		return false, fmt.Errorf("not a valid node")
	}
	response, err := http.Post(url+"/supervisor/reload", "application/json", nil)
	if err != nil {
		log.WithFields(log.Fields{"node": node, "url": url}).Warn("failed to reload remote supervisor: ", err)
		return false, fmt.Errorf("failed to reload remote supervisor: %v", err)
	}
	defer response.Body.Close()

	result := struct{ Success bool }{false}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		log.WithFields(log.Fields{"node": node, "url": url}).Warn("failed to decode response from remote node: ", err)
		return false, fmt.Errorf("failed to decode response from remote node: %v", err)
	}
	return result.Success, nil
}
