package tss

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/haven-protocol-org/go-haven-rpc-client/wallet"
	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/akildemir/moneroTss/blame"
	"github.com/akildemir/moneroTss/common"
	"github.com/akildemir/moneroTss/conversion"
	"github.com/akildemir/moneroTss/messages"
	"github.com/akildemir/moneroTss/monero_multi_sig/keysign"
	"github.com/akildemir/moneroTss/p2p"
)

func (t *TssServer) waitForSignatures(msgID, encodedTx string, walletClient wallet.Client, sigChan chan string, threshold int) (keysign.Response, error) {
	data, err := t.signatureNotifier.WaitForSignature(msgID, encodedTx, walletClient, t.conf.KeySignTimeout, sigChan, threshold)
	if err != nil {
		return keysign.Response{}, err
	}
	if data == nil {
		return keysign.Response{}, errors.New("keysign failed with nil signature")
	}
	return keysign.NewResponse(
		data.TransactionID,
		data.TxKey,
		common.Success,
		blame.Blame{},
	), nil
}

func (t *TssServer) generateSignature(msgID string, req keysign.Request, threshold int, allParticipants []string, blameMgr *blame.Manager, keysignInstance *keysign.MoneroKeySign, sigChan chan string) (keysign.Response, error) {
	allPeersID, err := conversion.GetPeerIDsFromPubKeys(allParticipants)
	if err != nil {
		t.logger.Error().Msg("invalid block height or public key")
		return keysign.Response{
			Status: common.Fail,
			Blame:  blame.NewBlame(blame.InternalError, []blame.Node{}),
		}, nil
	}

	oldJoinParty, err := conversion.VersionLTCheck(req.Version, messages.NEWJOINPARTYVERSION)
	if err != nil {
		return keysign.Response{
			Status: common.Fail,
			Blame:  blame.NewBlame(blame.InternalError, []blame.Node{}),
		}, errors.New("fail to parse the version")
	}
	// we use the old join party
	if oldJoinParty {
		allParticipants = req.SignerPubKeys
		myPk, err := conversion.GetPubKeyFromPeerID(t.p2pCommunication.GetHost().ID().String())
		if err != nil {
			t.logger.Info().Msgf("fail to convert the p2p id(%s) to pubkey, turn to wait for signature", t.p2pCommunication.GetHost().ID().String())
			return keysign.Response{}, p2p.ErrNotActiveSigner
		}
		isSignMember := false
		for _, el := range allParticipants {
			if myPk == el {
				isSignMember = true
				break
			}
		}
		if !isSignMember {
			t.logger.Info().Msgf("we(%s) are not the active signer", t.p2pCommunication.GetHost().ID().String())
			return keysign.Response{}, p2p.ErrNotActiveSigner
		}

	}

	joinPartyStartTime := time.Now()
	onlinePeers, leader, errJoinParty := t.joinParty(msgID, req.Version, req.BlockHeight, allParticipants, threshold, sigChan)
	joinPartyTime := time.Since(joinPartyStartTime)
	if errJoinParty != nil {
		// we received the signature from waiting for signature
		if errors.Is(errJoinParty, p2p.ErrSignReceived) {
			return keysign.Response{}, errJoinParty
		}
		t.tssMetrics.KeysignJoinParty(joinPartyTime, false)
		// this indicate we are processing the leaderness join party
		if leader == "NONE" {
			if onlinePeers == nil {
				t.logger.Error().Err(errJoinParty).Msg("error before we start join party")
				t.broadcastKeysignFailure(msgID, allPeersID)
				return keysign.Response{
					Status: common.Fail,
					Blame:  blame.NewBlame(blame.InternalError, []blame.Node{}),
				}, nil
			}

			blameNodes, err := blameMgr.NodeSyncBlame(req.SignerPubKeys, onlinePeers)
			if err != nil {
				t.logger.Err(err).Msg("fail to get peers to blame")
			}
			t.broadcastKeysignFailure(msgID, allPeersID)
			// make sure we blame the leader as well
			t.logger.Error().Err(err).Msgf("fail to form keysign party with online:%v", onlinePeers)
			return keysign.Response{
				Status: common.Fail,
				Blame:  blameNodes,
			}, nil
		}

		var blameLeader blame.Blame
		leaderPubKey, err := conversion.GetPubKeyFromPeerID(leader)
		if err != nil {
			t.logger.Error().Err(errJoinParty).Msgf("fail to convert the peerID to public key %s", leader)
			blameLeader = blame.NewBlame(blame.TssSyncFail, []blame.Node{})
		} else {
			blameLeader = blame.NewBlame(blame.TssSyncFail, []blame.Node{{leaderPubKey, nil, nil}})
		}

		t.broadcastKeysignFailure(msgID, allPeersID)
		// make sure we blame the leader as well
		t.logger.Error().Err(errJoinParty).Msgf("messagesID(%s)fail to form keysign party with online:%v", msgID, onlinePeers)
		return keysign.Response{
			Status: common.Fail,
			Blame:  blameLeader,
		}, nil

	}
	t.tssMetrics.KeysignJoinParty(joinPartyTime, true)
	isKeySignMember := false
	for _, el := range onlinePeers {
		if el == t.p2pCommunication.GetHost().ID() {
			isKeySignMember = true
		}
	}
	if !isKeySignMember {
		// we are not the keysign member so we quit keysign and waiting for signature
		t.logger.Info().Msgf("we(%s) are not the active signer", t.p2pCommunication.GetHost().ID().String())
		return keysign.Response{}, p2p.ErrNotActiveSigner
	}
	parsedPeers := make([]string, len(onlinePeers))
	for i, el := range onlinePeers {
		parsedPeers[i] = el.String()
	}

	signers, err := conversion.GetPubKeysFromPeerIDs(parsedPeers)
	if err != nil {
		sigChan <- "signature generated"
		return keysign.Response{
			Status: common.Fail,
			Blame:  blame.Blame{},
		}, nil
	}

	signedTx, err := keysignInstance.SignMessage(req.EncodedTx, signers)
	// the statistic of keygen only care about Tss it self, even if the following http response aborts,
	// it still counted as a successful keygen as the Tss model runs successfully.
	// as only the last node submit the signature, others will return nil of the signedTx
	if err != nil {
		t.logger.Error().Err(err).Msg("err in keysign")
		sigChan <- "signature generated"
		t.broadcastKeysignFailure(msgID, allPeersID)
		blameNodes := *blameMgr.GetBlame()
		return keysign.Response{
			Status: common.Fail,
			Blame:  blameNodes,
		}, blame.ErrTssTimeOut
	}

	// this indicates we are not the last node who submit the transaction
	if signedTx == nil {
		return keysign.NewResponse(
			"",
			"",
			common.Fail,
			blame.Blame{},
		), errors.New("not the final signer")
	}

	sigChan <- "signature generated"
	// update signature notification
	if err := t.signatureNotifier.BroadcastSignature(msgID, signedTx, allPeersID); err != nil {
		return keysign.Response{}, fmt.Errorf("fail to broadcast signature:%w", err)
	}

	return keysign.NewResponse(
		signedTx.TransactionID,
		signedTx.TxKey,
		common.Success,
		blame.Blame{},
	), nil
}

func (t *TssServer) updateKeySignResult(result keysign.Response, timeSpent time.Duration) {
	if result.Status == common.Success {
		t.tssMetrics.UpdateKeySign(timeSpent, true)
		return
	}
	t.tssMetrics.UpdateKeySign(timeSpent, false)
}

func (t *TssServer) KeySign(req keysign.Request) (keysign.Response, error) {
	t.logger.Info().
		Str("signer pub keys", strings.Join(req.SignerPubKeys, ",")).
		Msg("received keysign request")
	emptyResp := keysign.Response{}
	msgID, err := t.requestToMsgId(req)
	if err != nil {
		return emptyResp, err
	}

	keysignInstance, walletClient, err := keysign.NewMoneroKeySign(t.p2pCommunication.GetLocalPeerID(),
		t.conf,
		t.p2pCommunication.BroadcastMsgChan,
		t.stopChan, msgID,
		t.privateKey, t.p2pCommunication, req.RpcAddress)
	if err != nil {
		t.logger.Error().Err(err).Msgf("fail to create the monero keysign instance")
		return keysign.Response{}, err
	}
	defer func() {
		err := walletClient.CloseWallet()
		if err != nil {
			t.logger.Error().Err(err).Msgf("fail to close the wallet")
		}
	}()

	keySignChannels := keysignInstance.GetTssKeySignChannels()
	t.p2pCommunication.SetSubscribe(messages.TSSKeySignMsg, msgID, keySignChannels)
	t.p2pCommunication.SetSubscribe(messages.TSSKeySignVerMsg, msgID, keySignChannels)
	t.p2pCommunication.SetSubscribe(messages.TSSControlMsg, msgID, keySignChannels)
	t.p2pCommunication.SetSubscribe(messages.TSSTaskDone, msgID, keySignChannels)

	defer func() {
		t.p2pCommunication.CancelSubscribe(messages.TSSKeySignMsg, msgID)
		t.p2pCommunication.CancelSubscribe(messages.TSSKeySignVerMsg, msgID)
		t.p2pCommunication.CancelSubscribe(messages.TSSControlMsg, msgID)
		t.p2pCommunication.CancelSubscribe(messages.TSSTaskDone, msgID)

		t.p2pCommunication.ReleaseStream(msgID)
		t.signatureNotifier.ReleaseStream(msgID)
		t.partyCoordinator.ReleaseStream(msgID)
	}()

	localStateItem, err := t.stateManager.GetLocalState(req.PoolPubKey)
	if err != nil {
		return emptyResp, fmt.Errorf("fail to get local keygen state: %w", err)
	}

	oldJoinParty, err := conversion.VersionLTCheck(req.Version, messages.NEWJOINPARTYVERSION)
	if err != nil {
		return keysign.Response{
			Status: common.Fail,
			Blame:  blame.NewBlame(blame.InternalError, []blame.Node{}),
		}, errors.New("fail to parse the version")
	}

	if len(req.SignerPubKeys) == 0 && oldJoinParty {
		return emptyResp, errors.New("empty signer pub keys")
	}

	walletInfo, err := walletClient.IsMultisig()
	if err != nil {
		t.logger.Error().Err(err).Msgf("fail to get the wallet info")
		return keysign.Response{
			Status: common.Fail,
			Blame:  blame.NewBlame(blame.InternalError, []blame.Node{}),
		}, errors.New("fail to get the wallet info")
	}
	// monero wallet threshold=ecdsa tss threshold+1
	threshold := int(walletInfo.Threshold) - 1

	if len(req.SignerPubKeys) <= threshold && oldJoinParty {
		t.logger.Error().Msgf("not enough signers, threshold=%d and signers=%d", threshold, len(req.SignerPubKeys))
		return emptyResp, errors.New("not enough signers")
	}

	blameMgr := keysignInstance.GetTssCommonStruct().GetBlameMgr()

	var receivedSig, generatedSig keysign.Response
	var errWait, errGen error
	sigChan := make(chan string, 2)
	wg := sync.WaitGroup{}
	wg.Add(2)
	keysignStartTime := time.Now()
	// we wait for signatures
	go func() {
		defer wg.Done()
		receivedSig, errWait = t.waitForSignatures(msgID, req.EncodedTx, walletClient, sigChan, threshold)
		// we received an valid signature indeed
		if errWait == nil {
			sigChan <- "signature received"
			t.logger.Info().Msgf("for message %s we get the signature from the peer", msgID)
			return
		}
		t.logger.Info().Msgf("we fail to get the valid signature with error %v", errWait)
	}()

	// we generate the signature ourselves
	go func() {
		defer wg.Done()
		generatedSig, errGen = t.generateSignature(msgID, req, threshold, localStateItem.ParticipantKeys, blameMgr, keysignInstance, sigChan)
	}()
	wg.Wait()
	close(sigChan)
	keysignTime := time.Since(keysignStartTime)
	// we received the generated verified signature, so we return
	if errWait == nil {
		t.updateKeySignResult(receivedSig, keysignTime)
		return receivedSig, nil
	}
	// for this round, we are not the active signer
	if errors.Is(errGen, p2p.ErrSignReceived) || errors.Is(errGen, p2p.ErrNotActiveSigner) {
		t.updateKeySignResult(receivedSig, keysignTime)
		return receivedSig, nil
	}
	// we get the signature from our tss keysign
	t.updateKeySignResult(generatedSig, keysignTime)
	return generatedSig, errGen
}

func (t *TssServer) broadcastKeysignFailure(messageID string, peers []peer.ID) {
	if err := t.signatureNotifier.BroadcastFailed(messageID, peers); err != nil {
		t.logger.Err(err).Msg("fail to broadcast keysign failure")
	}
}
