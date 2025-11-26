package memory

import (
	"github.com/stretchr/testify/mock"
)

// MetricsMockcomponent is a mock implementation of Interface.
type MetricsMockcomponent struct {
	mock.Mock
}

// NewMetricsMockcomponent creates a new MetricsMockcomponent.
func NewMetricsMockcomponent() *MetricsMockcomponent {
	return &MetricsMockcomponent{}
}
