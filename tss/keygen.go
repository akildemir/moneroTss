package tss

import (
	"time"

	"github.com/akildemir/moneroTss/blame"
	"github.com/akildemir/moneroTss/common"
	"github.com/akildemir/moneroTss/conversion"
	"github.com/akildemir/moneroTss/messagesmn"
	"github.com/akildemir/moneroTss/monero_multi_sig/keygen"
)

func (t *TssServer) Keygen(req keygen.Request) (keygen.Response, error) {
	t.tssKeyGenLocker.Lock()
	defer t.tssKeyGenLocker.Unlock()
	status := common.Success
	msgID, err := t.requestToMsgId(req)
	if err != nil {
		return keygen.Response{}, err
	}

	keygenInstance := keygen.NewMoneroKeyGen(
		t.p2pCommunication.GetLocalPeerID(),
		t.conf,
		t.localNodePubKey,
		t.p2pCommunication.BroadcastMsgChan,
		t.stopChan,
		msgID,
		t.stateManager, t.privateKey, t.p2pCommunication)

	keygenMsgChannel := keygenInstance.GetTssKeyGenChannels()
	t.p2pCommunication.SetSubscribe(messagesmn.TSSKeyGenMsg, msgID, keygenMsgChannel)
	t.p2pCommunication.SetSubscribe(messagesmn.TSSKeyGenVerMsg, msgID, keygenMsgChannel)
	t.p2pCommunication.SetSubscribe(messagesmn.TSSControlMsg, msgID, keygenMsgChannel)
	t.p2pCommunication.SetSubscribe(messagesmn.TSSTaskDone, msgID, keygenMsgChannel)

	defer func() {
		t.p2pCommunication.CancelSubscribe(messagesmn.TSSKeyGenMsg, msgID)
		t.p2pCommunication.CancelSubscribe(messagesmn.TSSKeyGenVerMsg, msgID)
		t.p2pCommunication.CancelSubscribe(messagesmn.TSSControlMsg, msgID)
		t.p2pCommunication.CancelSubscribe(messagesmn.TSSTaskDone, msgID)

		t.p2pCommunication.ReleaseStream(msgID)
		t.partyCoordinator.ReleaseStream(msgID)
	}()
	sigChan := make(chan string)
	blameMgr := keygenInstance.GetTssCommonStruct().GetBlameMgr()
	joinPartyStartTime := time.Now()
	onlinePeers, leader, errJoinParty := t.joinParty(msgID, req.Version, req.BlockHeight, req.Keys, len(req.Keys)-1, sigChan)
	joinPartyTime := time.Since(joinPartyStartTime)
	if errJoinParty != nil {
		t.tssMetrics.KeygenJoinParty(joinPartyTime, false)
		t.tssMetrics.UpdateKeyGen(0, false)
		// this indicate we are processing the leaderless join party
		if leader == "NONE" {
			if onlinePeers == nil {
				t.logger.Error().Err(err).Msg("error before we start join party")
				return keygen.Response{
					Status: common.Fail,
					Blame:  blame.NewBlame(blame.InternalError, []blame.Node{}),
				}, nil
			}
			blameNodes, err := blameMgr.NodeSyncBlame(req.Keys, onlinePeers)
			if err != nil {
				t.logger.Err(errJoinParty).Msg("fail to get peers to blame")
			}
			// make sure we blame the leader as well
			t.logger.Error().Err(errJoinParty).Msgf("fail to form keygen party with online:%v", onlinePeers)
			return keygen.Response{
				Status: common.Fail,
				Blame:  blameNodes,
			}, nil

		}

		var blameLeader blame.Blame
		var blameNodes blame.Blame
		blameNodes, err = blameMgr.NodeSyncBlame(req.Keys, onlinePeers)
		if err != nil {
			t.logger.Err(errJoinParty).Msg("fail to get peers to blame")
		}
		leaderPubKey, err := conversion.GetPubKeyFromPeerID(leader)
		if err != nil {
			t.logger.Error().Err(errJoinParty).Msgf("fail to convert the peerID to public key with leader %s", leader)
			blameLeader = blame.NewBlame(blame.TssSyncFail, []blame.Node{})
		} else {
			blameLeader = blame.NewBlame(blame.TssSyncFail, []blame.Node{{leaderPubKey, nil, nil}})
		}
		if len(onlinePeers) != 0 {
			blameNodes.AddBlameNodes(blameLeader.BlameNodes...)
		} else {
			blameNodes = blameLeader
		}
		t.logger.Error().Err(errJoinParty).Msgf("fail to form keygen party with online:%v", onlinePeers)

		return keygen.Response{
			Status: common.Fail,
			Blame:  blameNodes,
		}, nil

	}

	t.tssMetrics.KeygenJoinParty(joinPartyTime, true)
	t.logger.Debug().Msg("keygen party formed")
	// the statistic of keygen only care about Tss it self, even if the
	// following http response aborts, it still counted as a successful keygen
	// as the Tss model runs successfully.
	beforeKeygen := time.Now()
	address, viewKey, err := keygenInstance.GenerateNewKey(req)
	keygenTime := time.Since(beforeKeygen)
	if err != nil {
		t.tssMetrics.UpdateKeyGen(keygenTime, false)
		t.logger.Error().Err(err).Msg("err in keygen")
		blameNodes := *blameMgr.GetBlame()
		return keygen.NewResponse("", "", common.Fail, blameNodes), err
	} else {
		t.tssMetrics.UpdateKeyGen(keygenTime, true)
	}

	blameNodes := *blameMgr.GetBlame()
	return keygen.NewResponse(
		address,
		viewKey,
		status,
		blameNodes,
	), nil
}
