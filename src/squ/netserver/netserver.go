package netserver

import (
	"fmt"
	"net"
	"squ/cmdexecstorage"
	common "squ/commonserver"
	executer "squ/executerserver"
	"squ/logger"
	receiver "squ/receiverserver"
	"squ/settings"
	"time"
)

const (
	SubSystemCommandStorage = iota
	SubSystemStatistic
)

const (
	SubSystemCallExit = iota
	SubSystemCallStatus
)

type Server struct {
	sockets              []common.SocketTarget
	active               bool
	keepAlivePeriod      int
	connectionOptions    common.ConnectionOptions
	subSystems           []int
	subSystemDoneChannel chan int
	subSystemCallChannel chan int
	cmdExecStorage *cmdexecstorage.CmdExecStorage
}

type ExternalSysMsg struct {
	Out bool
	Msg string
}

func NewServer(settings settings.SettingsProvider) *Server {
	server := Server{
		subSystems:        []int{SubSystemCommandStorage},
		sockets:           settings.GetSockets(),
		connectionOptions: settings.GetConnectionsOptions(),
		keepAlivePeriod:   settings.GetKeepAlivePeriod()}
	server.subSystemDoneChannel = make(chan int, len(server.subSystems))
	server.subSystemCallChannel = make(chan int, len(server.subSystems))
	logger.Debug("Sockets in conf: %d", len(server.sockets))
	return &server
}

func socketListen(
	sock string,
	sockName string,
	handler common.CmdHandler,
	outMsgChannel *chan ExternalSysMsg,
	provider *common.StateProvider,
	dataStreamManager *common.DataStreamManager,
	keepAlivePeriod time.Duration) {
	//
	listener, err := net.Listen("tcp", sock)
	if err != nil {
		logger.Terminate("Can't open connection: %s", err)
	} else {
		defer listener.Close()
		active := true
		for active {
			newConnection, err := listener.Accept()
			if err != nil {
				logger.Terminate(
					"Can't create connection to %s error: %s", sock, err)
			} else {
				clientAddr := fmt.Sprintf(
					"connection:%s type: %s", newConnection.RemoteAddr(), sockName)
				logger.Info("new %s", clientAddr)
				tcpConnection := newConnection.(*net.TCPConn)
				tcpConnection.SetKeepAlive(true)
				tcpConnection.SetKeepAlivePeriod(keepAlivePeriod * time.Second)
				// ---
				go common.NetHandler(
					clientAddr, provider, dataStreamManager, tcpConnection, handler)
			}
		}
	}
}

func (server *Server) Start() chan ExternalSysMsg {
	provider := common.NewStateProvider()
	dataStreamManager := common.NewDataStreamManager()
	if (*server).cmdExecStorage == nil {
		cmdStorage := cmdexecstorage.NewCmdExecStorage(
			dataStreamManager.PutBackHandler, false)
		(*server).cmdExecStorage = cmdStorage
	}

	outChannel := make(chan ExternalSysMsg, 1)

	for _, socketTarget := range server.sockets {
		logger.Debug("conf => %s", socketTarget)
		var handler common.CmdHandler
		switch socketTarget.Type {
		case common.NetRecеiver:
			handler = receiver.CommandHandler
		case common.NetExecuter:
			handler = executer.CommandHandler
		default:
			logger.Terminate("Unknown type %d server at %s", socketTarget.Type, socketTarget)
		}
		go socketListen(
			socketTarget.GetSocket(),
			socketTarget.GetTypeName(),
			handler,
			&outChannel,
			provider,
			dataStreamManager,
			time.Duration(server.keepAlivePeriod))
	}
	return outChannel
}

func (server *Server) Stop() bool {
	systemsDoneCount := 0
	server.cmdExecStorage.Stop()
	for systemsDoneCount < len((*server).subSystems) {
		(*server).subSystemCallChannel <- SubSystemCallExit
		logger.Info("exit command send to subsystem")
		subSystemAnswer := <-(*server).subSystemDoneChannel
		for _, systemCode := range (*server).subSystems {
			if subSystemAnswer == systemCode {
				systemsDoneCount++
				logger.Info("subsystem %d ready for exit", subSystemAnswer)
			}
		}
	}
	return true
}
