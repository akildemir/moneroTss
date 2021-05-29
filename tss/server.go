package tss

import (
	"github.com/akildemir/moneroTss/monero_multi_sig/keygen"
	"github.com/akildemir/moneroTss/monero_multi_sig/keysign"
)

// Server define the necessary functionality should be provide by a TSS Server implementation
type Server interface {
	Start() error
	Stop()
	GetLocalPeerID() string
	Keygen(req keygen.Request) (keygen.Response, error)
	KeySign(req keysign.Request) (keysign.Response, error)
}
