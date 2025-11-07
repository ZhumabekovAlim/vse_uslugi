package ws

// IntercityEvent describes websocket notifications about intercity orders.
type IntercityEvent struct {
	Type   string      `json:"type"`
	Action string      `json:"action"`
	Order  interface{} `json:"order"`
}
