package commonserver

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"squ/cmdexecstorage"
	"squ/logger"
	"squ/transport"
	"sync"
	"time"
)

const (
	NetExecuter = iota
	NetRecеiver
)

const (
	// answer error codes
	AnswerCodeFormatError = 1
	AnswerInternalError   = 2
	//
	PauseGetCmd              = 100 // ms
	execRequestChannelVolume = 1024 * 10
)

type SocketTarget struct {
	Post int    `json:"port"`
	Addr string `json:"addr"`
	Type int    `json:"type"`
}

type ConnectionOptions struct {
	BufferSize int
}

func (target *SocketTarget) GetSocket() string {
	return fmt.Sprintf("%s:%d", target.Addr, target.Post)
}

func (target SocketTarget) String() string {
	return target.GetSocket()
}

func (target *SocketTarget) GetTypeName() string {
	switch target.Type {
	case NetRecеiver:
		return "receiver"
	case NetExecuter:
		return "executer"
	default:
		return fmt.Sprintf("unknown type %d", target.Type)
	}
}

// methods
type MethodMap struct {
	storage    map[string]int
	changeLock *sync.RWMutex
}

var onceMethodMap *MethodMap

func (methods *MethodMap) Exists(method string) bool {
	methods.changeLock.RLock()
	defer methods.changeLock.RUnlock()

	if value, exists := methods.storage[method]; exists {
		return value > 0
	} else {
		return false
	}
}

func (methods *MethodMap) Add(method string) {
	methods.changeLock.Lock()
	defer methods.changeLock.Unlock()

	newValue := 1
	if value, exists := methods.storage[method]; exists {
		newValue = value + 1
	}
	methods.storage[method] = newValue
}

func (methods *MethodMap) Delete(method string) {
	methods.changeLock.Lock()
	defer methods.changeLock.Unlock()
	var newValue int
	if value, exists := methods.storage[method]; exists {
		if value-1 < 0 {
			// something wrong
			logger.Warn("Accounting method '%s' problem!", method)
		} else {
			newValue = value - 1
		}
	} else {
		newValue = 0
	}
	methods.storage[method] = newValue
}

func NewMethodMap() *MethodMap {
	if onceMethodMap == nil {
		methods := MethodMap{
			storage:    make(map[string]int),
			changeLock: new(sync.RWMutex)}
		onceMethodMap = &methods
	}
	return onceMethodMap
}

type DataStreamManager struct {
	execRequestChannel *chan transport.Command
	returnedCmdChannel *chan transport.Command
	PutBackHandler     cmdexecstorage.ReturnCommandHandler
}

func (manager *DataStreamManager) GetExecCmd() (bool, *transport.Command) {
	// TODO: priority and suppotred methods
	select {
	case cmd := <-*(manager.returnedCmdChannel):
		return false, &cmd
	case cmd := <-*(manager.execRequestChannel):
		return false, &cmd
	case <-time.After(time.Millisecond * PauseGetCmd):
		return true, nil
	}
}

func NewDataStreamManager() *DataStreamManager {
	backChannel := make(chan transport.Command, execRequestChannelVolume)

	backHandler := func(cmd *transport.Command, task string) {
		backChannel <- (*cmd)
		logger.Warn("Task %s returned with timeout, cmd: %s", task, cmd.String())
	}

	ch1 := make(chan transport.Command, execRequestChannelVolume)
	manager := DataStreamManager{
		execRequestChannel: &ch1,
		returnedCmdChannel: &backChannel,
		PutBackHandler:     backHandler}
	return &manager
}

// state manage
type StateProvider struct {
	updateCount     int
	availableMethod *MethodMap
}

type StateUpdater interface {
	Execute(provider *StateProvider) bool
	HasRollback() bool
	Rollback(provider *StateProvider) bool
}

func (provider *StateProvider) UpdateStateForward(updater StateUpdater) {
	if updater.Execute(provider) {
		provider.updateCount++
	}
}

func (provider *StateProvider) UpdateStateBack(updater StateUpdater) {
	if updater.Rollback(provider) {
		provider.updateCount++
	}
}

func NewStateProvider() *StateProvider {
	provider := StateProvider{
		availableMethod: NewMethodMap()}
	return &provider
}

func (provider *StateProvider) AddSupportedMethod(methodNames ...string) bool {
	result := false
	for _, methodName := range methodNames {
		result = true
		provider.availableMethod.Add(methodName)
	}
	return result
}

func (provider *StateProvider) RemoveSupportedMethod(methodNames ...string) bool {
	result := false
	for _, methodName := range methodNames {
		result = true
		provider.availableMethod.Delete(methodName)
	}
	return result
}

// ----

type CmdHandler func(
	about string,
	cmd *transport.Command,
	dataStreamManager *DataStreamManager) (
	*transport.Answer, StateUpdater, bool)

// main net handler
func NetHandler(
	about string,
	stateProvider *StateProvider,
	dataStreamManager *DataStreamManager,
	connection net.Conn,
	cmdHandler CmdHandler) {
	//
	buffer := bufio.NewReader(connection)
	inLoop := true
	defer connection.Close()
	var stateUpdaters []StateUpdater
	var outVolume uint

	var sendAnswer *transport.Answer
	for inLoop {
		lineData, _, err := buffer.ReadLine()
		if err == nil {
			// ok
			sendAnswer = nil
			if cmd, err := transport.ParseCommand(&lineData); err == nil {
				logger.Debug("cmd: %s => %s", about, cmd)
				answer, stateUpdater, hasChanges := cmdHandler(
					about, cmd, dataStreamManager)
				sendAnswer = answer
				if hasChanges {
					stateProvider.UpdateStateForward(stateUpdater)
					if stateUpdater.HasRollback() {
						stateUpdaters = append(stateUpdaters, stateUpdater)
					}
				}
			} else {
				// skip
				logger.Error("Command parse error: %s", err)
				sendAnswer = transport.NewErrorAnswer(
					cmd.Id, transport.ErrCodeFormat, "Command parse problem.")
			}
			data := sendAnswer.DataDump()
			line := append(*data, byte('\n'))
			if writen, err := connection.Write(line); err == nil {
				outVolume += uint(writen)
			}
		} else {
			inLoop = false
			if err == io.EOF {
				// breaken connection
				logger.Warn("Broken %s", about)
			}
		}
	}
	logger.Debug("Output data size: %d", outVolume)
	n := len(stateUpdaters)
	if n > 0 {
		logger.Debug("Back states: %d", n)
		for i := n - 1; i >= 0; i-- {
			stateProvider.UpdateStateBack(stateUpdaters[i])
		}
	}
}
