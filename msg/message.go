// Copyright 2020 ChainSafe Systems
// SPDX-License-Identifier: LGPL-3.0-only

package msg

import (
	"fmt"
	"math/big"
)

type ChainId uint8
type TransferType string
type ResourceId [32]byte

func (r ResourceId) Hex() string {
	return fmt.Sprintf("%x", r)
}

type Nonce uint64

func (n Nonce) Big() *big.Int {
	return big.NewInt(int64(n))
}

var FungibleTransfer TransferType = "FungibleTransfer"
var NonFungibleTransfer TransferType = "NonFungibleTransfer"
var GenericTransfer TransferType = "GenericTransfer"
var AckTransfer TransferType = "AckTransfer"

// Message is used as a generic format to communicate between chains
type Message struct {
	Source      ChainId      // Source where message was initiated
	Destination ChainId      // Destination chain of message
	Type        TransferType // type of bridge transfer
	Nonce       Nonce        // Nonce for the message transfer
	ResourceId  ResourceId
	Payload     []interface{} // data associated with event sequence
}

func NewFungibleTransfer(source, dest ChainId, nonce Nonce, amount *big.Int, resourceId ResourceId, recipient []byte) Message {
	return Message{
		Source:      source,
		Destination: dest,
		Type:        FungibleTransfer,
		Nonce:       nonce,
		ResourceId:  resourceId,
		Payload: []interface{}{
			amount.Bytes(),
			recipient,
		},
	}
}

func NewNonFungibleTransfer(source, dest ChainId, nonce Nonce, resourceId ResourceId, tokenId *big.Int, recipient, metadata []byte) Message {
	return Message{
		Source:      source,
		Destination: dest,
		Type:        NonFungibleTransfer,
		Nonce:       nonce,
		ResourceId:  resourceId,
		Payload: []interface{}{
			tokenId.Bytes(),
			recipient,
			metadata,
		},
	}
}

func NewGenericTransfer(source, dest ChainId, nonce Nonce, resourceId ResourceId, metadata []byte) Message {
	return Message{
		Source:      source,
		Destination: dest,
		Type:        GenericTransfer,
		Nonce:       nonce,
		ResourceId:  resourceId,
		Payload: []interface{}{
			metadata,
		},
	}
}

func NewAckTransfer(source, dest ChainId, nonce Nonce, resourceId ResourceId, ackData []byte) Message {
	return Message{
		Source:      source,
		Destination: dest,
		Type:        AckTransfer,
		Nonce:       nonce,
		ResourceId:  resourceId,
		Payload: []interface{}{
			ackData,
		},
	}
}

func ResourceIdFromSlice(in []byte) ResourceId {
	var res ResourceId
	copy(res[:], in)
	return res
}
