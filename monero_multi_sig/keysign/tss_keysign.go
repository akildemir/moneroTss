package keysign

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"

	btss "github.com/binance-chain/tss-lib/tss"
	coskey "github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	set "github.com/deckarep/golang-set"
	moneroWallet "github.com/haven-protocol-org/go-haven-rpc-client/wallet"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	tcrypto "github.com/tendermint/tendermint/crypto"

	"github.com/akildemir/moneroTss/blame"
	"github.com/akildemir/moneroTss/common"
	"github.com/akildemir/moneroTss/conversion"
	"github.com/akildemir/moneroTss/messages"
	"github.com/akildemir/moneroTss/monero_multi_sig"
	"github.com/akildemir/moneroTss/p2p"
	"github.com/akildemir/moneroTss/storage"
)

type MoneroKeySign struct {
	logger             zerolog.Logger
	moneroCommonStruct *common.TssCommon
	localNodePubKey    string
	stopChan           chan struct{} // channel to indicate whether we should stop
	localParty         *btss.PartyID
	commStopChan       chan struct{}
	p2pComm            *p2p.Communication
	stateManager       storage.LocalStateManager
	walletClient       moneroWallet.Client
}

type MoneroSpendProof struct {
	TransactionID string
	TxKey         string
}

func NewMoneroKeySign(localP2PID string,
	conf common.TssConfig,
	broadcastChan chan *messages.BroadcastMsgChan,
	stopChan chan struct{}, msgID string, privKey tcrypto.PrivKey, p2pComm *p2p.Communication, rpcAddress string) (*MoneroKeySign, moneroWallet.Client, error) {
	logItems := []string{"keySign", msgID}

	pk := coskey.PubKey{
		Key: privKey.PubKey().Bytes(),
	}
	pubKey, _ := sdk.Bech32ifyPubKey(sdk.Bech32PubKeyTypeAccPub, &pk)

	rpcWalletConfig := moneroWallet.Config{
		Address: rpcAddress,
	}
	moneroSignClient := MoneroKeySign{
		logger:             log.With().Strs("module", logItems).Logger(),
		localNodePubKey:    pubKey,
		moneroCommonStruct: common.NewTssCommon(localP2PID, broadcastChan, conf, msgID, privKey),
		stopChan:           stopChan,
		localParty:         nil,
		commStopChan:       make(chan struct{}),
		p2pComm:            p2pComm,
		walletClient:       moneroWallet.New(rpcWalletConfig),
	}

	walletName := pubKey + ".mo"
	passcode := moneroSignClient.GetTssCommonStruct().GetNodePrivKey()
	// now open the wallet
	walletOpenReq := moneroWallet.RequestOpenWallet{
		Filename: walletName,
		Password: passcode,
	}

	err := moneroSignClient.walletClient.OpenWallet(&walletOpenReq)
	if err != nil {
		moneroSignClient.logger.Error().Err(err).Msgf("fail to open the wallet")
		return nil, nil, err
	}

	return &moneroSignClient, moneroSignClient.walletClient, nil
}

func (tKeySign *MoneroKeySign) GetTssKeySignChannels() chan *p2p.Message {
	return tKeySign.moneroCommonStruct.TssMsg
}

func (tKeySign *MoneroKeySign) GetTssCommonStruct() *common.TssCommon {
	return tKeySign.moneroCommonStruct
}

func (tKeySign *MoneroKeySign) amIFirstNode(msgID string, parties []string) ([]string, int) {
	keyStore := make(map[string]string)
	hashes := make([]string, len(parties))
	for i, el := range parties {
		sum := sha256.Sum256([]byte(msgID + el))
		encodedSum := hex.EncodeToString(sum[:])
		keyStore[encodedSum] = el
		hashes[i] = encodedSum
	}
	sort.Strings(hashes)

	var sortedOrder []string
	myIndex := 0
	myIndexFound := false
	for i := 0; i < len(keyStore); i++ {
		if tKeySign.localNodePubKey == keyStore[hashes[i]] {
			myIndexFound = true
		}
		sortedOrder = append(sortedOrder, keyStore[hashes[i]])
		if !myIndexFound {
			myIndex += 1
		}
	}
	return sortedOrder, myIndex
}

func (tKeySign *MoneroKeySign) packAndSend(info string, exchangeRound int, localPartyID, toParty *btss.PartyID, msgType string) error {
	sendShare := common.MoneroShare{
		MultisigInfo:  info,
		MsgType:       msgType,
		ExchangeRound: exchangeRound,
	}
	msg, err := json.Marshal(sendShare)
	if err != nil {
		tKeySign.logger.Error().Err(err).Msg("fail to encode the wallet share")
		return err
	}
	roundInfo := msgType + strconv.FormatInt(int64(exchangeRound), 10)
	tKeySign.moneroCommonStruct.GetBlameMgr().SetLastMsg(roundInfo)
	if toParty == nil {
		r := btss.MessageRouting{
			From:        localPartyID,
			IsBroadcast: true,
		}
		return tKeySign.moneroCommonStruct.ProcessOutCh(msg, &r, roundInfo, messages.TSSKeySignMsg)
	}
	r := btss.MessageRouting{
		From:        localPartyID,
		To:          []*btss.PartyID{toParty},
		IsBroadcast: false,
	}
	return tKeySign.moneroCommonStruct.ProcessOutCh(msg, &r, roundInfo, messages.TSSKeySignMsg)
}

func (tKeySign *MoneroKeySign) submitSignature(signature string) ([]string, error) {
	client2Submit := moneroWallet.RequestSubmitMultisig{
		TxDataHex: signature,
	}
	signedTxHash, err := tKeySign.walletClient.SubmitMultisig(&client2Submit)
	return signedTxHash.TxHashList, err
}

func (tKeySign *MoneroKeySign) processPrepareMsg(shares []*common.MoneroShare) (map[string]monero_multi_sig.MoneroPrepareMsg, error) {
	// we store the peer's public keys
	peerPrepareMSg := make(map[string]monero_multi_sig.MoneroPrepareMsg)
	for _, el := range shares {
		dat, err := monero_multi_sig.DecodePrePareInfo(el.MultisigInfo)
		if err != nil {
			return nil, err
		}
		peerPrepareMSg[el.Sender] = dat
	}

	return peerPrepareMSg, nil
}

func (tKeySign *MoneroKeySign) calcualtePubkeyForUse(pubkeysAll map[string][]string, keys *moneroWallet.ResponseExportSigPubkey, orderedNodes []string) []string {
	var nodesBeforeMe []string
	for _, el := range orderedNodes {
		if el == tKeySign.localNodePubKey {
			break
		}
		nodesBeforeMe = append(nodesBeforeMe, el)
	}

	// now we figure out the public key that I will use.
	peerPubKeys := set.NewSet()
	for _, el := range nodesBeforeMe {
		pubKeys := pubkeysAll[el]
		for _, pk := range pubKeys {
			peerPubKeys.Add(pk)
		}
	}

	myPubKey := set.NewSet()
	for _, el := range keys.PubKeys {
		myPubKey.Add(el)
	}

	selectedKeys := myPubKey.Difference(peerPubKeys)
	var rselectedKeys []string
	for _, el := range selectedKeys.ToSlice() {
		rselectedKeys = append(rselectedKeys, el.(string))
	}

	return rselectedKeys
}

func (tKeySign *MoneroKeySign) submitAndGetConfirm(txForSubmit string, signedTx *MoneroSpendProof) error {
	// now we submit
	submitData := moneroWallet.RequestSubmitMultisig{
		TxDataHex: txForSubmit,
	}

	txID, err := tKeySign.walletClient.SubmitMultisig(&submitData)
	if err != nil {
		tKeySign.logger.Error().Err(err).Msgf("fail to submit the signature")
		return err
	}

	signedTx.TransactionID = txID.TxHashList[0]
	// currently, we only hanle one tx a keysign one request
	sendProof := moneroWallet.RequestGetTxKey{
		TxID: signedTx.TransactionID,
	}

	counter := 0
	var proofResp *moneroWallet.ResponseGetTxKey
	for ; counter < 10; counter++ {
		proofResp, err = tKeySign.getTxFromTxKey(&sendProof)
		if err != nil {
			time.Sleep(time.Second)
			continue
		}
		break
	}
	if counter >= 10 {
		tKeySign.logger.Error().Err(err).Msgf("fail to get the tx send proof")
		return errors.New("fail to get the tx key")
	}

	signedTx.TxKey = proofResp.TxKey
	return nil
}

// signMessage
func (tKeySign *MoneroKeySign) SignMessage(encodedTx string, parties []string) (*MoneroSpendProof, error) {
	var globalErr error
	partiesID, localPartyID, err := conversion.GetParties(parties, tKeySign.localNodePubKey)
	tKeySign.localParty = localPartyID
	if err != nil {
		return nil, fmt.Errorf("fail to form key sign party: %w", err)
	}

	if !common.Contains(partiesID, localPartyID) {
		tKeySign.logger.Info().Msgf("we are not in this rounds key sign")
		return nil, nil
	}

	tKeySign.logger.Debug().Msgf("local party: %+v", localPartyID)

	blameMgr := tKeySign.moneroCommonStruct.GetBlameMgr()

	partyIDMap := conversion.SetupPartyIDMap(partiesID)
	err1 := conversion.SetupIDMaps(partyIDMap, tKeySign.moneroCommonStruct.PartyIDtoP2PID)
	err2 := conversion.SetupIDMaps(partyIDMap, blameMgr.PartyIDtoP2PID)
	if err1 != nil || err2 != nil {
		tKeySign.logger.Error().Err(err).Msgf("error in creating mapping between partyID and P2P ID")
		return nil, err
	}

	tKeySign.moneroCommonStruct.SetPartyInfo(&common.PartyInfo{
		Party:      localPartyID,
		PartyIDMap: partyIDMap,
	})

	blameMgr.SetPartyInfo(localPartyID, partyIDMap)
	tKeySign.moneroCommonStruct.P2PPeers = conversion.GetPeersID(tKeySign.moneroCommonStruct.PartyIDtoP2PID, tKeySign.moneroCommonStruct.GetLocalPeerID())
	var keySignWg sync.WaitGroup

	walletInfo, err := tKeySign.walletClient.IsMultisig()
	if err != nil {
		tKeySign.logger.Error().Err(err).Msg("fail to query the wallet info")
		return nil, err
	}
	if !walletInfo.Multisig || !walletInfo.Ready {
		tKeySign.logger.Error().Err(err).Msg("it is not a multisig wallet or wallet is not ready")
		return nil, errors.New("not a multisig wallet or wallet is not ready(keygen done correctly?)")
	}

	tx, err := base64.StdEncoding.DecodeString(encodedTx)
	if err != nil {
		tKeySign.logger.Error().Err(err).Msg("fail to decode the transaction")
		return nil, err
	}

	var txSend moneroWallet.RequestTransfer
	err = json.Unmarshal(tx, &txSend)
	if err != nil {
		tKeySign.logger.Error().Err(err).Msg("fail to unmarshal the transaction")
		return nil, err
	}

	balanceReq := moneroWallet.RequestGetBalance{
		AccountIndex: 0,
		AssetType:    txSend.AssetType,
	}
	// we check whether we have enough fund to transfer
	var totalAmount uint64
	for _, el := range txSend.Destinations {
		totalAmount += el.Amount
	}
	counter := 0
	// because the monero wallet has high possibility to report incorrect balance when it is just opened,
	// we need to see 3 confirmations of the balance
	totalConfirmed := 0
	for ; counter < monero_multi_sig.MoneroWalletRetry; counter++ {
		time.Sleep(time.Second * 2)
		balance, err := tKeySign.walletClient.GetBalance(&balanceReq)
		if err != nil {
			tKeySign.logger.Error().Err(err).Msg("fail to get the balance of the wallet")
			return nil, err
		}
		height, err := tKeySign.walletClient.GetHeight()
		if err != nil {
			tKeySign.logger.Error().Err(err).Msg("fail to get the height of the wallet block")
			return nil, err
		}

		// it fail still lack of fund as the fee is not added here
		if balance.UnlockedBalance > totalAmount {
			tKeySign.logger.Info().Msgf("unlock balance is %v with height %d\n", balance.UnlockedBalance, height.Height)
			totalConfirmed += 1
			if totalConfirmed >= 3 {
				break
			}
		}
		tKeySign.logger.Warn().Msgf("fail to get the unlock balance, the wallet end may be slow")
	}
	if counter >= 10 && totalConfirmed == 0 {
		return nil, errors.New("not enough fund in wallet")
	}

	threshold := walletInfo.Threshold
	needToWait := threshold - 1 // we do not need to wait for ourselves

	// import message
	orderedNodes, _ := tKeySign.amIFirstNode(tKeySign.GetTssCommonStruct().GetMsgID(), parties)
	leader := orderedNodes[0]

	isLeader := leader == tKeySign.localNodePubKey
	var responseTransfer *moneroWallet.ResponseTransfer
	moneroShareChan := make(chan *common.MoneroShare, len(partiesID))

	keySignWg.Add(1)
	go func() {
		tKeySign.moneroCommonStruct.ProcessInboundmessages(tKeySign.commStopChan, &keySignWg, moneroShareChan)
	}()

	// we exchange the keysign preparation info
	exportedMultisigInfo, err := tKeySign.walletClient.ExportMultisigInfo()
	if err != nil {
		return nil, err
	}

	exportedPubKeys, err := tKeySign.walletClient.ExportSigPubKey()
	if err != nil {
		tKeySign.logger.Error().Err(err).Msgf("fail to export the siging public key")
		return nil, err
	}

	encodedMsg, err := monero_multi_sig.EncodePrePareInfo(exportedMultisigInfo.Info, exportedPubKeys.PubKeys)
	if err != nil {
		return nil, err
	}

	err = tKeySign.packAndSend(encodedMsg, 0, localPartyID, nil, common.MoneroExportedSignMsg)
	if err != nil {
		return nil, err
	}

	shareStore := monero_multi_sig.GenMoneroShareStore()
	tssConf := tKeySign.moneroCommonStruct.GetConf()
	var myShare, leaderShare string
	var signedTx MoneroSpendProof
	peerSigningPubkeys := make(map[string][]string)
	keySignWg.Add(1)
	go func() {
		defer func() {
			keySignWg.Done()
			close(tKeySign.commStopChan)
		}()
		for {
			select {
			case <-time.After(tssConf.KeySignTimeout):
				tKeySign.logger.Error().Msgf("fail to generate the signature with %v", tssConf.KeySignTimeout)
				globalErr = blame.ErrTssTimeOut
				return

			case share := <-moneroShareChan:
				switch share.MsgType {
				case common.MoneroExportedSignMsg:
					shares, ready := shareStore.StoreAndCheck(0, share, int(needToWait))
					if !ready {
						continue
					}

					peerPreparedMsg, err := tKeySign.processPrepareMsg(shares)
					if err != nil {
						globalErr = err
						tKeySign.logger.Error().Err(err).Msg("fail to process the prepare message")
						return
					}
					var multiSigInfo []string
					for sender, dat := range peerPreparedMsg {
						multiSigInfo = append(multiSigInfo, dat.ExchangeInfo)
						peerSigningPubkeys[sender] = dat.Pubkeys
					}
					info := moneroWallet.RequestImportMultisigInfo{
						Info: multiSigInfo,
					}
					_, err = tKeySign.walletClient.ImportMultisigInfo(&info)
					if err != nil {
						tKeySign.logger.Error().Err(err).Msg("fail to import the multisig info")
						globalErr = err
						err = tKeySign.moneroCommonStruct.NotifyTaskDone()
						if err != nil {
							tKeySign.logger.Error().Err(err).Msg("fail to broadcast the keysign done")
							globalErr = err
						}
						return
					}

					// if we are the leader, we need to initialise the wallet.
					if isLeader {
						responseTransfer, err = tKeySign.walletClient.Transfer(&txSend)
						if err != nil {
							tKeySign.logger.Error().Err(err).Msgf("we(%s) fail to create the transfer data ", tKeySign.localNodePubKey)
							globalErr = err // we will handle the error in the upper level
							err = tKeySign.moneroCommonStruct.NotifyTaskDone()
							if err != nil {
								tKeySign.logger.Error().Err(err).Msg("fail to broadcast the keysign done")
								globalErr = err
							}
							return
						}
						leaderShare = responseTransfer.MultisigTxSet
						myShare = leaderShare
						err = tKeySign.packAndSend(responseTransfer.MultisigTxSet, 1, localPartyID, nil, common.MoneroInitTransfer)
						if err != nil {
							// fixme notify the failure of keysign
							tKeySign.logger.Error().Err(err).Msg("fail to send the initialization transfer info")
							globalErr = err
							err = tKeySign.moneroCommonStruct.NotifyTaskDone()
							if err != nil {
								tKeySign.logger.Error().Err(err).Msg("fail to broadcast the keysign done")
								globalErr = err
							}
							return
						}
						tKeySign.logger.Info().Msg("leader have done the signature preparation")
					}
					// fixme what other nodes should do?
					tKeySign.logger.Info().Msgf("we(%s) have done the signature preparation", tKeySign.localNodePubKey)

				case common.MoneroInitTransfer:
					if isLeader {
						// the leader does not need to be involved in this round
						continue
					}
					if share.Sender != leader {
						continue
					}
					// now we need to figure out what public keys that we apply to sign the transactions.
					pks := tKeySign.calcualtePubkeyForUse(peerSigningPubkeys, exportedPubKeys, orderedNodes)

					leaderShare = share.MultisigInfo
					checkResult, err := tKeySign.verifyTransaction(leaderShare, txSend.Destinations)
					if err != nil || !checkResult {
						tKeySign.logger.Error().Msg("fail to verify the transaction")
						globalErr = errors.New("transaction cannot been verified")
						blameLeader := blame.NewNode(leader, nil, nil)
						blameMgr.GetBlame().AddBlameNodes(blameLeader)
						return
					}
					outData := moneroWallet.RequestSignMultisigParallel{
						TxDataHex: leaderShare,
						PubKeys:   pks,
					}
					ret, err := tKeySign.walletClient.SignMultisigParallel(&outData)
					if err != nil {
						globalErr = err
						err = tKeySign.moneroCommonStruct.NotifyTaskDone()
						if err != nil {
							tKeySign.logger.Error().Err(err).Msg("fail to broadcast the keysign done")
							globalErr = err
						}
						tKeySign.logger.Error().Err(err).Msg("fail to sign the transaction")
						return
					}

					myShare = ret.TxDataHex
					err = tKeySign.packAndSend(myShare, 1, localPartyID, nil, common.MoneroSignShares)
					if err != nil {
						tKeySign.logger.Error().Err(err).Msgf("fail to send the message")
						globalErr = err
						err = tKeySign.moneroCommonStruct.NotifyTaskDone()
						if err != nil {
							tKeySign.logger.Error().Err(err).Msg("fail to broadcast the keysign done")
							globalErr = err
						}
						return
					}
				case common.MoneroSignShares:
					var ready bool
					var shares []*common.MoneroShare
					if isLeader {
						shares, ready = shareStore.StoreAndCheck(1, share, int(needToWait))
					} else {
						shares, ready = shareStore.StoreAndCheck(1, share, int(needToWait-1))
					}
					if !ready {
						continue
					}
					var accuData []string
					var blameNodes []blame.Node
					accuData = append(accuData, leaderShare)
					for _, el := range shares {
						checkResult, err := tKeySign.verifyTransaction(el.MultisigInfo, txSend.Destinations)
						if err != nil || !checkResult {
							blameNode := blame.NewNode(el.Sender, nil, nil)
							blameNodes = append(blameNodes, blameNode)
						}
						accuData = append(accuData, el.MultisigInfo)
					}

					if len(blameNodes) != 0 {
						tKeySign.logger.Error().Msg("fail to verify the transaction")
						globalErr = errors.New("transaction cannot been verified")
						blameMgr.GetBlame().AddBlameNodes(blameNodes...)
						globalErr = errors.New("transaction verification failed")
						return
					}

					accuData = append(accuData, myShare)
					accReq := moneroWallet.RequestAccuMultisig{
						TxDataHex: accuData,
					}

					ret, err := tKeySign.walletClient.AccuMultisig(&accReq)
					if err != nil {
						tKeySign.logger.Error().Err(err).Msgf("fail to accumulate the signatures")
						globalErr = err
						err = tKeySign.moneroCommonStruct.NotifyTaskDone()
						if err != nil {
							tKeySign.logger.Error().Err(err).Msg("fail to broadcast the keysign done")
							globalErr = err
						}
						return
					}

					globalErr = tKeySign.submitAndGetConfirm(ret.TxDataHex, &signedTx)
					if globalErr != nil {
						tKeySign.logger.Error().Err(err).Msgf("fail to submit the transaction")
						err = tKeySign.moneroCommonStruct.NotifyTaskDone()
						if err != nil {
							tKeySign.logger.Error().Err(err).Msg("fail to broadcast the keysign done")
							globalErr = err
						}
						return
					}

					err = tKeySign.moneroCommonStruct.NotifyTaskDone()
					if err != nil {
						tKeySign.logger.Error().Err(err).Msg("fail to broadcast the keysign done")
						globalErr = err
					}
					tKeySign.logger.Info().Msgf("transaction %s has been submitted successfully with tx key %s", signedTx.TransactionID, signedTx.TxKey)
				}

			case <-tKeySign.moneroCommonStruct.GetTaskDone():
				return
			}
		}
	}()

	keySignWg.Wait()
	if globalErr != nil {
		tKeySign.logger.Error().Msgf("fail to create the monero signature with %s", tKeySign.GetTssCommonStruct().GetConf().KeyGenTimeout)
		lastMsg := blameMgr.GetLastMsg()
		if lastMsg == "" {
			tKeySign.logger.Error().Msg("fail to start the keygen, the last produced message of this node is none")
			return nil, errors.New("timeout before shared message is generated")
		}
		blameNodesBroadcast, err := blameMgr.GetBroadcastBlame(lastMsg)
		if err != nil {
			tKeySign.logger.Error().Err(err).Msg("error in get broadcast blame")
		}
		blameMgr.GetBlame().AddBlameNodes(blameNodesBroadcast...)
		return nil, globalErr
	}

	tKeySign.logger.Info().Msgf("%s successfully sign the message with TXID: %s and key: %s", tKeySign.p2pComm.GetHost().ID().String(), signedTx.TransactionID, signedTx.TxKey)
	return &signedTx, nil
}

func (tKeySign *MoneroKeySign) getTxFromTxKey(sendProof *moneroWallet.RequestGetTxKey) (*moneroWallet.ResponseGetTxKey, error) {
	var proofResp *moneroWallet.ResponseGetTxKey
	var err error
	proofResp, err = tKeySign.walletClient.GetTxKey(sendProof)
	if err != nil || proofResp == nil {
		tKeySign.logger.Error().Err(err).Msgf("fail to get the proof of the spend transaction")
		return nil, err
	}
	return proofResp, nil
}

func (tKeySign *MoneroKeySign) verifyTransaction(receivedShare string, myDest []*moneroWallet.Destination) (bool, error) {
	transactionCheck := moneroWallet.RequestCheckTransaction{
		Destinations: myDest,
		TxDataHex:    receivedShare,
	}
	ret, err := tKeySign.walletClient.CheckTransaction(&transactionCheck)
	if err != nil {
		return false, err
	}
	return ret.CheckResult, nil
}
