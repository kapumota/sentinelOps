//go:build !grpcvalidator

package security

func buildGRPCValidator(_ Options) InputValidator {
	return nil
}
