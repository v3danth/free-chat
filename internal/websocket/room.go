package websocket

type Room struct {
	ID uint64

	Clients map[*Client]bool
}

func NewRoom(id uint64) *Room {
	return &Room{
		ID:      id,
		Clients: make(map[*Client]bool),
	}
}
