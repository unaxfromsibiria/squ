package receiverserver

import (
	common "squ/commonserver"
	"squ/transport"
)

func CommandHandler(
	about string,
	cmd *transport.Command,
	dataStreamManager *common.DataStreamManager) (
	*transport.Answer, common.StateUpdater, bool) {
	//
	return nil, nil, false
}
