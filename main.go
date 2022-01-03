package main

import (
	"context"
	"fmt"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	"github.com/libp2p/go-libp2p-core/protocol"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	libp2pquic "github.com/libp2p/go-libp2p-quic-transport"
	"log"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/routing"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	libp2ptls "github.com/libp2p/go-libp2p-tls"
)

func main() {
	run()
}

func run() {
	// The context governs the lifetime of the libp2p node.
	// Cancelling it will stop the the host.

	// Now, normally you do not just want a simple host, you want
	// that is fully configured to best support your p2p application.
	// Let's create a second host setting some more options.
	ctx := context.Background()
	port := 8999
	networkID := "wormhole/testnet/2"
	// Set your own keypair
	priv, _, err := crypto.GenerateKeyPair(
		crypto.Ed25519, // Select your key type. Ed25519 are nice short
		-1,             // Select key length when possible (i.e. RSA).
	)
	if err != nil {
		panic(err)
	}

	h2, err := libp2p.New(ctx,
		// Use the keypair we generated
		libp2p.Identity(priv),
		// Multiple listen addresses
		libp2p.ListenAddrStrings(
			fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic", port),
			fmt.Sprintf("/ip6/::/udp/%d/quic", port),
		),
		// Enable TLS security as the only security protocol.
		libp2p.Security(libp2ptls.ID, libp2ptls.New),

		// Enable QUIC transport as the only transport.
		libp2p.Transport(libp2pquic.NewTransport),
		// support any other default transports (TCP)
		// Let's prevent our peer from having too many
		// connections by attaching a connection manager.
		libp2p.ConnectionManager(connmgr.NewConnManager(
			100,         // Lowwater
			400,         // HighWater,
			time.Minute, // GracePeriod
		)),

		// Let this host use the DHT to find other hosts
		libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
			// TODO(leo): Persistent data store (i.e. address book)
			idht, err := dht.New(ctx, h, dht.Mode(dht.ModeServer),
				// This intentionally makes us incompatible with the global IPFS DHT
				dht.ProtocolPrefix(protocol.ID("/"+networkID)),
			)
			return idht, err
		}),
	)
	if err != nil {
		panic(err)
	}
	defer h2.Close()

	// The last step to get fully up and running would be to connect to
	// bootstrap peers (or any other peers). We leave this commented as
	// this is an example and the peer will die as soon as it finishes, so
	// it is unnecessary to put strain on the network.

	/*
		// This connects to public bootstrappers
		for _, addr := range dht.DefaultBootstrapPeers {
			pi, _ := peer.AddrInfoFromP2pAddr(addr)
			// We ignore errors as some bootstrap peers may be down
			// and that is fine.
			h2.Connect(ctx, *pi)
		}
	*/
	log.Printf("Hello World, my host ID is %s\n", h2.ID().String())
	topic := fmt.Sprintf("%s/%s", networkID, "broadcast")

	log.Println("Subscribing pubsub topic: ", topic)
	ps, err := pubsub.NewGossipSub(ctx, h2)
	if err != nil {
		panic(err)
	}

	th, err := ps.Join(topic)
	if err != nil {
		log.Fatal("failed to join topic: %w", err)
	}

	_, err = th.Subscribe()
	if err != nil {
		log.Fatal("failed to subscribe topic: %w", err)
	}

	select {}
	fmt.Println("shut down bootstrap server...")
}

