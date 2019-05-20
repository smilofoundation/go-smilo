package discover

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"time"
)

type Node interface {
	String() string

}
type NodeID [64]byte
type nodeImpl struct {
	IP       net.IP // len 4 for IPv4 or 16 for IPv6
	UDP, TCP uint16 // port numbers
	ID       NodeID // the node's public key

	// Time when the node was added to the table.
	addedAt time.Time
}

func NewNode(id NodeID, ip net.IP, udpport uint16, tcpport uint16) *Node {
	var node Node
    node = &nodeImpl{IP: ip, UDP: udpport, TCP: tcpport, ID: id, addedAt:time.Now()}
    return &node
}

func PubkeyID(pub *ecdsa.PublicKey) NodeID {
	var id NodeID
	pbytes := elliptic.Marshal(pub.Curve, pub.X, pub.Y)
	if len(pbytes)-1 != len(id) {
		panic(fmt.Errorf("need %d bit pubkey, got %d bits", (len(id)+1)*8, len(pbytes)))
	}
	copy(id[:], pbytes[1:])
	return id
}

func (n *nodeImpl) String() string {
	u := url.URL{Scheme: "enode"}
	if n.IP == nil {
		u.Host = fmt.Sprintf("%x", n.ID[:])
	} else {
		addr := net.TCPAddr{IP: n.IP, Port: int(n.TCP)}
		u.User = url.User(fmt.Sprintf("%x", n.ID[:]))
		u.Host = addr.String()
		if n.UDP != n.TCP {
			u.RawQuery = "discport=" + strconv.Itoa(int(n.UDP))
		}
	}
	return u.String()
}