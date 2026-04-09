package featureflags

import (
	"context"
	"fmt"
	"os"

	flagsmith "github.com/Flagsmith/flagsmith-go-client/v3"
	"go.uber.org/zap"
)

var (
	client    *flagsmith.Client
	available bool
)

func Init() error {
	apiKey := os.Getenv("FLAGSMITH_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("FLAGSMITH_API_KEY not set")
	}

	opts := []flagsmith.Option{}
	if host := os.Getenv("FLAGSMITH_HOST"); host != "" {
		opts = append(opts, flagsmith.WithBaseURL(host))
	}

	client = flagsmith.NewClient(apiKey, opts...)
	available = true
	zap.L().Info("flagsmith initialized")
	return nil
}

func IsEnabled(flagName string) bool {
	if !available {
		return false
	}
	flags, err := client.GetEnvironmentFlags(context.Background())
	if err != nil {
		return false
	}
	enabled, _ := flags.IsFeatureEnabled(flagName)
	return enabled
}
