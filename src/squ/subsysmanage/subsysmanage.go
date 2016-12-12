package subsysmanage

import (
	"squ/logger"
	"time"
)

const (
	SubSystemCommandNone = iota
	SubSystemCommandStorage
	SubSystemStatistic
)

const (
	SubSystemCommandCode = iota
	SubSystemCommandCodeStop
	SubSystemCommandCodeStatus
	SubSystemCommandCodeStartService
)

const (
	AvgSubSystemsCount = 8
)

// subsystem message
type SubSystemMsg struct {
	Code   int
	System int
}

func NewSubSystemMsg(subSys, code int) *SubSystemMsg {
	return &(SubSystemMsg{Code: code, System: subSys})
}

// iface for subsystems
type SubSystemSwitcher interface {
	CallCommandService(commandCode int, doneChannel *chan SubSystemMsg)
	GetCode() int
}

// halper struct
type SubSystemOwner struct {
	ssInputChannel   chan SubSystemMsg
	activeSubsystems []SubSystemSwitcher
}

func NewSubSystemOwner() *SubSystemOwner {
	result := SubSystemOwner{
		ssInputChannel: make(chan SubSystemMsg, AvgSubSystemsCount)}

	return &result
}

// Registration of subsystem
func (owner *SubSystemOwner) RegSubSystem(subSysem SubSystemSwitcher) {
	(*owner).activeSubsystems = append((*owner).activeSubsystems, subSysem)
}

// send msg to subsystems with callback timeout
// return "true" if all systems will sends back messages
func (owner *SubSystemOwner) SendToSubSystems(commandCode int, timeout int) bool {
	n := 0
	logger.Debug("Send msg %d to subsystems", commandCode)
	for _, ss := range (*owner).activeSubsystems {
		n++
		go ss.CallCommandService(commandCode, &((*owner).ssInputChannel))
	}

	logger.Debug("Messages were sent to %d subsystems, wait answer %d ms.", n, timeout)
	wait := true
	var hasAnswer []int
	var backMsgList []SubSystemMsg
	tout := time.Millisecond * time.Duration(timeout)
	for wait {
		select {
		case msg := <-(*owner).ssInputChannel:
			{
				if msg.Code == commandCode {
					for _, ss := range (*owner).activeSubsystems {
						scode := ss.GetCode()
						if scode == msg.System {
							has := false
							for _, hereSs := range hasAnswer {
								if scode == hereSs {
									has = true
									break
								}
							}
							if has {
								// wtf
								logger.Warn("Code %d already received from %d", commandCode, scode)
								backMsgList = append(backMsgList, msg)
							} else {
								hasAnswer = append(hasAnswer, scode)
								logger.Debug(" => answer from %d", scode)
								wait = len(hasAnswer) < n
							}
						}
					}
				} else {
					backMsgList = append(backMsgList, msg)
				}
			}
		case <-time.After(tout):
			{
				wait = false
				logger.Warn("Can't send %d, have not answer at %d ms.", commandCode, timeout)
			}
		}
	}
	if len(backMsgList) > 0 {
		logger.Debug("return %d messages...", len(backMsgList))
		for _, msg := range backMsgList {
			(*owner).ssInputChannel <- msg
		}
	}
	return len(hasAnswer) >= n
}
