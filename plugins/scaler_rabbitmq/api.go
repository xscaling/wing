package rabbitmq

import "fmt"

type queueInfo struct {
	Messages               int         `json:"messages"`
	MessagesUnacknowledged int         `json:"messages_unacknowledged"`
	MessageReady           int         `json:"messages_ready"`
	MessageStat            messageStat `json:"message_stats"`
	Name                   string      `json:"name"`
}

type messageStat struct {
	PublishDetail publishDetail `json:"publish_details"`
}

type publishDetail struct {
	Rate float64 `json:"rate"`
}

func getComposedQueue(operation Operation, q []queueInfo) (queue queueInfo, err error) {
	if len(q) == 0 {
		return
	}
	operator := getSum
	switch operation {
	case OperationSum:
		operator = getSum
	case OperationAvg:
		operator = getAverage
	case OperationMax:
		operator = getMaximum
	default:
		err = fmt.Errorf("operation mode %s must be one of %s, %s, %s",
			operation, OperationSum, OperationAvg, OperationMax)
		return
	}
	queue.Messages, queue.MessageReady, queue.MessagesUnacknowledged,
		queue.MessageStat.PublishDetail.Rate = operator(q)
	return queue, nil
}

func getSum(q []queueInfo) (
	sumMessages int,
	sumMessagesReady int,
	sumMessagesUnacknowledged int,
	sumRate float64,
) {
	for _, value := range q {
		sumMessages += value.Messages
		sumMessagesReady += value.MessageReady
		sumMessagesUnacknowledged += value.MessagesUnacknowledged
		sumRate += value.MessageStat.PublishDetail.Rate
	}
	return sumMessages, sumMessagesReady, sumMessagesUnacknowledged, sumRate
}

func getAverage(q []queueInfo) (int, int, int, float64) {
	sumMessages, sumMessagesReady, sumMessagesUnacknowledged, sumRate := getSum(q)
	length := len(q)
	return sumMessages / length, sumMessagesReady / length, sumMessagesUnacknowledged / length, sumRate / float64(length)
}

func getMaximum(q []queueInfo) (
	maxMessages int,
	maxMessagesReady int,
	maxMessagesUnacknowledged int,
	maxRate float64,
) {
	for _, value := range q {
		if value.Messages > maxMessages {
			maxMessages = value.Messages
		}
		if value.MessageReady > maxMessagesReady {
			maxMessagesReady = value.MessageReady
		}
		if value.MessagesUnacknowledged > maxMessagesUnacknowledged {
			maxMessagesUnacknowledged = value.MessagesUnacknowledged
		}
		if value.MessageStat.PublishDetail.Rate > maxRate {
			maxRate = value.MessageStat.PublishDetail.Rate
		}
	}
	return maxMessages, maxMessagesReady, maxMessagesUnacknowledged, maxRate
}
