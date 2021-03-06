// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2016 Canonical Ltd
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License version 3 as
 * published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package builtin

import "github.com/snapcore/snapd/interfaces"

// http://bazaar.launchpad.net/~ubuntu-security/ubuntu-core-security/trunk/view/head:/data/apparmor/policygroups/ubuntu-core/16.04/network
const networkConnectedPlugAppArmor = `
# Description: Can access the network as a client.
# Usage: common
#include <abstractions/nameservice>
#include <abstractions/ssl_certs>

@{PROC}/sys/net/core/somaxconn r,
@{PROC}/sys/net/ipv4/tcp_fastopen r,
`

// http://bazaar.launchpad.net/~ubuntu-security/ubuntu-core-security/trunk/view/head:/data/seccomp/policygroups/ubuntu-core/16.04/network
const networkConnectedPlugSecComp = `
# Description: Can access the network as a client.
# Usage: common
connect
getpeername
getsockname
getsockopt
recv
recvfrom
recvmmsg
recvmsg
send
sendmmsg
sendmsg
sendto
setsockopt
shutdown

# LP: #1446748 - limit this to AF_UNIX/AF_LOCAL and perhaps AF_NETLINK
socket

# This is an older interface and single entry point that can be used instead
# of socket(), bind(), connect(), etc individually. While we could allow it,
# we wouldn't be able to properly arg filter socketcall for AF_INET/AF_INET6
# when LP: #1446748 is implemented.
socketcall
`

// NewNetworkInterface returns a new "network" interface.
func NewNetworkInterface() interfaces.Interface {
	return &commonInterface{
		name: "network",
		connectedPlugAppArmor: networkConnectedPlugAppArmor,
		connectedPlugSecComp:  networkConnectedPlugSecComp,
		reservedForOS:         true,
		autoConnect:           true,
	}
}
