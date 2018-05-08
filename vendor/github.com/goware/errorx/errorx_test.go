package errorx_test

import (
	"errors"
	"testing"

	"github.com/c2h5oh/errorx"
)

func TestErrorVerbosity(t *testing.T) {
	e := errorx.New(10, "error message", "error details", "error hint")

	errorx.SetVerbosity(errorx.Info)
	err := e.Error()
	expected := "error 10: error message"
	if err != expected {
		t.Errorf("Expected %s, got '%s'", expected, err)
	}

	errorx.SetVerbosity(errorx.Verbose)
	err = e.Error()
	expected = "error 10: error message | error details"
	if err != expected {
		t.Errorf("Expected %s, got '%s'", expected, err)
	}

	errorx.SetVerbosity(errorx.Debug)
	err = e.Error()
	expected = "errorx_test.go:28: error 10: error message | error details; error hint"
	if err != expected {
		t.Errorf("Expected %s, got '%s'", expected, err)
	}

	errorx.SetVerbosity(errorx.Trace)
	err = e.Error()
	expected = "errorx_test.go:35: error 10: error message | error details; error hint\nerrorx_test.go:35 github.com/c2h5oh/errorx_test.TestErrorVerbosity\ntesting.go:447 testing.tRunner\nasm_amd64.s:2232 runtime.goexit"
	if err != expected {
		t.Errorf("Expected %s, got '%s'", expected, err)
	}
}

func TestJsonVerbosity(t *testing.T) {
	e := errorx.New(12, "error message", "error details", "error hint")

	errorx.SetVerbosity(errorx.Info)
	err, _ := e.Json()
	expected := `{"error_code":12,"error_message":"error message"}`
	if string(err) != expected {
		t.Errorf(`Expected '%s', got '%s'`, expected, string(err))
	}

	errorx.SetVerbosity(errorx.Verbose)
	err, _ = e.Json()
	expected = `{"error_code":12,"error_message":"error message","error_details":["error details"]}`
	if string(err) != expected {
		t.Errorf(`Expected '%s', got '%s'`, expected, string(err))
	}

	errorx.SetVerbosity(errorx.Debug)
	err, _ = e.Json()
	expected = `{"error_code":12,"error_message":"error message","error_details":["error details","error hint"],"stack":[{"file":"errorx_test.go","line":60,"function":"github.com/c2h5oh/errorx_test.TestJsonVerbosity"}]}`
	if string(err) != expected {
		t.Errorf(`Expected '%s', got '%s'`, expected, string(err))
	}
}

func TestErrorCode(t *testing.T) {
	e := errorx.New(14, "error message", "error details", "error hint")

	if e.ErrorCode() != 14 {
		t.Errorf(`Invalide error code - expected 14, got %d`, e.ErrorCode())
	}
}

func TestErrorEmbedding(t *testing.T) {
	wrappableErrorx := errorx.New(200, "wrapped error message", "wrapped error details", "wrapped error hint")
	wrappableError := errors.New("wrapped regular error")
	e1 := errorx.New(10, "error message", "error details", "error hint")
	e1.Wrap(wrappableErrorx)
	e2 := errorx.New(11, "error message", "error details", "error hint")
	e2.Wrap(wrappableError)

	errorx.SetVerbosity(errorx.Info)
	expected := "error 10: error message"
	if e1.Error() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, e1.Error())
	}
	expected = "error 11: error message"
	if e2.Error() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, e2.Error())
	}

	errorx.SetVerbosity(errorx.Verbose)
	expected = "error 10: error message | error details\ncause: error 200: wrapped error message | wrapped error details"
	if e1.Error() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, e1.Error())
	}
	expected = "error 11: error message | error details\ncause: wrapped regular error"
	if e2.Error() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, e2.Error())
	}

	errorx.SetVerbosity(errorx.Debug)
	err := e1.Error()
	expected = "errorx_test.go:104: error 10: error message | error details; error hint\ncause: error 200: wrapped error message | wrapped error details; wrapped error hint"
	if err != expected {
		t.Errorf("Expected '%s', got '%s'", expected, err)
	}

	err = e2.Error()
	expected = "errorx_test.go:110: error 11: error message | error details; error hint\ncause: wrapped regular error"
	if err != expected {
		t.Errorf("Expected '%s', got '%s'", expected, err)
	}

	errorx.SetVerbosity(errorx.Trace)
	err = e1.Error()
	expected = "errorx_test.go:117: error 10: error message | error details; error hint\ncause: error 200: wrapped error message | wrapped error details; wrapped error hint\nerrorx_test.go:117 github.com/c2h5oh/errorx_test.TestErrorEmbedding\ntesting.go:447 testing.tRunner\nasm_amd64.s:2232 runtime.goexit"
	if err != expected {
		t.Errorf("Expected '%s', got '%s'", expected, err)
	}

	err = e2.Error()
	expected = "errorx_test.go:123: error 11: error message | error details; error hint\ncause: wrapped regular error\nerrorx_test.go:123 github.com/c2h5oh/errorx_test.TestErrorEmbedding\ntesting.go:447 testing.tRunner\nasm_amd64.s:2232 runtime.goexit"
	if err != expected {
		t.Errorf("Expected '%s', got '%s'", expected, err)
	}
}

func TestJsonErrorEmbedding(t *testing.T) {
	wrappableErrorx := errorx.New(200, "wrapped error message", "wrapped error details", "wrapped error hint")
	wrappableError := errors.New("wrapped regular error")
	e1 := errorx.New(10, "error message", "error details", "error hint")
	e1.Wrap(wrappableErrorx)
	e2 := errorx.New(11, "error message", "error details", "error hint")
	e2.Wrap(wrappableError)

	errorx.SetVerbosity(errorx.Info)
	e, _ := e1.Json()
	expected := `{"error_code":10,"error_message":"error message"}`
	if string(e) != expected {
		t.Errorf("Expected '%s', got '%s'", expected, string(e))
	}
	e, _ = e2.Json()
	expected = `{"error_code":11,"error_message":"error message"}`
	if string(e) != expected {
		t.Errorf("Expected '%s', got '%s'", expected, string(e))
	}

	errorx.SetVerbosity(errorx.Verbose)
	e, _ = e1.Json()
	expected = `{"error_code":10,"error_message":"error message","error_details":["error details"]}`
	if string(e) != expected {
		t.Errorf("Expected '%s', got '%s'", expected, string(e))
	}
	e, _ = e2.Json()
	expected = `{"error_code":11,"error_message":"error message","error_details":["error details"]}`
	if string(e) != expected {
		t.Errorf("Expected '%s', got '%s'", expected, string(e))
	}

	errorx.SetVerbosity(errorx.Debug)
	e, _ = e1.Json()
	expected = `{"error_code":10,"error_message":"error message","error_details":["error details","error hint"],"cause":{"error_code":200,"error_message":"wrapped error message","error_details":["wrapped error details","wrapped error hint"]},"stack":[{"file":"errorx_test.go","line":163,"function":"github.com/c2h5oh/errorx_test.TestJsonErrorEmbedding"}]}`
	if string(e) != expected {
		t.Errorf("Expected '%s', got '%s'", expected, string(e))
	}
	e, _ = e2.Json()
	expected = `{"error_code":11,"error_message":"error message","error_details":["error details","error hint"],"cause":{"error_message":"wrapped regular error"},"stack":[{"file":"errorx_test.go","line":168,"function":"github.com/c2h5oh/errorx_test.TestJsonErrorEmbedding"}]}`
	if string(e) != expected {
		t.Errorf("Expected '%s', got '%s'", expected, string(e))
	}

	errorx.SetVerbosity(errorx.Trace)
	e, _ = e1.Json()
	expected = `{"error_code":10,"error_message":"error message","error_details":["error details","error hint"],"cause":{"error_code":200,"error_message":"wrapped error message","error_details":["wrapped error details","wrapped error hint"]},"stack":[{"file":"errorx_test.go","line":175,"function":"github.com/c2h5oh/errorx_test.TestJsonErrorEmbedding"},{"file":"testing.go","line":447,"function":"testing.tRunner"},{"file":"asm_amd64.s","line":2232,"function":"runtime.goexit"}]}`
	if string(e) != expected {
		t.Errorf("Expected '%s', got '%s'", expected, string(e))
	}
	e, _ = e2.Json()
	expected = `{"error_code":11,"error_message":"error message","error_details":["error details","error hint"],"cause":{"error_message":"wrapped regular error"},"stack":[{"file":"errorx_test.go","line":180,"function":"github.com/c2h5oh/errorx_test.TestJsonErrorEmbedding"},{"file":"testing.go","line":447,"function":"testing.tRunner"},{"file":"asm_amd64.s","line":2232,"function":"runtime.goexit"}]}`
	if string(e) != expected {
		t.Errorf("Expected '%s', got '%s'", expected, string(e))
	}
}
