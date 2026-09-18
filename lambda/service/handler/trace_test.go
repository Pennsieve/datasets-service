package handler

import (
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/pennsieve/datasets-service/api/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTraceIDFromInboundHeader(t *testing.T) {
	for name, tt := range map[string]struct {
		headers        map[string]string
		expectedID     string
		expectedSource string
	}{
		"x-request-id": {
			headers:        map[string]string{"x-request-id": "req-abc"},
			expectedID:     "req-abc",
			expectedSource: "x-request-id",
		},
		"x-correlation-id": {
			headers:        map[string]string{"x-correlation-id": "corr-abc"},
			expectedID:     "corr-abc",
			expectedSource: "x-correlation-id",
		},
		"traceparent": {
			headers:        map[string]string{"traceparent": "00-trace-span-01"},
			expectedID:     "00-trace-span-01",
			expectedSource: "traceparent",
		},
		"x-amzn-trace-id": {
			headers:        map[string]string{"x-amzn-trace-id": "Root=1-abc"},
			expectedID:     "Root=1-abc",
			expectedSource: "x-amzn-trace-id",
		},
		"header lookup is case-insensitive": {
			headers:        map[string]string{"X-Request-Id": "req-mixed-case"},
			expectedID:     "req-mixed-case",
			expectedSource: "x-request-id",
		},
		"value is trimmed": {
			headers:        map[string]string{"x-request-id": "  req-padded  "},
			expectedID:     "req-padded",
			expectedSource: "x-request-id",
		},
		"earlier header wins over later": {
			headers: map[string]string{
				"x-request-id":    "req-wins",
				"x-amzn-trace-id": "Root=1-loses",
			},
			expectedID:     "req-wins",
			expectedSource: "x-request-id",
		},
	} {
		t.Run(name, func(t *testing.T) {
			request := &events.APIGatewayV2HTTPRequest{Headers: tt.headers}
			id, source := traceID(request)
			assert.Equal(t, tt.expectedID, id)
			assert.Equal(t, tt.expectedSource, source)
		})
	}
}

func TestTraceIDGeneratedWhenAbsent(t *testing.T) {
	for name, headers := range map[string]map[string]string{
		"no headers":            nil,
		"unrelated headers":     {"content-type": "application/json"},
		"empty correlation id":  {"x-request-id": ""},
		"whitespace-only value": {"x-request-id": "   "},
	} {
		t.Run(name, func(t *testing.T) {
			request := &events.APIGatewayV2HTTPRequest{Headers: headers}
			id, source := traceID(request)
			assert.Equal(t, logging.TraceIdGenerated, source)
			assert.NotEmpty(t, id)
		})
	}
}

// Each invocation without an inbound correlation id must get its own trace id.
func TestGeneratedTraceIDsAreUnique(t *testing.T) {
	request := &events.APIGatewayV2HTTPRequest{}
	first, _ := traceID(request)
	second, _ := traceID(request)
	assert.NotEqual(t, first, second)
}

func TestTraceIDNilRequest(t *testing.T) {
	id, source := traceID(nil)
	assert.Equal(t, logging.TraceIdGenerated, source)
	assert.NotEmpty(t, id)
}

// The AWS per-hop id and the internal trace id are distinct values recorded
// under distinct fields: a fresh AWS id per hop cannot correlate across hops,
// so conflating them would lose the cross-service link.
func TestHandlerKeepsAwsIdAndTraceIdDistinct(t *testing.T) {
	request := newTestRequest("GET", "/trashcan", "apigw-request-id", map[string]string{}, "")
	request.Headers = map[string]string{"x-request-id": "inbound-trace-id"}

	h := NewHandler(request, nil)
	require.Equal(t, "apigw-request-id", h.requestID)
	require.Equal(t, "inbound-trace-id", h.traceID)
	require.NotEqual(t, h.requestID, h.traceID)
}
