package rabbitmq

import "fmt"

type queueInfo struct {
	Messages               int         `json:"messages"`
	MessagesUnacknowledged int         `json:"messages_unacknowledged"`
	MessageStat            messageStat `json:"message_stats"`
	Name                   string      `json:"name"`
}

type messageStat struct {
	PublishDetail publishDetail `json:"publish_details"`
}

type publishDetail struct {
	Rate float64 `json:"rate"`
}

func getComposedQueue(operation Operation, q []queueInfo) (queueInfo, error) {
	var queue = queueInfo{}
	queue.Name = "composed-queue"
	queue.MessagesUnacknowledged = 0
	if len(q) > 0 {
		switch operation {
		case OperationSum:
			sumMessages, sumRate := getSum(q)
			queue.Messages = sumMessages
			queue.MessageStat.PublishDetail.Rate = sumRate
		case OperationAvg:
			avgMessages, avgRate := getAverage(q)
			queue.Messages = avgMessages
			queue.MessageStat.PublishDetail.Rate = avgRate
		case OperationMax:
			maxMessages, maxRate := getMaximum(q)
			queue.Messages = maxMessages
			queue.MessageStat.PublishDetail.Rate = maxRate
		default:
			return queue,
				fmt.Errorf("operation mode %s must be one of %s, %s, %s",
					operation, OperationSum, OperationAvg, OperationMax)
		}
	} else {
		queue.Messages = 0
		queue.MessageStat.PublishDetail.Rate = 0
	}

	return queue, nil
}

func getSum(q []queueInfo) (int, float64) {
	var sumMessages int
	var sumRate float64
	for _, value := range q {
		sumMessages += value.Messages
		sumRate += value.MessageStat.PublishDetail.Rate
	}
	return sumMessages, sumRate
}

func getAverage(q []queueInfo) (int, float64) {
	sumMessages, sumRate := getSum(q)
	length := len(q)
	return sumMessages / length, sumRate / float64(length)
}

func getMaximum(q []queueInfo) (int, float64) {
	var maxMessages int
	var maxRate float64
	for _, value := range q {
		if value.Messages > maxMessages {
			maxMessages = value.Messages
		}
		if value.MessageStat.PublishDetail.Rate > maxRate {
			maxRate = value.MessageStat.PublishDetail.Rate
		}
	}
	return maxMessages, maxRate
}
