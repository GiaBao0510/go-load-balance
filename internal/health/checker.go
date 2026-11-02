package health

import "context"

type HealthChecker interface {
	Run(ctx context.Context)
}