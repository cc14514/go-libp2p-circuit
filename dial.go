package relay

import (
	"context"
	"fmt"
	"github.com/libp2p/go-libp2p-core/transport"
	"math/rand"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/libp2p/go-libp2p-core/peer"
)
/*
func (d *RelayTransport) Dial(ctx context.Context, a ma.Multiaddr, p peer.ID) (transport.CapableConn, error) {
	c, err := d.Relay().Dial(ctx, a, p)
	if err != nil {
		return nil, err
	}
	c.tagHop()
	return d.upgrader.UpgradeOutbound(ctx, d, c, p)
}

func (r *Relay) Dial(ctx context.Context, a ma.Multiaddr, p peer.ID) (*Conn, error) {
	// split /a/p2p-circuit/b into (/a, /p2p-circuit/b)
	relayaddr, destaddr := ma.SplitFunc(a, func(c ma.Component) bool {
		return c.Protocol().Code == P_CIRCUIT
	})
	fmt.Println("<<<<<<<<<<<<<<<<<<<<")
	fmt.Println(p.Pretty(), a.String())
	fmt.Println("<<<<<<<<<<<<<<<<<<<<")
	// If the address contained no /p2p-circuit part, the second part is nil.
	if destaddr == nil {
		return nil, fmt.Errorf("%s is not a relay address", a)
	}

	if relayaddr == nil {
		return nil, fmt.Errorf(
			"can't dial a p2p-circuit without specifying a relay: %s",
			a,
		)
	}

	// Strip the /p2p-circuit prefix from the destaddr.
	_, destaddr = ma.SplitFirst(destaddr)

	dinfo := &peer.AddrInfo{ID: p, Addrs: []ma.Multiaddr{}}
	if destaddr != nil {
		dinfo.Addrs = append(dinfo.Addrs, destaddr)
	}

	var rinfo *peer.AddrInfo
	rinfo, err := peer.AddrInfoFromP2pAddr(relayaddr)
	if err != nil {
		return nil, fmt.Errorf("error parsing multiaddr '%s': %s", relayaddr.String(), err)
	}

	return r.DialPeer(ctx, *rinfo, *dinfo)
}

*/


// add by liangc >>>>
func (d *RelayTransport) Dial(ctx context.Context, a ma.Multiaddr, p peer.ID) (transport.CapableConn, error) {
	c, err := d.Relay().Dial(ctx, a, p)
	if err != nil {
		return nil, err
	}
	c.tagHop()
	return d.upgrader.UpgradeOutbound(ctx, d, c, p)
}

func (r *Relay) Dial(ctx context.Context, a ma.Multiaddr, p peer.ID) (*Conn, error) {
	if !r.Matches(a) {
		return nil, fmt.Errorf("%s is not a relay address", a)
	}
	parts := ma.Split(a)

	spl := ma.Cast(ma.CodeToVarint(P_CIRCUIT))
	//splNodelay := ma.Cast(ma.CodeToVarint(P_CIRCUIT_NODELAY))
	//aaa := r.host.Peerstore().PeerInfo(p).Addrs
	//fmt.Println("ZZZZZZZZZZZZZZZZZZ", p.Pretty(), a, "--", aaa)
	var relayaddr, destaddr ma.Multiaddr
	for i, p := range parts {
		//if p.Equal(splNodelay) {
		//ctx = context.WithValue(ctx, NodelayProtocol.Name, "true")
		//} else if p.Equal(spl) {
		if p.Equal(spl) {
			relayaddr = ma.Join(parts[:i]...)
			destaddr = ma.Join(parts[i+1:]...)
			//break
		}
	}
	dinfo := &peer.AddrInfo{ID: p, Addrs: []ma.Multiaddr{}}
	if len(destaddr.Bytes()) > 0 {
		dinfo.Addrs = append(dinfo.Addrs, destaddr)
	}
	if len(relayaddr.Bytes()) == 0 {
		// unspecific relay address, try dialing using known hop relays
		return r.tryDialRelays(ctx, *dinfo)
	}

	var rinfo *peer.AddrInfo
	rinfo, err := peer.AddrInfoFromP2pAddr(relayaddr)
	if err != nil {
		return nil, fmt.Errorf("error parsing multiaddr '%s': %s", relayaddr.String(), err)
	}

	return r.DialPeer(ctx, *rinfo, *dinfo)
}

func (r *Relay) tryDialRelays(ctx context.Context, dinfo peer.AddrInfo) (*Conn, error) {
	var relays []peer.ID
	r.mx.Lock()
	for p := range r.relays {
		relays = append(relays, p)
	}
	r.mx.Unlock()

	// shuffle list of relays, avoid overloading a specific relay
	for i := range relays {
		j := rand.Intn(i + 1)
		relays[i], relays[j] = relays[j], relays[i]
	}

	for _, relay := range relays {
		if len(r.host.Network().ConnsToPeer(relay)) == 0 {
			continue
		}

		rctx, cancel := context.WithTimeout(ctx, HopConnectTimeout)
		c, err := r.DialPeer(rctx, peer.AddrInfo{ID: relay}, dinfo)
		cancel()

		if err == nil {
			return c, nil
		}

		log.Debugf("error opening relay connection through %s: %s", dinfo.ID, err.Error())
	}

	return nil, fmt.Errorf("Failed to dial through %d known relay hosts", len(relays))
}

// add by liangc <<<<
