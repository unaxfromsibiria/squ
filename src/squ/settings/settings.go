package settings

import (
	"encoding/json"
	"io/ioutil"
	common "squ/commonserver"
	"squ/logger"
)

const (
	DefaultKeepAlivePeriod = 60
)

type settingsSrc struct {
	Name    string                `json:"name"`
	Sockets []common.SocketTarget `json:"sockets"`
}

type JsonFileSettings struct {
	src *settingsSrc
}

func (settings JsonFileSettings) GetName() string {
	if settings.src == nil {
		return ""
	} else {
		return settings.src.Name
	}
}

func (settings JsonFileSettings) IsActive() bool {
	if settings.src == nil {
		return false
	} else {
		return true
	}
}

func (settings JsonFileSettings) GetSockets() []common.SocketTarget {
	if settings.src != nil {
		return settings.src.Sockets
	} else {
		return make([]common.SocketTarget, 0)
	}
}

func (settings JsonFileSettings) GetConnectionsOptions() common.ConnectionOptions {
	return common.ConnectionOptions{BufferSize: 1024}
}

func (settings JsonFileSettings) GetKeepAlivePeriod() int {
	return DefaultKeepAlivePeriod
}

func NewJsonSettings(filePath string) *JsonFileSettings {
	if len(filePath) < 1 {
		logger.Terminate("Empty JSON file path.")
	}
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		logger.Terminate("Can't open settings: %s", err)
	}
	src := settingsSrc{}
	err = json.Unmarshal(content, &src)
	if err != nil {
		logger.Terminate("Json load from file % error: %s", filePath, err)
	}
	settings := JsonFileSettings{src: &src}
	return &settings
}

type SettingsProvider interface {
	GetName() string
	IsActive() bool
	GetSockets() []common.SocketTarget
	GetKeepAlivePeriod() int
	GetConnectionsOptions() common.ConnectionOptions
}
