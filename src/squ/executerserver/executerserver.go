package executerserver

import (
	"encoding/json"
	"fmt"
	"os"
	"squ/cmdexecstorage"
	common "squ/commonserver"
	"squ/helpers"
	"squ/logger"
	"squ/transport"
	"strings"
)

const (
	RegMethodName      = "registration"
	ResultMethodReturn = "result"
	GetExecute         = "execute"
	SendCommand        = "send" // only in debug mode
	CreateUid          = "uid"  // test create uid
)

var debugMode bool

// service format types
type RegParams struct {
	Methods []string
}

// StateUpdater
type MethodRegistrator struct {
	methodNames []string
}

func init() {
	debugMode = strings.ToUpper(os.Getenv("DEBUG")) == "TRUE"
}

func (registrator MethodRegistrator) Execute(provider *common.StateProvider) bool {
	var result bool
	if len(registrator.methodNames) > 0 {
		result = provider.AddSupportedMethod(registrator.methodNames...)
		if logger.DebugLevel {
			msg := "New methods:"
			for _, method := range registrator.methodNames {
				msg = fmt.Sprintf("%s\n  %s +1", msg, method)
			}
			logger.Debug(msg)
		}
	}
	return result
}

func (registrator MethodRegistrator) Rollback(provider *common.StateProvider) bool {
	var result bool
	if len(registrator.methodNames) > 0 {
		result = provider.RemoveSupportedMethod(registrator.methodNames...)
		if logger.DebugLevel {
			msg := "Remove methods:"
			for _, method := range registrator.methodNames {
				msg = fmt.Sprintf("%s\n  %s -1", msg, method)
			}
			logger.Debug(msg)
		}
	}
	return result
}

func (registrator MethodRegistrator) HasRollback() bool {
	return true
}

// result updater
type ResultReturner struct {
	Value string
}

func (returner ResultReturner) HasRollback() bool {
	return false
}

func (returner ResultReturner) Rollback(provider *common.StateProvider) bool {
	return false
}

func (returner ResultReturner) Execute(provider *common.StateProvider) bool {
	return false
}

// main
func CommandHandler(
	about string,
	cmd *transport.Command,
	dataStreamManager *common.DataStreamManager) (
	*transport.Answer, common.StateUpdater, bool) {
	//
	var answer *transport.Answer
	command := (*cmd)
	switch command.Method {
	// registration supported methods for connection
	case RegMethodName:
		{
			params := RegParams{}
			logger.Debug("Registrtion data %s", command.Params)
			if err := json.Unmarshal([]byte(command.Params), &params); err == nil {
				registrator := MethodRegistrator{methodNames: params.Methods}
				answer := transport.NewAnswer(command.Id, "{\"ok\": true}")
				return answer, registrator, true
			} else {
				logger.Error("Format error for %s from %s", command, about)
				answer := transport.NewErrorAnswer(
					0, common.AnswerCodeFormatError, fmt.Sprintf("%s", err))
				return answer, nil, false
			}
		}
	case CreateUid:
		{
			if debugMode {
				rand := helpers.NewSystemRandom()
				answer = transport.NewAnswer(command.Id, rand.Uid())
			} else {
				answer = transport.NewErrorAnswer(
					command.Id, common.AnswerAccessError, "Supported only for debug mode.")
			}
			return answer, nil, false

		}
	case ResultMethodReturn:
		{
			//store := cmdexecstorage.NewCmdExecStorage(nil)
			// free cell

		}
	case SendCommand:
		{
			// only for debug
			if debugMode {
				rand := helpers.NewSystemRandom()
				answer = transport.NewAnswer(command.Id, rand.Uid())
				dataStreamManager.AddCommand(cmd)
			} else {
				answer = transport.NewErrorAnswer(
					command.Id, common.AnswerAccessError, "Supported only for debug mode.")
			}
			return answer, nil, false

		}
	case GetExecute:
		{
			if timeout, cmd := dataStreamManager.GetExecCmd(); timeout {
				// no command
				logger.Debug("no command for %s", about)
				answer = transport.NewAnswer(command.Id, "{\"ok\": false}")
				logger.Debug("answer: %s", answer.String())
			} else {
				// get command and create task id
				uid := helpers.NewSystemRandom().Uid()
				logger.Debug("Execute task in %s for cmd: %s", uid, cmd.Method)
				// timeout can be in cmd
				timeout := helpers.FindTimeout(&(cmd.Params))
				// use once ptr to this store
				store := cmdexecstorage.NewCmdExecStorage(nil, false)
				if store.Push(uid, cmd, timeout) {
					answer = transport.PackCmd(cmd, uid)
				} else {
					logger.Error("Wrong command store at %p", store)
					answer = transport.NewErrorAnswer(
						cmd.Id,
						common.AnswerInternalError,
						"Problem with command data storage")
				}
			}
		}
	}
	return answer, nil, false
}

func init() {
	// make timer channel for GetExecute reject
}
