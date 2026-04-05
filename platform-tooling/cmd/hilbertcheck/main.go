package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	certapp "github.com/NikolayNam/collabsphere/platform-tooling/internal/certificates/application"
)

const (
	exitOK            = 0
	exitSchemaError   = 1
	exitParseError    = 2
	exitKernelError   = 3
	exitInternalError = 4
	exitUsageError    = 64
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	explain := false
	paths := make([]string, 0, 1)

	for _, arg := range args {
		switch arg {
		case "--explain":
			explain = true
		case "-h", "--help":
			printUsage(stderr)
			return exitUsageError
		default:
			paths = append(paths, arg)
		}
	}

	if len(paths) != 1 {
		printUsage(stderr)
		return exitUsageError
	}

	service := certapp.NewService()
	result, err := service.VerifyFile(context.Background(), paths[0])
	if err != nil {
		return printAndClassifyError(stderr, err)
	}

	if explain {
		trace, traceErr := certapp.ExplainResult(result)
		if traceErr != nil {
			return printAndClassifyError(stderr, traceErr)
		}
		_, _ = fmt.Fprintln(stdout, trace)
	} else {
		_, _ = fmt.Fprintf(stdout, "certificate accepted: %s\n", result.Certificate.ProofID)
	}

	return exitOK
}

func printUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, "usage: hilbertcheck [--explain] <certificate.json>")
}

func printAndClassifyError(stderr io.Writer, err error) int {
	appErr, ok := certapp.AsVerificationError(err)
	if !ok || appErr == nil {
		_, _ = fmt.Fprintf(stderr, "internal_error: %v\n", err)
		return exitInternalError
	}

	prefix, exitCode := classifyVerificationCode(appErr.Code)
	message := strings.TrimSpace(appErr.Message)
	if message == "" {
		message = err.Error()
	}
	_, _ = fmt.Fprintf(stderr, "%s: %s\n", prefix, message)
	return exitCode
}

func classifyVerificationCode(code string) (string, int) {
	switch code {
	case certapp.CodeSchemaInvalid:
		return "schema_error", exitSchemaError
	case certapp.CodeParseFailed:
		return "parse_error", exitParseError
	case certapp.CodeKernelRejected:
		return "kernel_validation_error", exitKernelError
	case certapp.CodeIOFailed, certapp.CodeInternal:
		return "internal_error", exitInternalError
	default:
		return "internal_error", exitInternalError
	}
}
