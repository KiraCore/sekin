package net

import (
	"fmt"

	"github.com/saiset-co/sai-interx-manager/p2p"
	"github.com/saiset-co/sai-interx-manager/p2p/proto"
	"github.com/saiset-co/sai-interx-manager/p2p/types"
)

type joinResponseHandler struct {
	responseCh chan types.JoinResponse
	errCh      chan error
}

func (h *joinResponseHandler) HandleMessage(msg p2p.Message, from p2p.Peer) error {
	if msg.Type() != string(proto.MessageTypeJoinResponse) {
		return nil
	}

	var joinResp types.JoinResponse
	if err := proto.UnmarshalPayload(msg.Payload(), &joinResp); err != nil {
		h.errCh <- fmt.Errorf("failed to unmarshal join response: %w", err)
		return err
	}

	h.responseCh <- joinResp
	return nil
}
