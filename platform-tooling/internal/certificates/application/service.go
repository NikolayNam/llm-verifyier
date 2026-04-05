package application

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/NikolayNam/collabsphere/platform-tooling/internal/certificates"
	"github.com/NikolayNam/collabsphere/platform-tooling/internal/certificates/hilbert"
)

const (
	CodeSchemaInvalid  = "CERTIFICATE_SCHEMA_INVALID"
	CodeParseFailed    = "CERTIFICATE_PARSE_FAILED"
	CodeKernelRejected = "CERTIFICATE_KERNEL_REJECTED"
	CodeIOFailed       = "CERTIFICATE_IO_FAILED"
	CodeInternal       = "CERTIFICATE_INTERNAL"
)

type Service struct{}

type VerificationResult struct {
	Certificate *certificates.Certificate
	Report      *hilbert.Report
}

type VerificationError struct {
	Code    string
	Message string
	Cause   error
}

func (e *VerificationError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	if e.Cause != nil {
		return e.Cause.Error()
	}
	return e.Code
}

func (e *VerificationError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func NewService() *Service {
	return &Service{}
}

func (s *Service) VerifyFile(ctx context.Context, path string) (*VerificationResult, error) {
	if err := ctxErr(ctx); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, newError(CodeIOFailed, "Read certificate file failed", err)
	}

	return s.VerifyJSON(ctx, data)
}

func (s *Service) VerifyJSON(ctx context.Context, raw []byte) (*VerificationResult, error) {
	if err := ctxErr(ctx); err != nil {
		return nil, err
	}

	cert, err := certificates.DecodeJSON(raw)
	if err != nil {
		return nil, mapLowerLayerError(err)
	}

	report, err := hilbert.VerifyCertificate(cert)
	if err != nil {
		return nil, mapLowerLayerError(err)
	}

	return &VerificationResult{
		Certificate: cert,
		Report:      report,
	}, nil
}

func ctxErr(ctx context.Context) error {
	if ctx == nil || ctx.Err() == nil {
		return nil
	}
	return newError(CodeInternal, "Certificate verification failed", ctx.Err())
}

func mapLowerLayerError(err error) error {
	switch certificates.ClassOf(err) {
	case certificates.ErrorClassSchema:
		return newError(CodeSchemaInvalid, wrapLowerLayerMessage("Certificate schema is invalid", err), err)
	case certificates.ErrorClassParse:
		return newError(CodeParseFailed, wrapLowerLayerMessage("Certificate parsing failed", err), err)
	case certificates.ErrorClassKernel:
		return newError(CodeKernelRejected, wrapLowerLayerMessage("Certificate kernel rejected the certificate", err), err)
	default:
		return newError(CodeInternal, wrapLowerLayerMessage("Certificate verification failed", err), err)
	}
}

func wrapLowerLayerMessage(prefix string, err error) string {
	if err == nil {
		return prefix
	}
	detail := strings.TrimSpace(err.Error())
	if detail == "" {
		return prefix
	}
	return prefix + ": " + detail
}

func ExplainResult(result *VerificationResult) (string, error) {
	if result == nil {
		return "", fmt.Errorf("verification result is nil")
	}
	if result.Report == nil {
		return "", fmt.Errorf("verification report is nil")
	}
	return result.Report.String(), nil
}

func CodeOf(err error) string {
	if verificationErr, ok := errors.AsType[*VerificationError](err); ok && verificationErr != nil {
		return verificationErr.Code
	}
	return CodeInternal
}

func AsVerificationError(err error) (*VerificationError, bool) {
	var verificationErr *VerificationError
	if !errors.As(err, &verificationErr) || verificationErr == nil {
		return nil, false
	}
	return verificationErr, true
}

func newError(code, message string, cause error) error {
	return &VerificationError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}
