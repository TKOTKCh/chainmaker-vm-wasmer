/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package protocol

import (
	"chainmaker.org/chainmaker/pb-go/v2/net"
)

// NetType is the type of net.
type NetType int

const (
	// Libp2p is a type of p2p net.
	Libp2p NetType = iota
	// Liquid is a type of net that developed by ourselves.
	Liquid
)

// DirectMsgHandler handle the msg received from other node.
type DirectMsgHandler func(from string, msgData []byte) error

// PubSubMsgHandler handle the msg published by other node.
type PubSubMsgHandler func(publisher string, msgData []byte) error

// ChainNodeInfo 网络节点信息
type ChainNodeInfo struct {
	NodeUid     string
	NodeAddress []string
	NodeTlsCert []byte
}

//MsgHandler P2P网络消息处理Handler
type MsgHandler func(from string, msg []byte, msgType net.NetMsg_MsgType) error

// Net is local net interface.
type Net interface {
	// GetNodeUid is the unique id of the node.
	GetNodeUid() string
	// InitPubSub will init new PubSub instance with given chainId and maxMessageSize.
	InitPubSub(chainId string, maxMessageSize int) error
	// BroadcastWithChainId  will broadcast a msg to a PubSubTopic with the pub-sub service which id is given chainId.
	BroadcastWithChainId(chainId string, topic string, netMsg []byte) error
	// SubscribeWithChainId register a PubSubMsgHandler to a PubSubTopic
	// with the pub-sub service which id is given chainId.
	SubscribeWithChainId(chainId string, topic string, handler PubSubMsgHandler) error
	// CancelSubscribeWithChainId cancel subscribe a PubSubTopic with the pub-sub service which id is given chainId.
	CancelSubscribeWithChainId(chainId string, topic string) error
	// SendMsg send msg to the node which id is given string.
	// 		msgFlag: is a flag used to distinguish msg type.
	SendMsg(chainId string, node string, msgFlag string, netMsg []byte) error
	// DirectMsgHandle register a DirectMsgHandler to the net.
	// 		msgFlag: is a flag used to distinguish msg type.
	DirectMsgHandle(chainId string, msgFlag string, handler DirectMsgHandler) error
	// CancelDirectMsgHandle unregister a DirectMsgHandler.
	// 		msgFlag: is a flag used to distinguish msg type.
	CancelDirectMsgHandle(chainId string, msgFlag string) error
	// AddSeed add a seed node addr.
	AddSeed(seed string) error
	// RefreshSeeds refresh the seed node addr list.
	RefreshSeeds(seeds []string) error
	// SetChainCustomTrustRoots set custom trust roots of chain.
	// In cert permission mode, if it is failed when verifying cert by access control of chains,
	// the cert will be verified by custom trust root pool again.
	SetChainCustomTrustRoots(chainId string, roots [][]byte)
	// ReVerifyPeers will verify permission of peers existed with the access control module of the chain
	// which id is the given chainId.
	ReVerifyPeers(chainId string)
	// IsRunning return true when the net instance is running.
	IsRunning() bool
	// Start the local net.
	Start() error
	// Stop the local net.
	Stop() error
	// ChainNodesInfo return base node info list of chain which id is the given chainId.
	ChainNodesInfo(chainId string) ([]*ChainNodeInfo, error)
	// GetNodeUidByCertId return node uid which mapped to the given cert id. If unmapped return error.
	GetNodeUidByCertId(certId string) (string, error)
	// AddAC add a AccessControlProvider for revoked validator.
	AddAC(chainId string, ac AccessControlProvider)
	// SetMsgPriority set the priority of the msg flag.
	// If priority control disabled, it is no-op.
	SetMsgPriority(msgFlag string, priority uint8)
}

// ChainNodesInfoProvider provide base node info list of chain.
type ChainNodesInfoProvider interface {
	// GetChainNodesInfo return base node info list of chain.
	GetChainNodesInfo() ([]*ChainNodeInfo, error)
}

// NetService P2P网络模块接口
type NetService interface {
	// BroadcastMsg broadcast a msg to the net.
	BroadcastMsg(msg []byte, msgType net.NetMsg_MsgType) error
	// Subscribe register a MsgHandler for subscribe.
	Subscribe(msgType net.NetMsg_MsgType, handler MsgHandler) error
	// CancelSubscribe cancel subscribe.
	CancelSubscribe(msgType net.NetMsg_MsgType) error
	// ConsensusBroadcastMsg broadcast a msg to the consensus nodes.
	ConsensusBroadcastMsg(msg []byte, msgType net.NetMsg_MsgType) error
	// ConsensusSubscribe register a MsgHandler handle the msg from consensus nodes for subscribe.
	ConsensusSubscribe(msgType net.NetMsg_MsgType, handler MsgHandler) error
	// CancelConsensusSubscribe cancel subscribe.
	CancelConsensusSubscribe(msgType net.NetMsg_MsgType) error
	// SendMsg send msg to any nodes.
	SendMsg(msg []byte, msgType net.NetMsg_MsgType, to ...string) error
	// ReceiveMsg register a MsgHandler to handle the msg received from other node.
	ReceiveMsg(msgType net.NetMsg_MsgType, handler MsgHandler) error

	// Start the net service.
	Start() error

	// Stop the net service.
	Stop() error

	// GetNodeUidByCertId return node uid which mapped to the given cert id. If unmapped return error.
	GetNodeUidByCertId(certId string) (string, error)

	// GetChainNodesInfoProvider return an implementation of ChainNodesInfoProvider.
	GetChainNodesInfoProvider() ChainNodesInfoProvider
}
