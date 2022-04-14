package keygen

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	bkg "github.com/binance-chain/tss-lib/ecdsa/keygen"
	btss "github.com/binance-chain/tss-lib/tss"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	tcrypto "github.com/tendermint/tendermint/crypto"

	moneroWallet "github.com/haven-protocol-org/go-haven-rpc-client/wallet"

	"github.com/akildemir/moneroTss/blame"
	"github.com/akildemir/moneroTss/common"
	"github.com/akildemir/moneroTss/conversion"
	"github.com/akildemir/moneroTss/messages"
	"github.com/akildemir/moneroTss/monero_multi_sig"
	"github.com/akildemir/moneroTss/p2p"
	"github.com/akildemir/moneroTss/storage"
)

type MoneroKeyGen struct {
	logger             zerolog.Logger
	localNodePubKey    string
	preParams          *bkg.LocalPreParams
	moneroCommonStruct *common.TssCommon
	stopChan           chan struct{} // channel to indicate whether we should stop
	localParty         *btss.PartyID
	stateManager       storage.LocalStateManager
	commStopChan       chan struct{}
	p2pComm            *p2p.Communication
}

func NewMoneroKeyGen(localP2PID string,
	conf common.TssConfig,
	localNodePubKey string,
	broadcastChan chan *messages.BroadcastMsgChan,
	stopChan chan struct{},
	msgID string,
	stateManager storage.LocalStateManager,
	privateKey tcrypto.PrivKey,
	p2pComm *p2p.Communication) *MoneroKeyGen {
	return &MoneroKeyGen{
		logger: log.With().
			Str("module", "keygen").
			Str("msgID", msgID).Logger(),
		localNodePubKey:    localNodePubKey,
		moneroCommonStruct: common.NewTssCommon(localP2PID, broadcastChan, conf, msgID, privateKey),
		stopChan:           stopChan,
		localParty:         nil,
		stateManager:       stateManager,
		commStopChan:       make(chan struct{}),
		p2pComm:            p2pComm,
	}
}

func (tKeyGen *MoneroKeyGen) GetTssKeyGenChannels() chan *p2p.Message {
	return tKeyGen.moneroCommonStruct.TssMsg
}

func (tKeyGen *MoneroKeyGen) GetTssCommonStruct() *common.TssCommon {
	return tKeyGen.moneroCommonStruct
}

func (tKeyGen *MoneroKeyGen) packAndSend(info string, exchangeRound int, localPartyID *btss.PartyID, msgType string) error {
	sendShare := common.MoneroShare{
		MultisigInfo:  info,
		MsgType:       msgType,
		ExchangeRound: exchangeRound,
	}
	msg, err := json.Marshal(sendShare)
	if err != nil {
		tKeyGen.logger.Error().Err(err).Msg("fail to encode the wallet share")
		return err
	}

	r := btss.MessageRouting{
		From:        localPartyID,
		IsBroadcast: true,
	}
	tKeyGen.moneroCommonStruct.GetBlameMgr().SetLastMsg(msgType)
	return tKeyGen.moneroCommonStruct.ProcessOutCh(msg, &r, msgType, messages.TSSKeyGenMsg)
}

func (tKeyGen *MoneroKeyGen) GenerateNewKey(keygenReq Request) (string, string, error) {
	partiesID, localPartyID, err := conversion.GetParties(keygenReq.Keys, tKeyGen.localNodePubKey)
	if err != nil {
		return "", "", fmt.Errorf("fail to get keygen parties: %w", err)
	}

	threshold, err := conversion.GetThreshold(len(partiesID))
	if err != nil {
		return "", "", err
	}

	// since the definition of threshold of monero is different from the original tss, we need to adjust it 1 more node
	threshold += 1

	// now we try to connect to the monero wallet rpc client
	client := moneroWallet.New(moneroWallet.Config{
		Address: keygenReq.RpcAddress,
	})

	walletName := tKeyGen.localNodePubKey + "-" + keygenReq.KeygenHeight + ".mo"
	passcode := tKeyGen.GetTssCommonStruct().GetNodePrivKey()
	walletDat := moneroWallet.RequestCreateWallet{
		Filename: walletName,
		Password: passcode,
		Language: "English",
	}
	err = client.CreateWallet(&walletDat)
	if err != nil {
		return "", "", err
	}

	defer func() {
		err := client.CloseWallet()
		if err != nil {
			tKeyGen.logger.Error().Err(err).Msg("fail to close the created wallet")
		}
	}()

	blameMgr := tKeyGen.moneroCommonStruct.GetBlameMgr()

	partyIDMap := conversion.SetupPartyIDMap(partiesID)
	err1 := conversion.SetupIDMaps(partyIDMap, tKeyGen.moneroCommonStruct.PartyIDtoP2PID)
	err2 := conversion.SetupIDMaps(partyIDMap, blameMgr.PartyIDtoP2PID)
	if err1 != nil || err2 != nil {
		tKeyGen.logger.Error().Msgf("error in creating mapping between partyID and P2P ID")
		return "", "", err
	}

	partyInfo := &common.PartyInfo{
		Party:      localPartyID,
		PartyIDMap: partyIDMap,
	}

	tKeyGen.moneroCommonStruct.SetPartyInfo(partyInfo)
	blameMgr.SetPartyInfo(localPartyID, partyIDMap)
	tKeyGen.moneroCommonStruct.P2PPeers = conversion.GetPeersID(tKeyGen.moneroCommonStruct.PartyIDtoP2PID, tKeyGen.moneroCommonStruct.GetLocalPeerID())
	// start keygen
	defer tKeyGen.logger.Debug().Msg("generate monero share")

	moneroShareChan := make(chan *common.MoneroShare, len(partiesID))

	var address string

	var keyGenWg sync.WaitGroup
	keyGenWg.Add(1)
	go func() {
		tKeyGen.moneroCommonStruct.ProcessInboundmessages(tKeyGen.commStopChan, &keyGenWg, moneroShareChan)
	}()

	share, err := client.PrepareMultisig()
	if err != nil {
		return "", "", err
	}

	var exchangeRound int32
	exchangeRound = 0
	err = tKeyGen.packAndSend(share.MultisigInfo, int(exchangeRound), localPartyID, common.MoneroKeyGenSharepre)
	if err != nil {
		return "", "", err
	}
	exchangeRound += 1

	var globalErr error
	peerNum := len(partiesID) - 1
	shareStore := monero_multi_sig.GenMoneroShareStore()
	keyGenWg.Add(1)
	go func() {
		defer keyGenWg.Done()
		for {
			select {
			case <-time.After(tKeyGen.GetTssCommonStruct().GetConf().KeyGenTimeout):
				close(tKeyGen.commStopChan)
				globalErr = errors.New("keygen timeout")

				return

			case share := <-moneroShareChan:
				switch share.MsgType {
				case common.MoneroKeyGenSharepre:
					currentRound := atomic.LoadInt32(&exchangeRound)
					shares, ready := shareStore.StoreAndCheck(int(currentRound)-1, share, peerNum)
					if !ready {
						continue
					}
					dat := make([]string, len(shares))
					for i, el := range shares {
						dat[i] = el.MultisigInfo
					}
					request := moneroWallet.RequestMakeMultisig{
						MultisigInfo: dat,
						Threshold:    uint64(threshold),
						Password:     passcode,
					}
					resp, err := client.MakeMultisig(&request)
					if err != nil {
						globalErr = err
						return
					}
					currentMsgType := common.MoneroKeyGenShareExchange + "@" + strconv.FormatInt(int64(currentRound), 10)
					err = tKeyGen.packAndSend(resp.MultisigInfo, int(currentRound), localPartyID, currentMsgType)
					if err != nil {
						globalErr = err
						return
					}
					atomic.AddInt32(&exchangeRound, 1)

				default:
					receivedMsgType := share.MsgType
					checkStr := strings.Split(receivedMsgType, "@")
					if len(checkStr) != 2 || checkStr[0] != common.MoneroKeyGenShareExchange {
						tKeyGen.logger.Error().Msg("not a valid monero share")
						globalErr = errors.New("not a valid share")
						return
					}
					_, err := strconv.ParseInt(checkStr[1], 10, 32)
					if err != nil {
						tKeyGen.logger.Error().Msg("not a valid monero share")
						globalErr = errors.New("not a valid share")
						return
					}

					currentRound := atomic.LoadInt32(&exchangeRound)
					shares, ready := shareStore.StoreAndCheck(int(currentRound)-1, share, peerNum)
					if !ready {
						continue
					}
					dat := make([]string, len(shares))
					for i, el := range shares {
						dat[i] = el.MultisigInfo
					}

					finRequest := moneroWallet.RequestExchangeMultisigKeys{
						MultisigInfo: dat,
						Password:     passcode,
					}
					resp, err := client.ExchangeMultiSigKeys(&finRequest)
					if err != nil {
						globalErr = err
						return
					}
					// this indicate the wallet address is generated
					if len(resp.Address) != 0 {
						address = resp.Address
						err = tKeyGen.moneroCommonStruct.NotifyTaskDone()
						if err != nil {
							tKeyGen.logger.Error().Err(err).Msg("fail to broadcast the keysign done")
						}
						continue
					}

					currentMsgType := common.MoneroKeyGenShareExchange + "@" + strconv.FormatInt(int64(currentRound), 10)
					err = tKeyGen.packAndSend(resp.MultisigInfo, int(currentRound), localPartyID, currentMsgType)
					if err != nil {
						globalErr = err
						return
					}
					atomic.AddInt32(&exchangeRound, 1)
				}
			case <-tKeyGen.moneroCommonStruct.GetTaskDone():
				close(tKeyGen.commStopChan)
				return
			}
		}
	}()

	keyGenWg.Wait()
	if globalErr != nil {
		tKeyGen.logger.Error().Msgf("fail to create the monero wallet with %s", tKeyGen.GetTssCommonStruct().GetConf().KeyGenTimeout)
		lastMsg := blameMgr.GetLastMsg()
		if lastMsg == "" {
			tKeyGen.logger.Error().Msg("fail to start the keygen, the last produced message of this node is none")
			return "", "", errors.New("timeout before shared message is generated")
		}
		blameNodesBroadcast, err := blameMgr.GetBroadcastBlame(lastMsg)
		if err != nil {
			tKeyGen.logger.Error().Err(err).Msg("error in get broadcast blame")
		}
		blameMgr.GetBlame().AddBlameNodes(blameNodesBroadcast...)
		return "", "", blame.ErrTssTimeOut
	}
	req := moneroWallet.RequestQueryKey{
		KeyType: "view_key",
	}
	resp, err := client.QueryKey(&req)
	if err != nil {
		tKeyGen.logger.Error().Err(err).Msgf("fail to query the key")
		return "", "", err
	}
	tKeyGen.logger.Info().Msgf("wallet address is  %v with private view key %v\n", address, resp.Key)
	return address, resp.Key, err
}
