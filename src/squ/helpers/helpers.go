package helpers

import (
	sha "crypto/sha1"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

const (
	acceptCahrs            = "abcdefghijkmnpqrstuvwxyz9876543210"
	acceptHexCahrs         = "abcdef9876543210"
	DefaultPasswordSize    = 64
	randPartSize           = 8
	asyncDictCountParts    = 64
	asyncDictCountPartsLim = asyncDictCountParts - 1
	randIntLimit = 1000
)

type SysRandom struct {
	rand.Rand
	mtx *sync.RWMutex
}

var onceRand *SysRandom

// replace base Intn
func (sysRand *SysRandom) safeIntn(n int) int {
	sysRand.mtx.Lock()
	defer sysRand.mtx.Unlock()
	return sysRand.Intn(n)
}

func (sysRand *SysRandom) FromRangeInt(min int, max int) int {
	return min + sysRand.safeIntn(max-min)
}

func (sysRand *SysRandom) Uid() string {
	src := fmt.Sprintf("%s-%d", sysRand.GetShotPrefix(), time.Now().UTC().UnixNano())
	return fmt.Sprintf("%x", sha.Sum([]byte(src)))
}

func (sysRand *SysRandom) getPrefix(base string) string {
	buf := make([]byte, randPartSize)
	n := int(len(base))
	var index int
	for i := 0; i < randPartSize; i++ {
		index = sysRand.safeIntn(n)
		buf[i] = base[index]
	}
	return fmt.Sprintf("%s", buf)
}

func (sysRand *SysRandom) GetShotPrefix() string {
	return sysRand.getPrefix(acceptCahrs)
}

func (sysRand *SysRandom) GetShotHexPrefix() string {
	return sysRand.getPrefix(acceptHexCahrs)
}

func (sysRand *SysRandom) Select(src *[]string) string {
	index := sysRand.FromRangeInt(0, len(*src))
	return (*src)[index]
}

// Simple yes/no answer (normal distribution)
func (sysRand *SysRandom) Question() bool {
	return sysRand.FromRangeInt(0, randIntLimit + 1) > randIntLimit / 2
}

func NewSystemRandom() *SysRandom {
	if onceRand == nil {
		newRand := SysRandom{
			*(rand.New(rand.NewSource(time.Now().UTC().UnixNano()))),
			new(sync.RWMutex)}
		onceRand = &newRand
	}
	return onceRand
}

