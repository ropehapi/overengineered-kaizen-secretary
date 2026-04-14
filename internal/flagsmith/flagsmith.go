package flagsmith

import (
	"context"
	"fmt"
	"time"

	"github.com/Flagsmith/flagsmith-go-client/v3"
)

type Client struct {
	fs *flagsmith.Client
}

func NewClient(apiKey string)*Client{
	fs := flagsmith.NewClient(
		apiKey,
		flagsmith.WithLocalEvaluation(context.Background()),
		flagsmith.WithEnvironmentRefreshInterval(time.Second*5),
		flagsmith.WithBaseURL("http://localhost:8000/api/v1/"),
	)

	return &Client{
		fs: fs,
	}
}

func (c *Client) IsEnable(featureName string) bool{
	flag, err := c.fs.GetEnvironmentFlags(context.Background())
	if err != nil {
		return false
	}

	enabled, err := flag.IsFeatureEnabled(featureName)
	if err != nil {
		return false
	}

	return enabled
}

func (c *Client) GetString(featureName, defaultValue string) string{
	flag, err := c.fs.GetEnvironmentFlags(context.Background())
	if err != nil {
		return defaultValue
	}

	value, err := flag.GetFeatureValue(featureName)
	if err != nil {
		return defaultValue
	}

	if str, ok := value.(string); ok {
		return str
	}

	return defaultValue
}

func (c *Client) PrintAllFlags() {
	flag, err := c.fs.GetEnvironmentFlags(context.Background())
	if err != nil {
		return
	}

	fmt.Printf("Flags atuais: %v", flag)
}