package main

import (
	"os"
	"os/signal"
	"squ/logger"
	"squ/netserver"
	"squ/settings"
	"syscall"
)

func main() {
	path := os.Getenv("CONF")
	var squSettings settings.SettingsProvider = settings.NewJsonSettings(path)
	if squSettings.IsActive() {
		logger.Info("Starting SQU-server.")
		signalChannel := make(chan os.Signal, 1)
		signal.Notify(signalChannel, os.Interrupt)
		signal.Notify(signalChannel, syscall.SIGTERM)
		server := netserver.NewServer(squSettings)
		outSignalsChannel := server.Start()

		alive := true
		for alive {
			select {
			case newSig := <-signalChannel:
				{
					if newSig != nil {
						logger.Info("Signal of termination.")
						alive = false
					}
				}
			case msg := <-outSignalsChannel:
				{
					if len(msg.Msg) > 0 {
						logger.Info("server: %s", msg.Msg)
					}
					if msg.Out {
						alive = false
					}
				}
			}
		}
		close(outSignalsChannel)
		close(signalChannel)
		for server.Stop() {
			logger.Warn("server stopping, wait..")
		}
	}
}
