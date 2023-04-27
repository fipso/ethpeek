package main

import (
	"log"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/nat"
	"github.com/ethereum/go-ethereum/params"
)

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
