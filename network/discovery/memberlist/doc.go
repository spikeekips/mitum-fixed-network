/*
Package memberlist provides node discovery and failed nodes detection by
hashicorp/memberlist.

Basically mitum memberlist uses the two features of hashicorp/memberlist:
 - join
 - leave

mitum memberlist implements Transport layer of hashicorp/memberlist, which
depends on UDP/TCP connections by default. Transport layer of mitum memberlist
uses quic network package of mitum, so there is no additional ports for
discovery.

The publish url of node is translated to virtual IPv6 address, it will
be used to identify node.

mitum memberlist maintains the joined nodes.

mitim memberlist allows multiple connections from same node up to
`discovery.maxNodeConns` by virtual IPv6 address.

To get the joined nodes, mitum memberlist supports, `Discovery.Nodes()`. it
returns `[]NodeConnInfo`. List of `NodeConnInfo` in same node are listed by
added order, but it's order does not mean the joined order.
*/
package memberlist
