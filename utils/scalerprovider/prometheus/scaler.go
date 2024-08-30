package prometheus

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	wingv1 "github.com/xscaling/wing/api/v1"
	"github.com/xscaling/wing/core/engine"
	"github.com/xscaling/wing/utils"

	"k8s.io/apimachinery/pkg/api/resource"
)

var (
	// bytes.Buffer pool used to efficiently generate targetStatus.target mostly
	bufferPool = sync.Pool{
		New: func() interface{} {
			return new(bytes.Buffer)
		},
	}
)

type ScalerConfig struct {
	Toleration     float64       `yaml:"toleration"`
	DefaultTimeout time.Duration `yaml:"defaultTimeout"`
	DefaultServer  Server        `yaml:"defaultServer"`
}

func (c ScalerConfig) Validate() error {
	if c.Toleration < 0 {
		return errors.New("toleration must be non-negative")
	}
	if c.DefaultServer.ServerAddress == nil {
		return errors.New("default server is required")
	}
	return nil
}

const (
	DefaultToleration = 0.05
)

func NewDefaultConfig() *ScalerConfig {
	return &ScalerConfig{
		Toleration:     DefaultToleration,
		DefaultTimeout: 30 * time.Second,
	}
}

type scaler struct {
	pluginName  string
	config      ScalerConfig
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

type FailureMode string

const (
	// This will abort scaling when error occurred
	FailAsError FailureMode = ""
	// Return zero value when error occurred
	FailAsZero FailureMode = "FailAsZero"
	// Return last value stored in status when error occurred, if there is no last value stored then abort scaling
	FailAsLastValue FailureMode = "FailAsLastValue"
)

type Settings struct {
	Server `json:",inline"`

	// Must be a single positive vector response query
	Query string `json:"query"`
	// To filter out jitter of metric
	Threshold float64 `json:"threshold"`

	// Those fallback strategies are aims to avoid scale down or abort when the metric is not available.
	// WARNING: Failover won't working after modify query string
	// Default `FailAsError` means return error when query prometheus failed and this will prevent scaling.
	FailureMode FailureMode `yaml:"failureMode,omitempty"`
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

func New(pluginName string, config ScalerConfig) (*scaler, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return &scaler{
		pluginName:  pluginName,
		config:      config,
		queryClient: NewQueryClient(config.DefaultTimeout),
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
	return s.CalculateDesiredReplicas(ctx, settings)
}

func (s *scaler) CalculateDesiredReplicas(ctx engine.ScalerContext, settings *Settings) (*engine.ScalerOutput, error) {
	if ctx.CurrentReplicas == 0 {
		return &engine.ScalerOutput{
			DesiredReplicas: 0,
		}, nil
	}

	provisionServer := s.config.DefaultServer
	if settings.ServerAddress != nil {
		provisionServer = settings.Server
	}

	// Start calculating desired replicas
	var (
		averageValue             = math.MaxFloat64
		desiredReplicas          = ctx.CurrentReplicas
		shouldUpdateAverageValue = true
	)

	targetStatusName := s.makeTargetStatusName(settings.Query)

	value, err := s.queryClient.Query(provisionServer, settings.Query, time.Now())
	if err != nil {
		// To avoid override status and doing nonsense update
		shouldUpdateAverageValue = false

		switch settings.FailureMode {
		case FailAsError:
			return nil, err
		case FailAsLastValue:
			// Try to get last value from status
			if targetStatus, ok := utils.GetTargetStatus(ctx.AutoscalerStatus, targetStatusName); ok {
				averageValue = float64(targetStatus.Metric.AverageValue.MilliValue()) / 1000
			} else {
				return nil, fmt.Errorf("unable to get latest value from status when failover is enabled: %s", err)
			}
		case FailAsZero:
			value = 0
		default:
			return nil, fmt.Errorf("unknown failure mode: `%s`", settings.FailureMode)
		}
	}
	// Empty result or return zero indeed
	if value == 0 {
		desiredReplicas = 0
	} else {
		// If averageValue is not set then calculate it, otherwise use previous set value
		if averageValue == math.MaxFloat64 {
			averageValue = value / float64(ctx.CurrentReplicas)
		}

		scaleRatio := averageValue / settings.Threshold
		// due to accuracy issue
		if math.Abs(100.0-scaleRatio*100) >= s.config.Toleration*100 {
			// desiredReplicas = ceil(averageValue / threshold) * currentReplicas
			// If current replicas is zero, then won't trigger scaling whatever the value is.
			desiredReplicas = int32(math.Ceil(scaleRatio * float64(ctx.CurrentReplicas)))
		}
	}
	if shouldUpdateAverageValue {
		utils.SetTargetStatus(ctx.AutoscalerStatus, wingv1.TargetStatus{
			Target:          targetStatusName,
			Scaler:          s.pluginName,
			DesiredReplicas: desiredReplicas,
			Metric: wingv1.MetricTarget{
				Type:         wingv1.AverageValueMetricType,
				AverageValue: resource.NewMilliQuantity(int64(averageValue*1000), resource.DecimalSI),
			},
		})
	}
	return &engine.ScalerOutput{
		DesiredReplicas:     desiredReplicas,
		ManagedTargetStatus: []string{targetStatusName},
	}, nil
}

func (s *scaler) makeTargetStatusName(query string) string {
	b := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(b)

	b.Reset()
	b.WriteString(query)
	return s.pluginName + "/" + utils.FarmHash(b)
}
