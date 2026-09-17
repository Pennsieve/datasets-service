package logging

// Structured log attribute keys. Use these constants rather than free text in
// the message, so CloudWatch Insights / DataDog can filter on a field, e.g.
// `filter datasetNodeId = "N:dataset:..."`.
//
// One canonical spelling per field: camelCase, matching the majority convention
// already in this repo (e.g. the handler's pre-existing requestID field becomes
// requestId here so that every id field reads the same way).
const (
	// ErrorKey carries the error value itself, as slog.Any(ErrorKey, err),
	// rather than being interpolated into the message text. Keeping the
	// message static and the error in its own field is what makes a class of
	// failures groupable in DataDog.
	ErrorKey = "error"

	// LevelKey is used only by this package when reporting its own
	// configuration (the requested/effective log level).
	LevelKey = "level"

	// Per-hop AWS-provided identifiers. These are deliberately NOT merged into
	// a single generic "requestId" field: AWS mints a fresh id at every hop, so
	// each names a different kind of id and only correlates within its own hop.
	// See TraceIdKey for the cross-hop identifier.
	ApiGatewayRequestIdKey = "apiGatewayRequestId"
	AwsRequestIdKey        = "awsRequestId"

	// TraceIdKey is this service's internal correlation id for one logical
	// operation. Unlike the AWS per-hop ids above, it is taken from an inbound
	// correlation header when the caller supplied one, so it can survive hops.
	TraceIdKey = "traceId"

	// TraceIdSourceKey records where the trace id came from: the name of the
	// inbound header it was read from, or "generated".
	TraceIdSourceKey = "traceIdSource"

	// Request context.
	MethodKey      = "method"
	PathKey        = "path"
	QueryParamsKey = "queryParams"
	RequestBodyKey = "requestBody"
	ClaimsKey      = "claims"
	StatusKey      = "status"

	// Domain identifiers.
	OrgIdKey         = "orgId"
	UserIdKey        = "userId"
	DatasetIdKey     = "datasetId"
	DatasetNodeIdKey = "datasetNodeId"
	PackageNodeIdKey = "packageNodeId"

	// Storage / messaging.
	S3BucketKey = "s3Bucket"
	S3KeyKey    = "s3Key"
	SnsTopicKey = "snsTopic"

	// Diagnostics.
	QueryKey  = "query"
	LimitKey  = "limit"
	OffsetKey = "offset"
)

// TraceIdGenerated is the TraceIdSourceKey value used when no inbound
// correlation header was present and this service minted the trace id itself.
const TraceIdGenerated = "generated"
