package handler

import (
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/google/uuid"
	"github.com/pennsieve/datasets-service/api/logging"
)

// inboundTraceHeaders are the request headers checked, in order, for a
// correlation id supplied by the caller.
//
// Why not just use the AWS request id: AWS mints a *fresh* id at every hop
// (API Gateway RequestContext.RequestID here, a new SQS MessageId at the next
// queue, a new AwsRequestID on a direct invoke). Each is only meaningful within
// its own hop, so none of them can tie together one logical operation that
// crosses service boundaries. Those per-hop ids are still logged, under their
// own distinctly-named fields, because they are what correlates a log line back
// to a specific AWS-side record. The trace id below is separate: it prefers an
// id the caller already established, so it can survive the hop.
//
// X-Amzn-Trace-Id is included because API Gateway/X-Ray populate it, and it is
// the one header likely to already be present across Pennsieve services today.
var inboundTraceHeaders = []string{
	"x-request-id",
	"x-correlation-id",
	"traceparent",
	"x-amzn-trace-id",
}

// traceID returns a correlation id for one logical operation plus the name of
// the header it came from (or logging.TraceIdGenerated when this service minted
// it). Header lookup is case-insensitive: API Gateway v2 lowercases header keys,
// but tests and direct invocations may not.
func traceID(request *events.APIGatewayV2HTTPRequest) (string, string) {
	if request != nil {
		for _, name := range inboundTraceHeaders {
			if value := lookupHeader(request.Headers, name); value != "" {
				return value, name
			}
		}
	}
	return uuid.NewString(), logging.TraceIdGenerated
}

func lookupHeader(headers map[string]string, name string) string {
	for k, v := range headers {
		if strings.EqualFold(k, name) {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
