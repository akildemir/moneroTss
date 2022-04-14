package keygen

// Request request to do keygen
type Request struct {
	Keys         []string `json:"keys"`
	RpcAddress   string   `json:"rpc_address"`
	BlockHeight  int64    `json:"block_height"`
	KeygenHeight string   `json:"keygen_height"`
	Version      string   `json:"tss_version"`
}

// NewRequest creeate a new instance of keygen.Request
func NewRequest(keys []string, blockHeight int64, version, rpcAddress string) Request {
	return Request{
		Keys:        keys,
		BlockHeight: blockHeight,
		Version:     version,
		RpcAddress:  rpcAddress,
	}
}
