package keygen

import (
	"gitlab.com/akil27/moneroTss/blame"
	"gitlab.com/akil27/moneroTss/common"
)

// Response keygen response
type Response struct {
	PoolAddress string        `json:"pool_address"`
	ViewKey     string        `json:"view_key"`
	Status      common.Status `json:"status"`
	Blame       blame.Blame   `json:"blame"`
}

// NewResponse create a new instance of keygen.Response
func NewResponse(addr, viewKey string, status common.Status, blame blame.Blame) Response {
	return Response{
		PoolAddress: addr,
		ViewKey:     viewKey,
		Status:      status,
		Blame:       blame,
	}
}
