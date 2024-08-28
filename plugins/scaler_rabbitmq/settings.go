package rabbitmq

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/xscaling/wing/utils/http/client"
	"github.com/xscaling/wing/utils/http/client/encoding"

	"github.com/streadway/amqp"
)

type Mode string

const (
	ModeQueueLength Mode = "QueueLength"
	ModeMessageRate Mode = "MessageRate"
)

type Protocol string

const (
	ProtocolHTTP Protocol = "http"
	ProtocolAMQP Protocol = "amqp"
)

type MessageType string

const (
	MessageTypeAll            MessageType = "all"
	MessageTypeUnacknowledged MessageType = "unacknowledged"
	MessageTypeReady          MessageType = "ready"
	DefaultMessageType                    = MessageTypeAll
)

type Operation string

const (
	OperationSum Operation = "sum"
	OperationAvg Operation = "avg"
	OperationMax Operation = "max"
)

type Settings struct {
	// QueueLength or MessageRate
	Mode Mode `json:"mode"`
	// Trigger value (queue length or publish/sec. rate)
	Value float64 `json:"value"`
	// Connection string for either HTTP or AMQP protocol
	Host string `json:"host"`
	// Either http or amqp protocol
	// If empty, it will be set from host scheme
	Protocol Protocol `json:"protocol"`

	// Name of queue
	QueueName string `json:"queueName"`
	// Override the vhost from the connection info
	VhostName *string `json:"vhostName"`
	// Message type
	MessageType MessageType `json:"messageType"`

	// Specify if the queueName contains a regex
	UseRegex bool `json:"useRegex"`
	// Specify the operation to apply in case of multiples queues
	Operation Operation `json:"operation"`

	// Custom metric name for trigger
	MetricName string `json:"metricName"`
}

func (s *Settings) Validate() error {
	hostURL, err := url.Parse(s.Host)
	for text, hit := range map[string]bool{
		"mode is required with valid value":                     s.Mode != ModeMessageRate && s.Mode != ModeQueueLength,
		"value must be positive":                                s.Value <= 0,
		fmt.Sprintf("host is invalid: %s", err):                 err != nil,
		"queue name is required":                                s.QueueName == "",
		"operation is required with valid value if using regex": s.UseRegex && s.Operation != OperationSum && s.Operation != OperationAvg && s.Operation != OperationMax,
	} {
		if hit {
			return errors.New(text)
		}
	}
	if s.Protocol == "" {
		if hostURL != nil {
			switch hostURL.Scheme {
			case "amqp", "amqps":
				s.Protocol = ProtocolAMQP
			case "http", "https":
				s.Protocol = ProtocolHTTP
			default:
				return fmt.Errorf("unknown host URL scheme `%s`", hostURL.Scheme)
			}
		}
	}
	switch s.MessageType {
	case MessageTypeAll, MessageTypeUnacknowledged, MessageTypeReady:
		// Valid, do nothing
	case "":
		// Fill with default value
		s.MessageType = DefaultMessageType
	default:
		return fmt.Errorf("unknown message type `%s`", s.MessageType)
	}
	return nil
}

const statusMetricNameJoiner = "/"

func (s *Settings) GetStatusMetricName() (result string) {
	splits := []string{PluginName}
	if mn := s.MetricName; mn != "" {
		splits = append(splits, mn)
	} else {
		splits = append(splits, s.QueueName)
		if s.Mode == ModeQueueLength {
			splits = append(splits, "length")
		} else {
			splits = append(splits, "rate")
		}
	}
	return strings.Join(splits, statusMetricNameJoiner)
}

func (s *Settings) request(timeout time.Duration) (metricValue float64, err error) {
	requester := s.requestViaChannel
	if s.Protocol == ProtocolHTTP {
		requester = s.requestViaHTTP
	}
	messages, publishRate, err := requester(timeout)
	if err != nil {
		return
	}
	metricValue = publishRate
	if s.Mode == ModeQueueLength {
		metricValue = float64(messages)
	}
	return
}

func getInitMessagesAndPublishRate() (messages int, publishRate float64) {
	return -1, -1
}

func (s *Settings) requestViaHTTP(timeout time.Duration) (messages int, publishRate float64, err error) {
	messages, publishRate = getInitMessagesAndPublishRate()

	parsedURL, err := url.Parse(s.Host)
	if err != nil {
		return
	}

	resourceFormat := "api/queues"
	vhost := parsedURL.Path
	// Override vhost if requested.
	if s.VhostName != nil {
		vhost = "/" + *s.VhostName
	}
	// Override vhost with subpath seperator if vhost is empty
	if vhost == "" || vhost == "/" || vhost == "//" {
		vhost = "/%2F/"
	}

	// Construct path and options
	var (
		info  queueInfo
		infos []queueInfo
	)
	options := []client.Option{
		client.WithExpectedStatusCode(http.StatusOK),
	}
	if s.UseRegex {
		query := url.Values{}
		query.Add("use_regex", "true")
		query.Add("pagination", "false")
		query.Add("name", s.QueueName)
		options = append(options, client.WithQuery(query),
			client.WithReceiver(&infos, errors.New("invalid info list response body")))
	} else {
		resourceFormat = resourceFormat + vhost + s.QueueName
		options = append(options, client.WithReceiver(&info, errors.New("invalid info response body")))
	}

	// Get endpoint
	parsedURL.Path = ""
	endpoint := parsedURL.String()

	// Request
	requester := client.NewRequester(encoding.JSONEncoding{}, timeout)
	_, err = client.NewClient(requester, endpoint).Request(http.MethodGet, resourceFormat, options...)
	if err != nil {
		return
	}

	// Composed infos by operation if using regex
	if s.UseRegex {
		info, err = getComposedQueue(s.Operation, infos)
		if err != nil {
			return
		}
	}

	// Set message by type
	switch s.MessageType {
	case MessageTypeAll:
		messages = info.Messages
	case MessageTypeReady:
		messages = info.MessageReady
	case MessageTypeUnacknowledged:
		messages = info.MessagesUnacknowledged
	}

	return messages, info.MessageStat.PublishDetail.Rate, nil
}

const (
	defaultHeartbeat = 10 * time.Second
	defaultLocale    = "en_US"
)

func (s *Settings) requestViaChannel(timeout time.Duration) (messages int, publishRate float64, err error) {
	messages, publishRate = getInitMessagesAndPublishRate()
	conn, err := amqp.DialConfig(s.Host, amqp.Config{
		Dial:      amqp.DefaultDial(timeout),
		Heartbeat: defaultHeartbeat,
		Locale:    defaultLocale,
	})
	if err != nil {
		return
	}
	defer func() { _ = conn.Close() }()
	channel, err := conn.Channel()
	if err != nil {
		return
	}
	items, err := channel.QueueInspect(s.QueueName)
	if err == nil {
		messages = items.Messages
		publishRate = 0
	}
	return
}
