package cmdexecstorage_test

import (
	"fmt"
	"squ/cmdexecstorage"
	"squ/helpers"
	"squ/transport"
	"strconv"
	"strings"
	"testing"
	"time"
)

// fixture
// my nice pytest I miss you
const (
	UidFixture = `
6944fe8b976ba6a11e7bb702ebb73a8d47d2463d|79|1
1012936e40712e253d61e2334e7521d7c7ae9049|20|0
759b9ff96b9b179016ad95149a7b6b7cbecb580e|153|1
8dd1dcfdf1a7aec55ec4ba51c3c4634b5b72c020|221|1
d2e494f485c247502c644a519bc65d4536e93fb0|233|1
6e0cdf7b9558643c7529e931083657dd9bf4ef11|13|1
1e0cdf7b9558643e931083657dd9bf4ef11|13|1
d20414f485c247502c644a519bc65d4536e93fb0|1|1
c32|0|1
c0298|40|1
`
)

func TestUidParseToIndex(t *testing.T) {
	var hash string
	var testValue int
	var trueCase bool
	for _, line := range strings.Split(UidFixture, "\n") {
		if line != "" {
			for index, part := range strings.Split(line, "|") {
				switch index {
				case 0:
					{
						hash = strings.Trim(part, " ")
					}
				case 1:
					{
						if val, err := strconv.Atoi(part); err == nil {
							testValue = val
						} else {
							testValue = -1
						}
					}
				case 2:
					{
						trueCase = part == "1"
					}
				}
			}
			indexVal := cmdexecstorage.GetMapIndex(hash)
			eq := (indexVal == testValue)
			com := "!="
			if trueCase {
				com = "=="
			}
			t.Logf("%s -> %d %s %d ?", hash, indexVal, com, testValue)
			if trueCase != eq {
				t.Error("compare is failed!")
			}
		}
	}
}

func TestCmdExecStorageSimpleCmdReturn(t *testing.T) {
	var cmdPtrStr string
	backHandler := func(cmd *transport.Command, task string) {
		// returned
		cmdPtr := fmt.Sprintf("%p", cmd)
		t.Logf("return cmd ptr: %s", cmdPtr)
		if cmdPtr != cmdPtrStr {
			t.Error("Unknown command returned!")
		}
	}
	storage := cmdexecstorage.NewCmdExecStorage(backHandler, true)
	t.Logf("Storage %p run.", storage)
	time.Sleep(500 * time.Millisecond)
	cmd := transport.NewCommand("test_1")
	cmdPtrStr = fmt.Sprintf("%p", cmd)
	t.Logf("cmd ptr: %s", cmdPtrStr)
	storage.Push("1012936e40712e253d61e2334e7521d7c7ae9049", cmd, 600)
	time.Sleep(700 * time.Millisecond)
	storage.Stop()
}

func TestCmdExecStorageAsyncCmdReturn(t *testing.T) {
	var returned int
	backHandler := func(cmd *transport.Command, task string) {
		if fmt.Sprintf("%p", cmd) == (*cmd).Params {
			returned++
		}
	}
	fillMethod := func(timeout int, count int, stor *cmdexecstorage.CmdExecStorage) {
		rand := helpers.NewSystemRandom()
		for index := 0; index < count; index++ {
			methodName := fmt.Sprintf("method_%d", index)
			id := rand.Uid()
			cmd := transport.NewCommand(methodName)
			cmd.Params = fmt.Sprintf("%p", cmd)
			stor.Push(id, cmd, timeout)
		}
	}
	storage := cmdexecstorage.NewCmdExecStorage(backHandler, true)
	groupSize := 100
	groupCount := 3
	t.Logf("Storage %p run.", storage)
	time.Sleep(100 * time.Millisecond)
	// run
	for index := 0; index < groupCount; index++ {
		go fillMethod(300+index*5, groupSize, storage)
		time.Sleep(5 * time.Millisecond)
	}
	time.Sleep(time.Duration((groupCount+1)*300) * time.Millisecond)
	t.Logf("Returned: %d", returned)
	if groupSize*groupCount != returned {
		t.Error("Error with returned command.")
	}
	if storage.Volume() > 0 {
		t.Error("Storage must be empty!")
	}
	storage.Stop()
}

func TestCmdExecStorageAsyncCmdFree(t *testing.T) {
	var returned int
	var free int

	backHandler := func(cmd *transport.Command, task string) {
		if fmt.Sprintf("%p", cmd) == (*cmd).Params {
			returned++
		}
	}

	fillMethod := func(
		timeout int,
		count int,
		stor *cmdexecstorage.CmdExecStorage,
		freeChannel *chan string) {
		//
		rand := helpers.NewSystemRandom()
		for index := 0; index < count; index++ {
			methodName := fmt.Sprintf("method_%d", index)
			id := rand.Uid()
			cmd := transport.NewCommand(methodName)
			cmd.Params = fmt.Sprintf("%p", cmd)
			stor.Push(id, cmd, timeout)
			if rand.Question() {
				(*freeChannel) <- id
			}
		}
	}
	//
	clearWorker := func(
		stor *cmdexecstorage.CmdExecStorage,
		freeChannel *chan string) {
		//
		for id := range *freeChannel {
			if id == "" {
				break
			} else {
				if stor.Free(id) {
					free++
				}
			}
		}
	}
	storage := cmdexecstorage.NewCmdExecStorage(backHandler, true)
	groupSize := 100
	groupCount := 3
	t.Logf("Storage %p run.", storage)
	time.Sleep(100 * time.Millisecond)
	freeCmdChannel := make(chan string, groupCount*groupSize)
	// run
	go clearWorker(storage, &freeCmdChannel)
	for index := 0; index < groupCount; index++ {
		go fillMethod(300+index*5, groupSize, storage, &freeCmdChannel)
		time.Sleep(5 * time.Millisecond)
	}
	time.Sleep(time.Duration((groupCount+1)*300) * time.Millisecond)
	t.Logf("Returned: %d Free: %d", returned, free)
	if groupSize*groupCount != returned+free {
		t.Error("Error with returned/free command.")
	}
	if storage.Volume() > 0 {
		t.Error("Storage must be empty!")
	}
	storage.Stop()
}
