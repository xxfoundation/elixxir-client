package channelbot

// everyone's in every channel for now
var subscribers = []uint64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

func AddSubscriber(newSubscriber uint64) {
	subscribers = append(subscribers, newSubscriber)
}

func KickSubscriber(kickedSubscriber uint64) {
	for i, subscriber := range subscribers {
		if subscriber == kickedSubscriber {
			subscribers = append(subscribers[:i], subscribers[i+1:]...)
			return
		}
	}
}
