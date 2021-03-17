package metaname

type metanameRR struct {
	Reference string `json:"reference,omitempty"`
	Name      string `json:"name,omitempty"`
	Type      string `json:"type,omitempty"`
	Aux       int    `json:"aux,omitempty"`
	Ttl       int    `json:"ttl,omitempty"`
	Data      string `json:"data,omitempty"`
}

type rpcRequest struct {
	Jsonrpc string        `json:"jsonrpc"`
	Id      string        `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

type metanameResponse struct {
	Jsonrpc string            `json:"jsonrpc"`
	Id      string            `json:"id"`
	Result  interface{}       `json:"result,omitempty"`
	Error   metanameErrorInfo `json:"error,omitempty"`
}

type metanameErrorInfo struct {
	Code    int         `json:"code"`
	Data    interface{} `json:"data"`
	Message string      `json:"message"`
}
