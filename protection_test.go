// Package ecstp (ecs-task-protection) provides an easy function for enabling and disabling ECS
// task termination protection and can be called from inside an ECS task. See
// https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task-scale-in-protection.html
package ecstp

import (
	"context"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/stretchr/testify/assert"
)

type SuccessfulTestClient struct{}

func (c *SuccessfulTestClient) UpdateTaskProtection(
	ctx context.Context, params *ecs.UpdateTaskProtectionInput, optFns ...func(*ecs.Options),
) (*ecs.UpdateTaskProtectionOutput, error) {
	return &ecs.UpdateTaskProtectionOutput{}, nil
}

type FailureTestClient struct{}

func (c *FailureTestClient) UpdateTaskProtection(
	ctx context.Context, params *ecs.UpdateTaskProtectionInput, optFns ...func(*ecs.Options),
) (*ecs.UpdateTaskProtectionOutput, error) {
	return &ecs.UpdateTaskProtectionOutput{
		Failures: []types.Failure{
			{
				Reason: aws.String("test"),
			},
		},
	}, nil
}

type UpdateTaskProtectionTest struct {
	Name     string
	Input    *UpdateTaskProtectionInput
	ExpError string
}

func TestUpdateTaskProtection(t *testing.T) {
	tests := []UpdateTaskProtectionTest{
		{
			Name: "failures from the ECS API should cause an error to be returned",
			Input: &UpdateTaskProtectionInput{
				Context: context.TODO(),
				Client:  &FailureTestClient{},
				Metadata: &MetadataBody{
					Cluster: "test",
					TaskARN: "test",
				},
				Protect: true,
			},
			ExpError: "failed to protect task: test",
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			err := UpdateTaskProtection(test.Input)
			assert.Error(t, err)
			assert.Equal(t, test.ExpError, err.Error())
		})
	}
}

type GetTaskArnTest struct {
	Name     string
	ExpError string
}

func TestGetTaskArn(t *testing.T) {
	os.Unsetenv("ECS_CONTAINER_METADATA_URI_V4")

	tests := []GetTaskArnTest{
		{
			Name:     "should throw an error when ECS_CONTAINER_METADATA_URI_V4 env var is not set",
			ExpError: "unable to retrieve Task ARN - can't get Metadata URI",
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			meta, err := GetTaskArn()
			assert.Nil(t, meta)
			if assert.Error(t, err) {
				assert.Equal(t, test.ExpError, err.Error())
			}
		})
	}
}
