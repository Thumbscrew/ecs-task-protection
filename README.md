# ecs-task-protection
[![Go](https://github.com/Thumbscrew/ecs-task-protection/actions/workflows/go.yml/badge.svg)](https://github.com/Thumbscrew/ecs-task-protection/actions/workflows/go.yml)[![API Reference](https://img.shields.io/badge/api-reference-blue.svg)](https://pkg.go.dev/github.com/Thumbscrew/ecs-task-protection/v2)

Small Go module to enable/disable protection for an ECS task.

## Example usage

```go
// create ECS client
ecsClient := ecs.New(ecs.Options{})

// create task protection client
protClient := ecstp.NewClient(ecsClient)

// enable protection
output, err := protClient.UpdateTaskProtection(context.Background(), &ecstp.UpdateTaskProtectionInput{
    Protect: true,
})

// disable protection
out, err := protClient.UpdateTaskProtection(context.Background(), &ecstp.UpdateTaskProtectionInput{
    Protect: false,
})

// enable protection with 60 minute expiry
out, err := protClient.UpdateTaskProtection(context.Background(), &ecstp.UpdateTaskProtectionInput{
    Protect: true,
    ExpiresInMinutes: aws.Int32(60),
})

// get Cluster and TaskARN metadata
body, err := protClient.GetTaskArn(context.Background())

// enable protection with provided metadata
out, err := protClient.UpdateTaskProtection(context.Background(), &ecstp.UpdateTaskProtectionInput{
    Metadata: &ecstp.MetadataBody{
        Cluster: "test-cluster",
        TaskARN: "arn:aws:ecs:eu-west-2:123456789012:task/example/taskid",
    },
    Protect: true,
})
```
