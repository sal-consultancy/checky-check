package main

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	ssmtypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

func normalizeIdentityType(identity Identity) string {
	identityType := strings.ToLower(strings.TrimSpace(identity.Type))
	if identityType == "" {
		return "ssh"
	}
	return identityType
}

func isAWSSSMIdentity(identity Identity) bool {
	return normalizeIdentityType(identity) == "aws_ssm"
}

func runSSMCommand(identity Identity, host string, target Target, command string, timeout time.Duration) (string, error) {
	if strings.TrimSpace(target.InstanceID) == "" && !targetHasTag(target) {
		return "", &CheckExecutionError{
			Type:    "ssm_config_error",
			Message: "aws_ssm host requires target.instance_id or target.tag.key/value",
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	loadOptions := []func(*awsconfig.LoadOptions) error{}
	if region := strings.TrimSpace(identity.Region); region != "" {
		loadOptions = append(loadOptions, awsconfig.WithRegion(region))
	}
	if profile := strings.TrimSpace(identity.Profile); profile != "" {
		loadOptions = append(loadOptions, awsconfig.WithSharedConfigProfile(profile))
	}

	awsConfig, err := awsconfig.LoadDefaultConfig(ctx, loadOptions...)
	if err != nil {
		return "", &CheckExecutionError{
			Type:    "ssm_config_error",
			Message: fmt.Sprintf("failed to load AWS configuration for SSM: %v", err),
		}
	}
	if strings.TrimSpace(awsConfig.Region) == "" {
		return "", &CheckExecutionError{
			Type:    "ssm_config_error",
			Message: "aws_ssm identity requires a region in config or AWS_REGION/AWS_DEFAULT_REGION",
		}
	}

	instanceID, err := resolveSSMInstanceID(ctx, awsConfig, host, target)
	if err != nil {
		return "", err
	}

	client := ssm.NewFromConfig(awsConfig)
	sendOutput, err := client.SendCommand(ctx, &ssm.SendCommandInput{
		DocumentName: stringPtr("AWS-RunShellScript"),
		InstanceIds:  []string{instanceID},
		Parameters: map[string][]string{
			"commands":         []string{command},
			"executionTimeout": []string{ssmExecutionTimeout(timeout)},
		},
	})
	if err != nil {
		return "", classifySSMContextError(ctx, "ssm_send_command_error", fmt.Sprintf("failed to send SSM command to %s: %v", instanceID, err))
	}
	if sendOutput.Command == nil || sendOutput.Command.CommandId == nil || strings.TrimSpace(*sendOutput.Command.CommandId) == "" {
		return "", &CheckExecutionError{
			Type:    "ssm_send_command_error",
			Message: fmt.Sprintf("SSM did not return a command id for %s", instanceID),
		}
	}

	commandID := *sendOutput.Command.CommandId
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		invocation, err := client.GetCommandInvocation(ctx, &ssm.GetCommandInvocationInput{
			CommandId:  &commandID,
			InstanceId: &instanceID,
		})
		if err != nil {
			if ctx.Err() != nil {
				return "", classifySSMContextError(ctx, "ssm_timeout", fmt.Sprintf("SSM command %s on %s timed out after %v", commandID, instanceID, timeout))
			}

			var notReady *ssmtypes.InvocationDoesNotExist
			if errors.As(err, &notReady) {
				select {
				case <-ctx.Done():
					return "", classifySSMContextError(ctx, "ssm_timeout", fmt.Sprintf("SSM command %s on %s timed out after %v", commandID, instanceID, timeout))
				case <-ticker.C:
					continue
				}
			}

			return "", &CheckExecutionError{
				Type:    "ssm_invocation_error",
				Message: fmt.Sprintf("failed to read SSM command invocation %s for %s: %v", commandID, instanceID, err),
			}
		}

		output := combineSSMOutput(invocation.StandardOutputContent, invocation.StandardErrorContent)
		switch invocation.Status {
		case ssmtypes.CommandInvocationStatusSuccess:
			return output, nil
		case ssmtypes.CommandInvocationStatusFailed,
			ssmtypes.CommandInvocationStatusCancelled,
			ssmtypes.CommandInvocationStatusCancelling,
			ssmtypes.CommandInvocationStatusTimedOut:
			return output, &CheckExecutionError{
				Type:    "ssm_command_error",
				Message: fmt.Sprintf("SSM command %s on %s ended with status %s and response code %d", commandID, instanceID, invocation.Status, invocation.ResponseCode),
				Output:  output,
			}
		case ssmtypes.CommandInvocationStatusPending,
			ssmtypes.CommandInvocationStatusInProgress,
			ssmtypes.CommandInvocationStatusDelayed:
			select {
			case <-ctx.Done():
				return output, classifySSMContextError(ctx, "ssm_timeout", fmt.Sprintf("SSM command %s on %s timed out after %v", commandID, instanceID, timeout))
			case <-ticker.C:
				continue
			}
		default:
			return output, &CheckExecutionError{
				Type:    "ssm_unexpected_status",
				Message: fmt.Sprintf("SSM command %s on %s returned unexpected status %s", commandID, instanceID, invocation.Status),
				Output:  output,
			}
		}
	}
}

func classifySSMContextError(ctx context.Context, fallbackType string, message string) *CheckExecutionError {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return &CheckExecutionError{
			Type:    "ssm_timeout",
			Message: message,
		}
	}

	return &CheckExecutionError{
		Type:    fallbackType,
		Message: message,
	}
}

func combineSSMOutput(stdout *string, stderr *string) string {
	var parts []string
	if stdout != nil && strings.TrimSpace(*stdout) != "" {
		parts = append(parts, strings.TrimSpace(*stdout))
	}
	if stderr != nil && strings.TrimSpace(*stderr) != "" {
		parts = append(parts, strings.TrimSpace(*stderr))
	}
	return strings.Join(parts, "\n")
}

func stringPtr(value string) *string {
	return &value
}

func ssmExecutionTimeout(timeout time.Duration) string {
	timeoutSeconds := int(timeout.Seconds())
	if timeoutSeconds < 1 {
		timeoutSeconds = 1
	}
	return strconv.Itoa(timeoutSeconds)
}

func targetHasTag(target Target) bool {
	return strings.TrimSpace(target.Tag.Key) != "" || strings.TrimSpace(target.Tag.Value) != ""
}

func targetTagComplete(target Target, host string) bool {
	return strings.TrimSpace(target.Tag.Key) != "" && resolveTargetTagValue(target, host) != ""
}

func resolveTargetTagValue(target Target, host string) string {
	if value := strings.TrimSpace(target.Tag.Value); value != "" {
		return value
	}
	return strings.TrimSpace(host)
}

func resolveSSMInstanceID(ctx context.Context, awsConfig aws.Config, host string, target Target) (string, error) {
	if instanceID := strings.TrimSpace(target.InstanceID); instanceID != "" {
		return instanceID, nil
	}

	if !targetTagComplete(target, host) {
		return "", &CheckExecutionError{
			Type:    "ssm_config_error",
			Message: "aws_ssm tag target requires target.tag.key; target.tag.value defaults to the host name when omitted",
		}
	}

	tagKey := strings.TrimSpace(target.Tag.Key)
	tagValue := resolveTargetTagValue(target, host)
	client := ec2.NewFromConfig(awsConfig)
	output, err := client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		Filters: []ec2types.Filter{
			{
				Name:   stringPtr(fmt.Sprintf("tag:%s", tagKey)),
				Values: []string{tagValue},
			},
			{
				Name:   stringPtr("instance-state-name"),
				Values: []string{"pending", "running", "stopping", "stopped"},
			},
		},
	})
	if err != nil {
		return "", classifySSMContextError(ctx, "ssm_config_error", fmt.Sprintf("failed to resolve EC2 instance by tag %s=%s: %v", tagKey, tagValue, err))
	}

	var instanceIDs []string
	for _, reservation := range output.Reservations {
		for _, instance := range reservation.Instances {
			if instance.InstanceId != nil && strings.TrimSpace(*instance.InstanceId) != "" {
				instanceIDs = append(instanceIDs, strings.TrimSpace(*instance.InstanceId))
			}
		}
	}

	if len(instanceIDs) == 0 {
		return "", &CheckExecutionError{
			Type:    "ssm_config_error",
			Message: fmt.Sprintf("no EC2 instance found for tag %s=%s", tagKey, tagValue),
		}
	}
	if len(instanceIDs) > 1 {
		return "", &CheckExecutionError{
			Type:    "ssm_config_error",
			Message: fmt.Sprintf("tag %s=%s matched multiple EC2 instances: %s", tagKey, tagValue, strings.Join(instanceIDs, ", ")),
		}
	}

	return instanceIDs[0], nil
}
