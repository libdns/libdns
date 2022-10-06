package njalla

type NjallaRequest struct {
	Method string      `json:"method"`
	Params interface{} `json:"params"`
}

type NjallaRecord struct {
	ID      string `json:"id"`
	Content string `json:"content"`
	Domain  string `json:"domain"`
	Name    string `json:"name"`
	TTL     int    `json:"ttl"`
	Type    string `json:"type"`
}
