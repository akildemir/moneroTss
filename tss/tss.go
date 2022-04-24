package tss

import (
	"encoding/base64"
	"errors"
	"fmt"
	"sort"
	"sync"

	bkeygen "github.com/binance-chain/tss-lib/ecdsa/keygen"
	coskey "github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types/bech32/legacybech32"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	tcrypto "github.com/tendermint/tendermint/crypto"

	"github.com/akildemir/moneroTss/common"
	"github.com/akildemir/moneroTss/conversion"
	"github.com/akildemir/moneroTss/messages"
	moneroKeyGen "github.com/akildemir/moneroTss/monero_multi_sig/keygen"
	moneroKeySign "github.com/akildemir/moneroTss/monero_multi_sig/keysign"
	"github.com/akildemir/moneroTss/monitor"
	"github.com/akildemir/moneroTss/p2p"
	"github.com/akildemir/moneroTss/storage"
)

// TssServer is the structure that can provide all keysign and key gen features
type TssServer struct {
	conf              common.TssConfig
	logger            zerolog.Logger
	p2pCommunication  *p2p.Communication
	localNodePubKey   string
	preParams         *bkeygen.LocalPreParams
	tssKeyGenLocker   *sync.Mutex
	stopChan          chan struct{}
	partyCoordinator  *p2p.PartyCoordinator
	stateManager      storage.LocalStateManager
	signatureNotifier *moneroKeySign.SignatureNotifier
	privateKey        tcrypto.PrivKey
	tssMetrics        *monitor.Metric
}

// NewTss create a new instance of Tss
func NewTss(
	comm *p2p.Communication,
	priKey tcrypto.PrivKey,
	baseFolder string,
	conf common.TssConfig,
	preParams *bkeygen.LocalPreParams,
) (*TssServer, error) {
	pk := coskey.PubKey{
		Key: priKey.PubKey().Bytes()[:],
	}

	pubKey, err := sdk.MarshalPubKey(sdk.AccPK, &pk)
	if err != nil {
		return nil, fmt.Errorf("fail to genearte the key: %w", err)
	}

	stateManager, err := storage.NewFileStateMgr(baseFolder)
	if err != nil {
		return nil, fmt.Errorf("fail to create file state manager")
	}

	pc := p2p.NewPartyCoordinator(comm.GetHost(), conf.PartyTimeout)
	sn := moneroKeySign.NewSignatureNotifier(comm.GetHost())
	metrics := monitor.NewMetric()
	if conf.EnableMonitor {
		metrics.Enable()
	}
	tssServer := TssServer{
		conf:              conf,
		logger:            log.With().Str("module", "tss_mn").Logger(),
		p2pCommunication:  comm,
		localNodePubKey:   pubKey,
		preParams:         preParams,
		tssKeyGenLocker:   &sync.Mutex{},
		stopChan:          make(chan struct{}),
		partyCoordinator:  pc,
		stateManager:      stateManager,
		signatureNotifier: sn,
		privateKey:        priKey,
		tssMetrics:        metrics,
	}

	return &tssServer, nil
}

// Start Tss server
func (t *TssServer) Start() error {
	log.Info().Msg("Starting the TSS servers")
	return nil
}

// Stop Tss server
func (t *TssServer) Stop() {
	close(t.stopChan)
	// stop the p2p and finish the p2p wait group
	err := t.p2pCommunication.Stop()
	if err != nil {
		t.logger.Error().Msgf("error in shutdown the p2p server")
	}
	t.partyCoordinator.Stop()
	log.Info().Msg("The Tss and p2p server has been stopped successfully")
}

func (t *TssServer) requestToMsgId(request interface{}) (string, error) {
	var dat []byte
	var keys []string
	switch value := request.(type) {
	case moneroKeyGen.Request:
		keys = value.Keys
	case moneroKeySign.Request:
		msgToSign, err := base64.StdEncoding.DecodeString(value.EncodedTx)
		if err != nil {
			t.logger.Error().Err(err).Msg("error in decode the keysign req")
			return "", err
		}
		keys = value.SignerPubKeys
		dat = msgToSign
	default:
		t.logger.Error().Msg("unknown request type")
		return "", errors.New("unknown request type")
	}
	keyAccumulation := ""
	sort.Strings(keys)
	for _, el := range keys {
		keyAccumulation += el
	}
	dat = append(dat, []byte(keyAccumulation)...)
	return common.MsgToHashString(dat)
}

func (t *TssServer) joinParty(msgID, version string, blockHeight int64, participants []string, threshold int, sigChan chan string) ([]peer.ID, string, error) {
	oldJoinParty, err := conversion.VersionLTCheck(version, messages.NEWJOINPARTYVERSION)
	if err != nil {
		return nil, "", fmt.Errorf("fail to parse the version with error:%w", err)
	}
	if oldJoinParty {
		t.logger.Info().Msg("we apply the leadless join party")
		peerIDs, err := conversion.GetPeerIDsFromPubKeys(participants)
		if err != nil {
			return nil, "NONE", fmt.Errorf("fail to convert pub key to peer id: %w", err)
		}
		var peersIDStr []string
		for _, el := range peerIDs {
			peersIDStr = append(peersIDStr, el.String())
		}
		onlines, err := t.partyCoordinator.JoinPartyWithRetry(msgID, peersIDStr)
		return onlines, "NONE", err
	} else {
		t.logger.Info().Msg("we apply the join party with a leader")

		if len(participants) == 0 {
			t.logger.Error().Msg("we fail to have any participants or passed by request")
			return nil, "", errors.New("no participants can be found")
		}
		peersID, err := conversion.GetPeerIDsFromPubKeys(participants)
		if err != nil {
			return nil, "", errors.New("fail to convert the public key to peer ID")
		}
		var peersIDStr []string
		for _, el := range peersID {
			peersIDStr = append(peersIDStr, el.String())
		}

		return t.partyCoordinator.JoinPartyWithLeader(msgID, blockHeight, peersIDStr, threshold, sigChan)
	}
}

// GetLocalPeerID return the local peer
func (t *TssServer) GetLocalPeerID() string {
	return t.p2pCommunication.GetLocalPeerID()
}
