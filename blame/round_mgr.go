package blame

import (
	"sync"

	"github.com/akildemir/moneroTss/messagesmn"
)

type RoundInfo struct {
	Index    int
	RoundMsg string
}

type RoundMgr struct {
	storedMsg   map[string]*messagesmn.WireMessage
	storeLocker *sync.Mutex
}

func NewTssRoundMgr() *RoundMgr {
	return &RoundMgr{
		storeLocker: &sync.Mutex{},
		storedMsg:   make(map[string]*messagesmn.WireMessage),
	}
}

func (tr *RoundMgr) Get(key string) *messagesmn.WireMessage {
	tr.storeLocker.Lock()
	defer tr.storeLocker.Unlock()
	ret, ok := tr.storedMsg[key]
	if !ok {
		return nil
	}
	return ret
}

func (tr *RoundMgr) Set(key string, msg *messagesmn.WireMessage) {
	tr.storeLocker.Lock()
	defer tr.storeLocker.Unlock()
	tr.storedMsg[key] = msg
}

func (tr *RoundMgr) GetByRound(roundInfo string) []string {
	var standbyNodes []string
	for _, el := range tr.storedMsg {
		if el.RoundInfo == roundInfo {
			standbyNodes = append(standbyNodes, el.Routing.From.Id)
		}
	}
	return standbyNodes
}
