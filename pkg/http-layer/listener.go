package httplayer

import (
	"net"
	"sync"

	"github.com/stay-miku-39/split-socket/pkg/utils"
)

type SessionId [16]byte

type HTTPListener struct {
	underlayer          net.Listener
	c2sConnMutex        sync.Mutex
	c2sConnPoolMap      map[SessionId]*HTTPServerConnection
	s2cConnMutex        sync.Mutex
	s2cConnPoolMap      map[SessionId]*HTTPServerConnection
	timeoutS2CConnMutex sync.Mutex
	timeoutS2CConnList  *utils.ChainList[*HTTPServerConnection]
}
