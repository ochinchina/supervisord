package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/gorilla/mux"
	"github.com/ochinchina/supervisord/config"
)

// DynamicProgramRequest 表示动态程序提交请求
type DynamicProgramRequest struct {
	ProgramName string            `json:"program_name"`
	Command     string            `json:"command"`
	WorkingDir  string            `json:"working_dir,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
	AutoStart   bool              `json:"auto_start,omitempty"`
	AutoRestart bool              `json:"auto_restart,omitempty"`
	NumProcs    int               `json:"num_procs,omitempty"`
}

// PRRTEConfig PRRTE相关配置
type PRRTEConfig struct {
	DVMUri      string            `json:"dvm_uri,omitempty"`
	HostFile    string            `json:"host_file,omitempty"`
	MapBy       string            `json:"map_by,omitempty"`
	BindTo      string            `json:"bind_to,omitempty"`
	ExtraArgs   []string          `json:"extra_args,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
}

// ResourceConfig 资源配置
type ResourceConfig struct {
	CPULimit    string `json:"cpu_limit,omitempty"`
	MemoryLimit string `json:"memory_limit,omitempty"`
	Timeout     int    `json:"timeout,omitempty"`
}

// MPIJobStatus 作业状态
type MPIJobStatus struct {
	JobID      string     `json:"job_id"`
	JobName    string     `json:"job_name"`
	Status     string     `json:"status"` // PENDING, RUNNING, COMPLETED, FAILED, CANCELLED
	SubmitTime time.Time  `json:"submit_time"`
	StartTime  *time.Time `json:"start_time,omitempty"`
	EndTime    *time.Time `json:"end_time,omitempty"`
	ExitCode   *int       `json:"exit_code,omitempty"`
	ErrorMsg   string     `json:"error_msg,omitempty"`
	NumProcs   int        `json:"num_procs"`
	WorkingDir string     `json:"working_dir"`
}

// DVMStatus DVM状态
type DVMStatus struct {
	Status      string     `json:"status"` // STOPPED, STARTING, RUNNING, STOPPING, ERROR
	StartTime   *time.Time `json:"start_time,omitempty"`
	DVMUri      string     `json:"dvm_uri,omitempty"`
	ProcessName string     `json:"process_name,omitempty"`
	ErrorMsg    string     `json:"error_msg,omitempty"`
}

// DynamicProgramAPI 动态程序API
type DynamicProgramAPI struct {
	router      *mux.Router
	supervisor  *Supervisor
	programs    map[string]*DynamicProgramStatus
	progMutex   sync.RWMutex
	progCounter int
}

// DynamicProgramStatus 动态程序状态
type DynamicProgramStatus struct {
	ProgramID   string     `json:"program_id"`
	ProgramName string     `json:"program_name"`
	Command     string     `json:"command"`
	Status      string     `json:"status"` // PENDING, RUNNING, STOPPED, FAILED
	SubmitTime  time.Time  `json:"submit_time"`
	StartTime   *time.Time `json:"start_time,omitempty"`
	WorkingDir  string     `json:"working_dir"`
}

// NewDynamicProgramAPI 创建新的动态程序API实例
func NewDynamicProgramAPI(supervisor *Supervisor) *DynamicProgramAPI {
	return &DynamicProgramAPI{
		router:      mux.NewRouter(),
		supervisor:  supervisor,
		programs:    make(map[string]*DynamicProgramStatus),
		progCounter: 0,
	}
}

// CreateHandler 创建HTTP处理器
func (api *DynamicProgramAPI) CreateHandler() http.Handler {
	// 动态程序管理端点
	api.router.HandleFunc("/dynamic/programs", api.SubmitProgram).Methods("POST")
	api.router.HandleFunc("/dynamic/programs", api.ListPrograms).Methods("GET")
	api.router.HandleFunc("/dynamic/programs/{program_id}", api.GetProgramStatus).Methods("GET")
	api.router.HandleFunc("/dynamic/programs/{program_id}", api.StopProgram).Methods("DELETE")
	api.router.HandleFunc("/dynamic/programs/{program_id}/start", api.StartProgram).Methods("POST")

	return api.router
}

// SubmitProgram 提交动态程序
func (api *DynamicProgramAPI) SubmitProgram(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()

	var progReq DynamicProgramRequest
	if err := json.NewDecoder(req.Body).Decode(&progReq); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	// 验证请求
	if err := api.validateProgramRequest(&progReq); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 生成程序ID
	api.progMutex.Lock()
	api.progCounter++
	programID := fmt.Sprintf("dynamic-prog-%d", api.progCounter)
	api.progMutex.Unlock()

	// 创建程序状态
	progStatus := &DynamicProgramStatus{
		ProgramID:   programID,
		ProgramName: progReq.ProgramName,
		Command:     progReq.Command,
		Status:      "PENDING",
		SubmitTime:  time.Now(),
		WorkingDir:  progReq.WorkingDir,
	}

	// 存储程序状态
	api.progMutex.Lock()
	api.programs[programID] = progStatus
	api.progMutex.Unlock()

	// 异步创建程序
	go api.createProgram(programID, &progReq)

	// 返回程序ID
	response := map[string]string{"program_id": programID, "status": "SUBMITTED"}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ListPrograms 列出所有程序
func (api *DynamicProgramAPI) ListPrograms(w http.ResponseWriter, req *http.Request) {
	api.progMutex.RLock()
	programs := make([]*DynamicProgramStatus, 0, len(api.programs))
	for _, prog := range api.programs {
		programs = append(programs, prog)
	}
	api.progMutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(programs)
}

// GetProgramStatus 获取程序状态
func (api *DynamicProgramAPI) GetProgramStatus(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	programID := vars["program_id"]

	api.progMutex.RLock()
	prog, exists := api.programs[programID]
	api.progMutex.RUnlock()

	if !exists {
		http.Error(w, "Program not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(prog)
}

// StopProgram 停止程序
func (api *DynamicProgramAPI) StopProgram(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	programID := vars["program_id"]

	api.progMutex.Lock()
	prog, exists := api.programs[programID]
	if !exists {
		api.progMutex.Unlock()
		http.Error(w, "Program not found", http.StatusNotFound)
		return
	}

	if prog.Status == "RUNNING" {
		prog.Status = "STOPPED"

		// 停止对应的supervisord进程
		api.stopProgramProcess(programID)
	}
	api.progMutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "STOPPED"})
}

// StartProgram 启动程序
func (api *DynamicProgramAPI) StartProgram(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	programID := vars["program_id"]

	api.progMutex.Lock()
	prog, exists := api.programs[programID]
	if !exists {
		api.progMutex.Unlock()
		http.Error(w, "Program not found", http.StatusNotFound)
		return
	}

	if prog.Status == "STOPPED" || prog.Status == "FAILED" {
		prog.Status = "RUNNING"
		now := time.Now()
		prog.StartTime = &now

		// 启动对应的supervisord进程
		api.startProgramProcess(programID)
	}
	api.progMutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "STARTED"})
}

// validateProgramRequest 验证程序请求
func (api *DynamicProgramAPI) validateProgramRequest(req *DynamicProgramRequest) error {
	if req.ProgramName == "" {
		return fmt.Errorf("program_name is required")
	}
	if req.Command == "" {
		return fmt.Errorf("command is required")
	}
	return nil
}

// createProgram 创建程序
func (api *DynamicProgramAPI) createProgram(programID string, progReq *DynamicProgramRequest) {
	// 更新程序状态
	api.updateProgramStatus(programID, "CREATING")

	// 创建动态程序配置
	programName := fmt.Sprintf("dynamic-%s", programID)
	entry, err := api.createDynamicProgramConfig(programName, progReq)
	if err != nil {
		errMsg := err.Error()
		api.updateProgramStatusWithError(programID, "FAILED", &errMsg)
		return
	}

	// 在supervisord中创建进程
	proc := api.supervisor.procMgr.CreateProcess(api.supervisor.GetSupervisorID(), entry)
	if proc == nil {
		errMsg := "Failed to create process"
		api.updateProgramStatusWithError(programID, "FAILED", &errMsg)
		return
	}

	// 如果设置了自动启动，则启动进程
	if progReq.AutoStart {
		proc.Start(false)
		api.updateProgramStatus(programID, "RUNNING")
	} else {
		api.updateProgramStatus(programID, "STOPPED")
	}
}

// createDynamicProgramConfig 创建动态程序配置
func (api *DynamicProgramAPI) createDynamicProgramConfig(programName string, progReq *DynamicProgramRequest) (*config.Entry, error) {
	entry := &config.Entry{
		Name:      fmt.Sprintf("program:%s", programName),
		Group:     "dynamic",
		ConfigDir: "/tmp",
	}

	// 设置基本配置
	keyValues := map[string]string{
		"command":                 progReq.Command,
		"process_name":            fmt.Sprintf("%s_%%(process_num)02d", programName),
		"numprocs":                "1",
		"autostart":               "false",
		"autorestart":             fmt.Sprintf("%t", progReq.AutoRestart),
		"startsecs":               "1",
		"startretries":            "3",
		"exitcodes":               "0",
		"stopsignal":              "TERM",
		"stopwaitsecs":            "10",
		"stopasgroup":             "true",
		"killasgroup":             "true",
		"redirect_stderr":         "false",
		"stdout_logfile":          fmt.Sprintf("/tmp/%s.log", programName),
		"stderr_logfile":          fmt.Sprintf("/tmp/%s.err", programName),
		"stdout_logfile_maxbytes": "50MB",
		"stderr_logfile_maxbytes": "50MB",
	}

	// 设置工作目录
	if progReq.WorkingDir != "" {
		keyValues["directory"] = progReq.WorkingDir
	} else {
		keyValues["directory"] = "/tmp"
	}

	// 设置环境变量
	if len(progReq.Environment) > 0 {
		envVars := make([]string, 0)
		for k, v := range progReq.Environment {
			envVars = append(envVars, fmt.Sprintf("%s=%s", k, v))
		}
		keyValues["environment"] = strings.Join(envVars, ",")
	}

	// 设置numprocs
	if progReq.NumProcs > 0 {
		keyValues["numprocs"] = strconv.Itoa(progReq.NumProcs)
	}

	// 使用反射设置keyValues字段
	api.setEntryKeyValues(entry, keyValues)

	return entry, nil
}

// updateProgramStatus 更新程序状态
func (api *DynamicProgramAPI) updateProgramStatus(programID, status string) {
	api.progMutex.Lock()
	defer api.progMutex.Unlock()

	prog, exists := api.programs[programID]
	if !exists {
		return
	}

	prog.Status = status
	if status == "RUNNING" && prog.StartTime == nil {
		now := time.Now()
		prog.StartTime = &now
	}
}

// updateProgramStatusWithError 更新程序状态并设置错误信息
func (api *DynamicProgramAPI) updateProgramStatusWithError(programID, status string, errorMsg *string) {
	api.progMutex.Lock()
	defer api.progMutex.Unlock()

	prog, exists := api.programs[programID]
	if !exists {
		return
	}

	prog.Status = status
	// 注意：DynamicProgramStatus结构体需要添加ErrorMsg字段
}

// stopProgramProcess 停止程序进程
func (api *DynamicProgramAPI) stopProgramProcess(programID string) {
	programName := fmt.Sprintf("dynamic-%s", programID)
	procs := api.supervisor.procMgr.FindMatch(programName)
	for _, proc := range procs {
		proc.Stop(false)
	}
}

// startProgramProcess 启动程序进程
func (api *DynamicProgramAPI) startProgramProcess(programID string) {
	programName := fmt.Sprintf("dynamic-%s", programID)
	procs := api.supervisor.procMgr.FindMatch(programName)
	for _, proc := range procs {
		proc.Start(false)
	}
}

// setEntryKeyValues 设置Entry的keyValues字段
func (api *DynamicProgramAPI) setEntryKeyValues(entry *config.Entry, keyValues map[string]string) {
	// 使用反射设置私有字段keyValues
	entryValue := reflect.ValueOf(entry).Elem()
	keyValuesField := entryValue.FieldByName("keyValues")

	if keyValuesField.IsValid() && keyValuesField.CanSet() {
		// 如果字段可以直接设置
		keyValuesField.Set(reflect.ValueOf(keyValues))
	} else {
		// 使用unsafe包设置私有字段
		keyValuesFieldPtr := (*map[string]string)(unsafe.Pointer(uintptr(unsafe.Pointer(entry)) + unsafe.Offsetof(entry.ConfigDir) + unsafe.Sizeof(entry.ConfigDir) + unsafe.Sizeof(entry.Group) + unsafe.Sizeof(entry.Name)))
		*keyValuesFieldPtr = keyValues
	}
}
