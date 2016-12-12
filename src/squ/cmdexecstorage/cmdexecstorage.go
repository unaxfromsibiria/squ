package cmdexecstorage

import (
	"math"
	"os"
	"squ/logger"
	subsys "squ/subsysmanage"
	"squ/transport"
	"strconv"
	"sync"
	"time"
)

const (
	defaultStoppingDelay = 25 //ms
)

// find max positive int
func simplePosMax(arr *[]int) int {
	var result int
	for _, a := range *arr {
		if a > result {
			result = a
		}
	}
	return result
}

const (
	DefaultClearIterTimeout = 250 // ms
)

// get n char from hash - i's index of target map in array
var HashHexPositions []int
var MapsCount int
var MinHashSize int
var ClearIterTimeout int

func init() {
	HashHexPositions = []int{2, 4}
	MapsCount = int(math.Pow(16.0, float64(len(HashHexPositions))))
	MinHashSize = simplePosMax(&HashHexPositions) + 1

	ClearIterTimeout = DefaultClearIterTimeout
	if val, err := strconv.Atoi(os.Getenv("STORAGE_ITER_TIME")); err == nil {
		if val > 0 {
			ClearIterTimeout = val
		}
	}
}

type ReturnCommandHandler func(cmd *transport.Command, task string)

// get hex char from position and create int index value
func GetMapIndex(hexStr string) int {
	var result int
	if len(hexStr) >= MinHashSize {
		hexB := []byte(hexStr)
		var harr []byte
		for _, index := range HashHexPositions {
			harr = append(harr, hexB[index])
		}
		if val, err := strconv.ParseInt(string(harr), 16, 32); err == nil {
			result = int(val)
		} else {
			logger.Error("Hash parse error: %s", err)
		}
	}
	return result
}

//cmd info
type cmdInfo struct {
	cmd       *transport.Command
	waitIndex int
	waitLimit int
}

// storage cell
type cellMap struct {
	lock    *sync.RWMutex
	storage map[string]cmdInfo
}

func (cell *cellMap) push(hash string, cmd *transport.Command, timeLimit int) {
	cellmap := *cell
	cellmap.lock.Lock()
	defer cellmap.lock.Unlock()
	iterLimit := int(float32(timeLimit) / float32(ClearIterTimeout))
	cellmap.storage[hash] = cmdInfo{
		cmd:       cmd,
		waitLimit: iterLimit}
}

func (cell *cellMap) remove(hash string) bool {
	cellmap := *cell
	cellmap.lock.Lock()
	defer cellmap.lock.Unlock()

	_, exists := cellmap.storage[hash]
	if exists {
		delete(cellmap.storage, hash)
	}
	return exists
}

func (cell *cellMap) size() int {
	cellmap := *cell
	cellmap.lock.Lock()
	defer cellmap.lock.Unlock()
	return len(cellmap.storage)
}

// Increase wait index.
func (cell *cellMap) incWaitIndex() {
	cellmap := *cell
	cellmap.lock.Lock()
	defer cellmap.lock.Unlock()
	for hash, val := range cellmap.storage {
		val.waitIndex++
		cellmap.storage[hash] = val
	}
}

// Get all command, where wait limit extended.
func (cell *cellMap) getOld() map[string]*transport.Command {
	cellmap := *cell
	cellmap.lock.RLock()
	defer cellmap.lock.RUnlock()
	result := make(map[string]*transport.Command)
	for hash, _ := range cellmap.storage {
		if cellmap.storage[hash].waitIndex > cellmap.storage[hash].waitLimit {
			result[hash] = cellmap.storage[hash].cmd
		}
	}
	return result
}

// Free cell for such hash set.
func (cell *cellMap) free(hashSet ...string) {
	cellmap := *cell
	cellmap.lock.Lock()
	defer cellmap.lock.Unlock()
	for _, hash := range hashSet {
		delete(cellmap.storage, hash)
	}
}

// Iteration for search old command
func (cell *cellMap) clearIteration() map[string]*transport.Command {
	cell.incWaitIndex()
	result := cell.getOld()
	hashSet := make([]string, len(result))
	i := 0
	for h, _ := range result {
		hashSet[i] = h
		i++
	}
	cell.free(hashSet...)
	return result
}

func newCellMap() *cellMap {
	result := cellMap{
		storage: make(map[string]cmdInfo),
		lock:    new(sync.RWMutex)}
	return &result
}

// Storage for command in clients task
type CmdExecStorage struct {
	cells         []*cellMap
	returnHandler ReturnCommandHandler
	exitChannel   chan bool
	active        bool
}

// add command to store for saving at >= timeLimit
func (storage *CmdExecStorage) Push(hash string, cmd *transport.Command, timeLimit int) bool {
	if !(*storage).active {
		return false
	}
	mapIndex := GetMapIndex(hash)
	cellRef := (*storage).cells[mapIndex]
	cellRef.push(hash, cmd, timeLimit)
	return true
}

// Stop all activity and close channel
func (storage *CmdExecStorage) ForceStop() {
	(*storage).exitChannel <- true
	time.Sleep(time.Millisecond * time.Duration(ClearIterTimeout))
}

// Free cell in storage
func (storage *CmdExecStorage) Free(hash string) bool {
	mapIndex := GetMapIndex(hash)
	cellRef := (*storage).cells[mapIndex]
	return cellRef.remove(hash)
}

// Volume of storage
func (storage *CmdExecStorage) Volume() int {
	var result int
	for index := 0; index < MapsCount; index++ {
		result += (*storage).cells[index].size()
	}
	return result
}

// run watching and periodic clearing
func (storage *CmdExecStorage) run() {
	(*storage).active = true
	active := true
	index := 0
	logger.Debug("Storage at %p started.", storage)
	for active {
		select {
		case <-(*storage).exitChannel:
			{
				active = false
			}
		case <-time.After(time.Millisecond * time.Duration(ClearIterTimeout)):
			{
				for index = 0; index < MapsCount; index++ {
					cmdMap := (*storage).cells[index].clearIteration()
					if len(cmdMap) > 0 {
						// big delay will be here?
						for hash, cmdRef := range cmdMap {
							(*storage).returnHandler(cmdRef, hash)
						}
					}
				}
			}
		}
	}
	(*storage).active = false
	close((*storage).exitChannel)
	logger.Debug("Storage at %p stopped.", storage)
}

var onceStorage *CmdExecStorage

// New storage for command with
// rhandler - rollback handler (for processing comman after timeout event)
// refresh - params used for testing, avoid using it
func NewCmdExecStorage(
	rhandler ReturnCommandHandler,
	refresh bool) *CmdExecStorage {
	//
	var store *CmdExecStorage
	if onceStorage != nil && !refresh {
		store = onceStorage
	} else {
		withProblem := rhandler == nil
		if withProblem {
			logger.Error("Empty handler for comand return back!")
		}
		result := CmdExecStorage{
			returnHandler: rhandler,
			active:        !withProblem,
			exitChannel:   make(chan bool, 1),
			cells:         make([]*cellMap, MapsCount)}
		for index := 0; index < MapsCount; index++ {
			result.cells[index] = newCellMap()
		}
		onceStorage = &result
		if !withProblem {
			go result.run()
		}
		store = &result
	}
	logger.Debug("Store for executed command at %p", store)
	return store
}

// subsys SubSystemSwitcher
func (storage *CmdExecStorage) CallCommandService(commandCode int, doneChannel *chan subsys.SubSystemMsg) {
	ssCode := storage.GetCode()
	switch commandCode {
	case subsys.SubSystemCommandCodeStop:
		{
			storage.ForceStop()
			delay := time.Millisecond * time.Duration(ClearIterTimeout)
			for (*storage).active {
				time.Sleep(delay)
			}
			(*doneChannel) <- *(subsys.NewSubSystemMsg(ssCode, subsys.SubSystemCommandCodeStop))

		}
	case subsys.SubSystemCommandCodeStartService:
		{
			logger.Debug("Storage at %p has command for starting.", storage)
			(*doneChannel) <- *(subsys.NewSubSystemMsg(ssCode, subsys.SubSystemCommandCodeStartService))
		}
	default:
		{
			logger.Warn("Storage at %p got unsupported command %d", storage, commandCode)
		}

	}
}

func (storage *CmdExecStorage) GetCode() int {
	return subsys.SubSystemCommandStorage
}
