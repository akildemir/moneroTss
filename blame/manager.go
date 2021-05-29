package blame

import (
	"sync"

	btss "github.com/binance-chain/tss-lib/tss"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Manager struct {
	logger          zerolog.Logger
	blame           *Blame
	lastUnicastPeer map[string][]peer.ID
	shareMgr        *ShareMgr
	roundMgr        *RoundMgr
	partyInfo       *PartyInfo
	PartyIDtoP2PID  map[string]peer.ID
	lastMsgLocker   *sync.RWMutex
	lastMsg         string
	acceptedShares  *sync.Map
}

func NewBlameManager() *Manager {
	return &Manager{
		logger:          log.With().Str("module", "blame_manager").Logger(),
		partyInfo:       nil,
		PartyIDtoP2PID:  make(map[string]peer.ID),
		lastUnicastPeer: make(map[string][]peer.ID),
		shareMgr:        NewTssShareMgr(),
		roundMgr:        NewTssRoundMgr(),
		blame:           &Blame{},
		lastMsgLocker:   &sync.RWMutex{},
		acceptedShares:  &sync.Map{},
	}
}

func (m *Manager) GetBlame() *Blame {
	return m.blame
}

func (m *Manager) GetShareMgr() *ShareMgr {
	return m.shareMgr
}

func (m *Manager) GetRoundMgr() *RoundMgr {
	return m.roundMgr
}

func (m *Manager) GetAcceptShares() *sync.Map {
	return m.acceptedShares
}

func (m *Manager) SetLastMsg(lastMsg string) {
	m.lastMsgLocker.Lock()
	defer m.lastMsgLocker.Unlock()
	m.lastMsg = lastMsg
}

func (m *Manager) GetLastMsg() string {
	m.lastMsgLocker.RLock()
	defer m.lastMsgLocker.RUnlock()
	return m.lastMsg
}

func (m *Manager) SetPartyInfo(party *btss.PartyID, partyIDMap map[string]*btss.PartyID) {
	partyInfo := &PartyInfo{
		Party:      party,
		PartyIDMap: partyIDMap,
	}
	m.partyInfo = partyInfo
}

func (m *Manager) SetLastUnicastPeer(peerID peer.ID, roundInfo string) {
	l, ok := m.lastUnicastPeer[roundInfo]
	if !ok {
		peerList := []peer.ID{peerID}
		m.lastUnicastPeer[roundInfo] = peerList
	} else {
		l = append(l, peerID)
		m.lastUnicastPeer[roundInfo] = l
	}
}
