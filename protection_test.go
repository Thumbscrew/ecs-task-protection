// Package ecstp (ecs-task-protection) provides an easy function for enabling and disabling ECS
// task termination protection and can be called from inside an ECS task. See
// https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task-scale-in-protection.html
package ecstp

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
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
	protectedTasks := make([]types.ProtectedTask, len(params.Tasks))

	for i, task := range params.Tasks {
		protectedTasks[i] = types.ProtectedTask{
			TaskArn:           &task,
			ProtectionEnabled: params.ProtectionEnabled,
		}
	}

	return &ecs.UpdateTaskProtectionOutput{
		ProtectedTasks: protectedTasks,
	}, nil
}

type FailureTestClient struct{}

func (c *FailureTestClient) UpdateTaskProtection(
	ctx context.Context, params *ecs.UpdateTaskProtectionInput, optFns ...func(*ecs.Options),
) (*ecs.UpdateTaskProtectionOutput, error) {
	failedTasks := make([]types.Failure, len(params.Tasks))

	for i, task := range params.Tasks {
		failedTasks[i] = types.Failure{
			Arn:    &task,
			Reason: aws.String("failed"),
		}
	}

	return &ecs.UpdateTaskProtectionOutput{
		Failures: failedTasks,
	}, nil
}

func TestClient_UpdateTaskProtection(t *testing.T) {
	type fields struct {
		ECSClient                ECSClient
		MetadataEndpointOverride string
	}
	type args struct {
		ctx   context.Context
		input *UpdateTaskProtectionInput
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *ecs.UpdateTaskProtectionOutput
		wantErr bool
	}{
		{
			name: "should return a successful response with protection enabled",
			fields: fields{
				ECSClient: &SuccessfulTestClient{},
			},
			args: args{
				ctx: context.Background(),
				input: &UpdateTaskProtectionInput{
					Metadata: &MetadataBody{
						TaskARN: "test",
					},
					Protect: true,
				},
			},
			want: &ecs.UpdateTaskProtectionOutput{
				ProtectedTasks: []types.ProtectedTask{
					{
						TaskArn:           aws.String("test"),
						ProtectionEnabled: true,
					},
				},
			},
		},
		{
			name: "should return a successful response with protection disabled",
			fields: fields{
				ECSClient: &SuccessfulTestClient{},
			},
			args: args{
				ctx: context.Background(),
				input: &UpdateTaskProtectionInput{
					Metadata: &MetadataBody{
						TaskARN: "test",
					},
					Protect: false,
				},
			},
			want: &ecs.UpdateTaskProtectionOutput{
				ProtectedTasks: []types.ProtectedTask{
					{
						TaskArn:           aws.String("test"),
						ProtectionEnabled: false,
					},
				},
			},
		},
		{
			name: "should return a response with protection failures",
			fields: fields{
				ECSClient: &FailureTestClient{},
			},
			args: args{
				ctx: context.Background(),
				input: &UpdateTaskProtectionInput{
					Metadata: &MetadataBody{
						TaskARN: "test",
					},
				},
			},
			want: &ecs.UpdateTaskProtectionOutput{
				Failures: []types.Failure{
					{
						Arn:    aws.String("test"),
						Reason: aws.String("failed"),
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				ECSClient:                tt.fields.ECSClient,
				MetadataEndpointOverride: tt.fields.MetadataEndpointOverride,
			}
			got, err := c.UpdateTaskProtection(tt.args.ctx, tt.args.input)
			if assert.NoError(t, err) {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestClient_GetTaskArn(t *testing.T) {
	tests := []struct {
		name    string
		want    *MetadataBody
		wantErr bool
	}{
		{
			name: "should return a MetadataBody with test cluster and task ARN",
			want: &MetadataBody{
				Cluster: "test_cluster",
				TaskARN: "test_arn",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, `{"Cluster": "test_cluster", "TaskARN": "test_arn"}`)
			}))
			defer ts.Close()

			c := &Client{
				MetadataEndpointOverride: ts.URL,
			}
			got, err := c.GetTaskArn(context.Background())
			if assert.NoError(t, err) {
				assert.Equal(t, &MetadataBody{
					Cluster: "test_cluster",
					TaskARN: "test_arn",
				}, got)
			}
		})
	}
}

func TestNewClient(t *testing.T) {
	type args struct {
		ecsClient ECSClient
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "should test ECS client compatibility with ECSClient interface",
			args: args{
				ecsClient: ecs.New(ecs.Options{}),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = NewClient(tt.args.ecsClient)
		})
	}
}
