// Package ecstp (ecs-task-protection) provides an easy function for enabling and disabling ECS
// task termination protection and can be called from inside an ECS task. See
// https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task-scale-in-protection.html
package ecstp

import (
	"context"
	"encoding/json"
	"errors"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/go-resty/resty/v2"
)

// MetadataBody represents the JSON body returned from the metadata task API
type MetadataBody struct {
	Cluster string `json:"Cluster"`
	TaskARN string `json:"TaskARN"`
}

// UpdateTaskProtectionInput defines the parameters required for UpdateTaskProtection.
//
// Client must not be nil. ExpiresInMinutes must be between 1 and 2880, but can be nil.
// Setting to nil will use the default protection period. See
// https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task-scale-in-protection.html
type UpdateTaskProtectionInput struct {
	Context          context.Context
	Client           *ecs.Client
	Protect          bool
	ExpiresInMinutes *int32
}

// GetTaskArn calls the Instance metadata API to retrieve the current Cluster and Task ARN.
//
// The Instance metadata API URI is obtained through the env variable `ECS_CONTAINER_METADATA_URI_V4`.
// Returns a pointer to struct MetadataBody representing the API response or returns an error if the
// env variable cannot be found, the API was unreachable or the response can't be unmarshalled.
func GetTaskArn() (*MetadataBody, error) {
	ecsMetadataEndpoint, ok := os.LookupEnv("ECS_CONTAINER_METADATA_URI_V4")
	if !ok {
		return nil, errors.New("unable to retrieve Task ARN - can't get Metadata URI")
	}

	client := resty.New()
	res, err := client.R().Get(ecsMetadataEndpoint + "/task")
	if err != nil {
		return nil, err
	}

	var metadata *MetadataBody
	if err = json.Unmarshal(res.Body(), &metadata); err != nil {
		return nil, err
	}

	return metadata, nil
}

// UpdateTaskProtection uses the provided input to enable or disable task protection.
//
// UpdateTaskProtection makes calls GetTaskArn to retrieve the Cluster and Task ARN and then calls
// the UpdateTaskProtection ECS API to enable or disable protection. Returns any errors from the
// ECS API or an error if task protection fails to enable.
func UpdateTaskProtection(input *UpdateTaskProtectionInput) error {
	metadata, err := GetTaskArn()
	if err != nil {
		return err
	}

	output, err := input.Client.UpdateTaskProtection(input.Context, &ecs.UpdateTaskProtectionInput{
		Cluster: aws.String(metadata.Cluster),
		Tasks: []string{
			metadata.TaskARN,
		},
		ProtectionEnabled: input.Protect,
		ExpiresInMinutes:  input.ExpiresInMinutes,
	})
	if err != nil {
		return err
	}

	failures := output.Failures
	if len(failures) > 0 && input.Protect {
		return errors.New("failed to protect task: " + *failures[0].Reason)
	}

	return nil
}
