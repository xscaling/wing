package prometheus

import (
	"errors"
	"math"
	"time"

	wingv1 "github.com/xscaling/wing/api/v1"
	"github.com/xscaling/wing/core/engine"
	"github.com/xscaling/wing/utils"

	"k8s.io/apimachinery/pkg/api/resource"
)

type scaler struct {
	config      PluginConfig
	queryClient QueryClient
}

var _ engine.Scaler = &scaler{}

type Server struct {
	// Left empty to use the default prometheus server
	ServerAddress *string `json:"serverAddress" yaml:"serverAddress"`
	InsecureSSL   *bool   `json:"insecureSSL" yaml:"insecureSSL"`
	// Authenticated is true if the server requires authentication.
	// Auth - bearer token
	BearerToken *string `json:"bearerToken,omitempty" yaml:"bearerToken,omitempty"`
	// Auth - username/password
	Username *string `json:"username,omitempty" yaml:"username,omitempty"`
	Password *string `json:"password,omitempty" yaml:"password,omitempty"`
	// Do not supports TLS authentication currently
}

type Settings struct {
	// Must be a single positive vector response query
	Query string `json:"query"`
	// To filter out jitter of metric
	Threshold float64 `json:"threshold"`

	Server `json:",inline"`
}

func (s *Settings) Validate() error {
	if s.Query == "" {
		return errors.New("query is empty")
	}
	if s.Threshold <= 0 {
		return errors.New("threshold must be positive")
	}
	return nil
}

func New(config PluginConfig) (*scaler, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return &scaler{
		config:      config,
		queryClient: NewQueryClient(config.Timeout),
	}, nil
}

func (s *scaler) Get(ctx engine.ScalerContext) (*engine.ScalerOutput, error) {
	settings := new(Settings)
	if err := ctx.LoadSettings(settings); err != nil {
		return nil, err
	}
	if err := settings.Validate(); err != nil {
		return nil, err
	}
	provisionServer := s.config.DefaultServer
	if settings.ServerAddress != nil {
		provisionServer = settings.Server
	}
	value, err := s.queryClient.Query(provisionServer, settings.Query, time.Now())
	if err != nil {
		return nil, err
	}
	// Start calculating desired replicas
	var (
		averageValue    = 0.0
		desiredReplicas = ctx.CurrentReplicas
	)

	// Empty result or return zero indeed
	if value == 0 || ctx.CurrentReplicas == 0 {
		desiredReplicas = 0
	} else {
		averageValue = value / float64(ctx.CurrentReplicas)

		scaleRatio := averageValue / settings.Threshold
		// due to accuracy issue
		if math.Abs(100.0-scaleRatio*100) >= s.config.Toleration*100 {
			// desiredReplicas = ceil(averageValue / threshold) * currentReplicas
			// If current replicas is zero, then won't trigger scaling whatever the value is.
			desiredReplicas = int32(math.Ceil(scaleRatio * float64(ctx.CurrentReplicas)))
		}
	}
	utils.SetTargetStatus(ctx.AutoscalerStatus, wingv1.TargetStatus{
		Target:          PluginName,
		DesiredReplicas: desiredReplicas,
		Metric: wingv1.MetricTarget{
			Type:         wingv1.AverageValueMetricType,
			AverageValue: resource.NewMilliQuantity(int64(averageValue*1000), resource.DecimalSI),
		},
	})
	return &engine.ScalerOutput{
		DesiredReplicas: desiredReplicas,
	}, nil
}
