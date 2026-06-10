//go:build grpcvalidator

package security

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	validatorv1 "sentinelops/gen/go/validator/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type GRPCValidator struct {
	addr    string
	timeout time.Duration
	conn    *grpc.ClientConn
	client  validatorv1.ValidatorClient
}

func buildGRPCValidator(opts Options) InputValidator {
	addr := strings.TrimSpace(opts.GRPCAddr)
	if addr == "" {
		return nil
	}
	timeout := opts.GRPCTimeout
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	return &GRPCValidator{addr: addr, timeout: timeout}
}

func (v *GRPCValidator) Validate(input string) error {
	ctx, cancel := context.WithTimeout(context.Background(), v.timeout)
	defer cancel()

	client, closeFn, err := v.clientForRequest(ctx)
	if err != nil {
		return &ExternalRuntimeError{binary: "input-guard-grpc", msg: err.Error(), err: err}
	}
	defer closeFn()

	resp, err := client.ValidateInput(ctx, &validatorv1.ValidateInputRequest{
		CorrelationId: fmt.Sprintf("grpc-%d", time.Now().UnixNano()),
		Input:         input,
		Context: &validatorv1.ValidationContext{
			InputType: "shell_command",
			Timestamp: time.Now().Unix(),
		},
	})
	if err != nil {
		return &ExternalRuntimeError{binary: "input-guard-grpc", msg: err.Error(), err: err}
	}
	if resp.GetValid() {
		return nil
	}
	reason := strings.TrimSpace(resp.GetReason())
	if reason == "" {
		reason = "entrada rechazada por validador gRPC"
	}
	return errors.New(reason)
}

func (v *GRPCValidator) clientForRequest(ctx context.Context) (validatorv1.ValidatorClient, func(), error) {
	if v.client != nil && v.conn != nil {
		return v.client, func() {}, nil
	}
	conn, err := grpc.DialContext(ctx, v.addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}
	return validatorv1.NewValidatorClient(conn), func() { _ = conn.Close() }, nil
}
