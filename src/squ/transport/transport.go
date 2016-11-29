package transport

import (
	"encoding/json"
	"fmt"
	"squ/logger"
)

const (
	JSONRpcVersion = "2.0"
)

// errors
const (
	ErrCodeNot = iota
	ErrCodeUnknown
	ErrCodeFormat
	ErrCodeProblemDumpJson
)

func dumps(cmd interface{}, endline bool) (string, error) {
	if data, err := json.Marshal(cmd); err != nil {
		return "", err
	} else {
		if endline {
			return fmt.Sprintf("%s\n", data), nil
		} else {
			return string(data), nil
		}
	}
}

type Command struct {
	Jsonrpc string `json:"jsonrpc"`
	Id      int    `json:"id"`
	// used
	Method string `json:"method"`
	Params string `json:"params"`
}

func NewCommand(method string) *Command {
	return &Command{
		Jsonrpc: JSONRpcVersion,
		Method: method,
		Params: "{}"}
}

func (cmd *Command) Dump() (*string, error) {
	(*cmd).Jsonrpc = JSONRpcVersion
	var resultErr error
	var result *string
	if data, err := dumps(cmd, true); err != nil {
		resultErr = err
	} else {
		result = &data
	}
	return result, resultErr
}

func (cmd *Command) DataDump() *[]byte {
	(*cmd).Jsonrpc = JSONRpcVersion
	var result *[]byte
	if data, err := json.Marshal(cmd); err == nil {
		result = &data
	} else {
		result = nil
	}
	return result
}

func (cmd *Command) Load(data *[]byte) error {
	return json.Unmarshal(*data, cmd)
}

type ErrorDescription struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (description *ErrorDescription) Exists() bool {
	return ErrCodeNot != (*description).Code
}

type baseAnswer struct {
	Jsonrpc string `json:"jsonrpc"`
	Id      int    `json:"id"`
	Result  string `json:"result"`
}

type Answer struct {
	baseAnswer
	Error ErrorDescription `json:"error"`
}

func (answer *Answer) DataDump() *[]byte {
	var result *[]byte
	if (*answer).Error.Exists() {
		if data, err := json.Marshal(answer); err == nil {
			result = &data
		} else {
			logger.Error("Answer encode error: %s", err)
		}
	} else {
		baseData := (*answer).baseAnswer
		if data, err := json.Marshal(&baseData); err == nil {
			result = &data
		} else {
			logger.Error("Answer encode error: %s", err)
		}
	}
	return result
}

func (cmd Command) String() string {
	return fmt.Sprintf("Commad(id=%d, method=%s)", cmd.Id, cmd.Method)
}

func NewErrorAnswer(id, code int, msg string) *Answer {
	result := Answer{Error: ErrorDescription{Code: code, Message: msg}}
	result.Jsonrpc = JSONRpcVersion
	result.Id = id
	return &result
}

func NewAnswer(id int, res string) *Answer {
	result := Answer{baseAnswer: baseAnswer{Result: res, Jsonrpc: JSONRpcVersion}}
	result.Id = id
	return &result
}

// Command with task ID
type TaskCommand struct {
	Command
	Task string `json:"task"`
}


/// Answer constructor from command
func PackCmd(cmd *Command, uid string) *Answer {
	result := Answer{
		baseAnswer: baseAnswer{Id: cmd.Id, Jsonrpc: JSONRpcVersion}}
	tcmd := TaskCommand{Command: *cmd, Task: uid}
	if data, err := json.Marshal(&tcmd); err == nil {
		result.Result = string(data)
	} else {
		result.Error = ErrorDescription{
			Code: ErrCodeProblemDumpJson,
			Message: fmt.Sprint("Problem with task command: %s", err)}
	}
	return &result
}

func ParseCommand(content *[]byte) (*Command, error) {
	cmd := Command{}
	err := cmd.Load(content)
	if err == nil {
		return &cmd, nil
	} else {
		return nil, err
	}
}
