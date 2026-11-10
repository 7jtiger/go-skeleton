package controller

import "time"

func (p *SignalingController) enqueueBroadcast(room *VDRoom, message *BroadcastMessage) bool {
	select {
	case <-p.ctx.Done():
		return false
	case <-room.done:
		return false
	default:
	}
	select {
	case room.Broadcast <- message:
		return true
	case <-p.ctx.Done():
		return false
	case <-room.done:
		return false
	default:
		return false
	}
}

func (room *VDRoom) removeClient(client *WSClient) {
	room.mu.Lock()
	delete(room.Clients, client)
	delete(room.UserNames, client)
	room.lastActivity = time.Now()
	room.mu.Unlock()
	client.closeSend()
}

func (p *SignalingController) enqueueWaitingEvent(event *BroadcastMessage) {
	select {
	case p.waitingEvents <- event:
	case <-p.ctx.Done():
	default: // Presence is advisory; get-user-list remains the authoritative snapshot.
	}
}

func (p *SignalingController) waitingEventWorker() {
	for {
		select {
		case <-p.ctx.Done():
			return
		case first := <-p.waitingEvents:
			events := []*BroadcastMessage{first}
			// One client-map snapshot per burst, preserving existing individual events.
			for len(events) < 64 {
				select {
				case event := <-p.waitingEvents:
					events = append(events, event)
				default:
					goto broadcast
				}
			}
		broadcast:
			p.waitingRoom.mu.RLock()
			clients := make([]*WSClient, 0, len(p.waitingRoom.Clients))
			for _, client := range p.waitingRoom.Clients {
				clients = append(clients, client)
			}
			p.waitingRoom.mu.RUnlock()
			for _, client := range clients {
				for _, event := range events {
					if client != event.sender && !p.trySend(client, event.message) {
						client.conn.Close()
						break
					}
				}
			}
		}
	}
}
