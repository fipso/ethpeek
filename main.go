package main

import (
	"errors"
	"log"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/protocols/eth"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/nat"
	"github.com/ethereum/go-ethereum/params"
)

var protocolLengths = map[uint]uint64{eth.ETH68: 17, eth.ETH67: 17, eth.ETH66: 17}

const maxMessageSize = 10 * 1024 * 1024

var txPool *testTxPool

type testTxPool struct {
	pool map[common.Hash]*types.Transaction // Hash map of collected transactions

	txFeed event.Feed   // Notification feed to allow waiting for inclusion
	lock   sync.RWMutex // Protects the transaction pool
}

func main() {
	conf := p2p.Config{
		ListenAddr:      ":30303",
		MaxPeers:        50,
		NAT:             nat.Any(),
		EnableMsgEvents: true,
	}

	// Load private key
	var err error
	conf.PrivateKey, err = crypto.LoadECDSA("key.pem")
	if err != nil {
		log.Fatal(err)
	}

	// Setup tx pool
	txPool = &testTxPool{
		pool: make(map[common.Hash]*types.Transaction),
	}

	// Setup default protocols
	conf.Protocols = makeProtocols()

	// Load default Bootstrap Nodes
	urls := params.MainnetBootnodes
	conf.BootstrapNodes = make([]*enode.Node, 0, len(urls))
	for _, url := range urls {
		if url != "" {
			node, err := enode.Parse(enode.ValidSchemes, url)
			if err != nil {
				log.Fatal(err)
			}
			conf.BootstrapNodes = append(conf.BootstrapNodes, node)
		}
	}

	conf.BootstrapNodesV5 = make([]*enode.Node, 0, len(urls))
	for _, url := range urls {
		if url != "" {
			node, err := enode.Parse(enode.ValidSchemes, url)
			if err != nil {
				log.Fatal(err)
			}
			conf.BootstrapNodesV5 = append(conf.BootstrapNodesV5, node)
		}
	}

	server := &p2p.Server{Config: conf}

	// Start p2p server
	err = server.Start()
	if err != nil {
		log.Fatal(err)
	}

	// Print p2p events
	eventChan := make(chan *p2p.PeerEvent)
	server.SubscribeEvents(eventChan)
	for event := range eventChan {
		log.Printf("P2P Event: %v", event)
		// if event.Type == p2p.PeerEventTypeMsgRecv {
		// 	log.Printf("P2P Event: %v", event)
		// }
	}

	for {
		log.Println("Peers:", server.PeerCount())
		time.Sleep(time.Second)
	}

}

// MakeProtocols constructs the P2P protocol definitions for `eth`.
func makeProtocols() []p2p.Protocol {
	protocols := make([]p2p.Protocol, len(eth.ProtocolVersions))
	for i, version := range eth.ProtocolVersions {
		version := version // Closure

		protocols[i] = p2p.Protocol{
			Name:    eth.ProtocolName,
			Version: version,
			Length:  protocolLengths[version],
			Run: func(p *p2p.Peer, rw p2p.MsgReadWriter) error {
				peer := eth.NewPeer(version, p, rw, *txPool)
				defer peer.Close()

				// return backend.RunPeer(peer, func(peer *Peer) error {
				// 	return Handle(backend, peer)
				// })

				return handleMessage(p, rw)
				//return nil
			},
			NodeInfo: func() interface{} {
				return eth.NodeInfo{
					Network:    1,
					Head:       common.HexToHash("0x96b446813d53ad01c1a4142fec7dafca15d9ec5aa694c94e4420789dd4f4231c"),
					Config:     params.MainnetChainConfig,
					Genesis:    params.MainnetGenesisHash,
					Difficulty: params.MainnetTerminalTotalDifficulty,
				}
			},
			PeerInfo: func(id enode.ID) interface{} {
				// return backend.PeerInfo(id)
				return nil
			},
			// Attributes:     []enr.Entry{currentENREntry(backend.Chain())},
			// DialCandidates: dnsdisc,
		}
	}
	return protocols
}

//func makeHandler() {
//	// h := &handler{
//	// 	networkID:      config.Network,
//	// 	forkFilter:     forkid.NewFilter(config.Chain),
//	// 	eventMux:       config.EventMux,
//	// 	database:       config.Database,
//	// 	txpool:         config.TxPool,
//	// 	chain:          config.Chain,
//	// 	peers:          newPeerSet(),
//	// 	merger:         config.Merger,
//	// 	requiredBlocks: config.RequiredBlocks,
//	// 	quitSync:       make(chan struct{}),
//	// }

//	clock := new(mclock.Simulated)
//	rand := rand.New(rand.NewSource(0x3a29)) // Same used in package tests!!!

//	f := fetcher.NewTxFetcherForTests(
//		func(common.Hash) bool { return false },
//		func(txs []*types.Transaction) []error {
//			log.Println(txs)
//			return make([]error, len(txs))
//		},
//		func(string, []common.Hash) error { return nil },
//		clock, rand,
//	)
//	f.Start()
//	//defer f.Stop()
//}

func handleMessage(peer *p2p.Peer, rw p2p.MsgReadWriter) error {
	// Read the next message from the remote peer, and ensure it's fully consumed
	msg, err := rw.ReadMsg()
	if err != nil {
		return err
	}
	if msg.Size > maxMessageSize {
		return errors.New("message too large")
	}
	defer msg.Discard()

	switch msg.Code {
	case eth.PooledTransactionsMsg:
		handlePooledTransactions66(msg, peer)
	// case eth.NewPooledTransactionHashesMsg:
	// 	if peer.Version
	}

	return nil
}

func handleNewPooledTransactionHashes68(msg eth.Decoder, peer *p2p.Peer) error {
	ann := new(eth.NewPooledTransactionHashesPacket68)
	if err := msg.Decode(ann); err != nil {
		return err
	}
	if len(ann.Hashes) != len(ann.Types) || len(ann.Hashes) != len(ann.Sizes) {
		return errors.New("invalid transaction announcement")
	}
	// Schedule all the unknown hashes for retrieval
	for _, hash := range ann.Hashes {
		log.Println(hash)
	}

	return nil
}


func handleNewPooledTransactionHashes66(msg eth.Decoder, peer *p2p.Peer) error {
	ann := new(eth.NewPooledTransactionHashesPacket66)
	if err := msg.Decode(ann); err != nil {
		return err
	}
	for _, hash := range *ann {
		//peer.markTransaction(hash)
		log.Println(hash)
	}

	return nil
}

func handlePooledTransactions66(msg eth.Decoder, peer *p2p.Peer) error {
	// Transactions can be processed, parse all of them and deliver to the pool
	var txs eth.PooledTransactionsPacket66
	if err := msg.Decode(&txs); err != nil {
		return err
	}
	for _, tx := range txs.PooledTransactionsPacket {
		// Validate and mark the remote transaction
		if tx == nil {
			return errors.New("nil transaction")
		}

		log.Println(tx)
	}
	return nil
}
