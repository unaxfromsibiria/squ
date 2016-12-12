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
	subsys "squ/subsysmanage"
	"time"
)

const (
	SubSystemStopTimeout = 15
)

type Server struct {
	subsys.SubSystemOwner
	sockets           []common.SocketTarget
	active            bool
	keepAlivePeriod   int
	connectionOptions common.ConnectionOptions
	cmdExecStorage    *cmdexecstorage.CmdExecStorage
}

func NewServer(settings settings.SettingsProvider) *Server {
	server := Server{
		SubSystemOwner:    *(subsys.NewSubSystemOwner()),
		sockets:           settings.GetSockets(),
		connectionOptions: settings.GetConnectionsOptions(),
		keepAlivePeriod:   settings.GetKeepAlivePeriod()}

	logger.Debug("Sockets in conf: %d", len(server.sockets))
	return &server
}

func socketListen(
	sock string,
	sockName string,
	handler common.CmdHandler,
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

func (server *Server) Start() {
	provider := common.NewStateProvider()
	dataStreamManager := common.NewDataStreamManager()
	if (*server).cmdExecStorage == nil {
		cmdStorage := cmdexecstorage.NewCmdExecStorage(
			dataStreamManager.PutBackHandler, false)
		(*server).cmdExecStorage = cmdStorage
		(*server).RegSubSystem(cmdStorage)
	}
	defer (*server).SendToSubSystems(
		subsys.SubSystemCommandCodeStartService, 1000*SubSystemStopTimeout)

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
			provider,
			dataStreamManager,
			time.Duration(server.keepAlivePeriod))
	}
}

func (server *Server) Stop() bool {
	logger.Info("Exit command send to subsystem, wait %d sec.", SubSystemStopTimeout)
	if server.SendToSubSystems(subsys.SubSystemCommandCodeStop, 1000*SubSystemStopTimeout) {
		return true
	} else {
		logger.Warn("Subsystem stoped incorrectly, timeout extended.")
		return false
	}
}
