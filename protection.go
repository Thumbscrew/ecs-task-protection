// Package ecstp (ecs-task-protection) provides an easy function for enabling and disabling ECS
// task termination protection and can be called from inside an ECS task. See
// https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task-scale-in-protection.html.
package ecstp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
)

// ECSClient is an interface representing the AWS ECS Client.
type ECSClient interface {
	UpdateTaskProtection(
		ctx context.Context, params *ecs.UpdateTaskProtectionInput, optFns ...func(*ecs.Options),
	) (*ecs.UpdateTaskProtectionOutput, error)
}

// MetadataBody represents the JSON body returned from the metadata task API.
type MetadataBody struct {
	Cluster string `json:"Cluster"`
	TaskARN string `json:"TaskARN"`
}

// Client is a wrapper around an ECS Client that enables and disables ECS task protection.
type Client struct {
	ECSClient
	MetadataEndpointOverride string
}

func NewClient(ecsClient ECSClient) *Client {
	return &Client{
		ECSClient: ecsClient,
	}
}

// UpdateTaskProtectionInput defines the parameters required for UpdateTaskProtection.
//
// If Metadata is nil, UpdateTaskProtection will attempt to get the metadata via GetTaskArn.
// ExpiresInMinutes must be between 1 and 2880, but can be nil. Setting to nil will use the default
// protection period. See
// https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task-scale-in-protection.html.
type UpdateTaskProtectionInput struct {
	Metadata         *MetadataBody
	Protect          bool
	ExpiresInMinutes *int32
}

// GetTaskArn calls the Instance metadata API to retrieve the current Cluster and Task ARN.
//
// The Instance metadata API URI is obtained through the env variable `ECS_CONTAINER_METADATA_URI_V4`.
// Returns a pointer to struct MetadataBody representing the API response or returns an error if the
// env variable cannot be found, the API was unreachable or the response can't be unmarshalled.
func (c *Client) GetTaskArn(ctx context.Context) (*MetadataBody, error) {
	ecsMetadataEndpoint := c.MetadataEndpointOverride

	if ecsMetadataEndpoint == "" {
		var ok bool
		ecsMetadataEndpoint, ok = os.LookupEnv("ECS_CONTAINER_METADATA_URI_V4")
		if !ok {
			return nil, errors.New("unable to retrieve Task ARN - can't get Metadata URI")
		}
	}

	req, err := http.NewRequestWithContext(ctx, "GET", ecsMetadataEndpoint+"/task", nil)
	if err != nil {
		return nil, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	b, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var metadata *MetadataBody
	if err = json.Unmarshal(b, &metadata); err != nil {
		return nil, err
	}

	return metadata, nil
}

// UpdateTaskProtection uses the provided input to enable or disable task protection.
//
// UpdateTaskProtection calls GetTaskArn to retrieve the Cluster and Task ARN (if not provided via
// Metadata in input) and then calls the UpdateTaskProtection ECS API to enable or disable
// protection. Directly returns the result of the UpdateTaskProtection.
func (c *Client) UpdateTaskProtection(ctx context.Context, input *UpdateTaskProtectionInput) (*ecs.UpdateTaskProtectionOutput, error) {
	var metadata *MetadataBody
	if input.Metadata == nil {
		var err error
		metadata, err = c.GetTaskArn(ctx)
		if err != nil {
			return nil, err
		}
	} else {
		metadata = input.Metadata
	}

	return c.ECSClient.UpdateTaskProtection(ctx, &ecs.UpdateTaskProtectionInput{
		Cluster: aws.String(metadata.Cluster),
		Tasks: []string{
			metadata.TaskARN,
		},
		ProtectionEnabled: input.Protect,
		ExpiresInMinutes:  input.ExpiresInMinutes,
	})
}
