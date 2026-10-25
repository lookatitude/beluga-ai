package cloudstorage

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/lookatitude/beluga-ai/v2/config"
	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/rag/loader"
	"github.com/lookatitude/beluga-ai/v2/schema"
)

func init() {
	loader.Register("cloudstorage", func(cfg config.ProviderConfig) (loader.DocumentLoader, error) {
		return New(cfg)
	})
}

// defaultMaxBytes is the default cap on a downloaded object's size.
const defaultMaxBytes = 32 << 20

// azureAPIVersion is sent with Azure requests; bearer (Entra ID) auth needs
// x-ms-version 2017-11-09 or later.
const azureAPIVersion = "2023-11-03"

// emptyPayloadHash is the hex SHA-256 of an empty body, used for SigV4 GETs.
const emptyPayloadHash = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

var (
	// s3BucketPattern is the S3 bucket naming rule (DNS-compatible).
	s3BucketPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]{1,61}[a-z0-9]$`)
	// gcsBucketPattern is the GCS bucket naming rule.
	gcsBucketPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{1,220}[a-z0-9]$`)
	// azureAccountPattern is the Azure storage-account naming rule.
	azureAccountPattern = regexp.MustCompile(`^[a-z0-9]{3,24}$`)
	// regionPattern restricts the configured AWS region.
	regionPattern = regexp.MustCompile(`^[a-z0-9-]{1,32}$`)
)

// Loader loads files from cloud storage services.
type Loader struct {
	httpClient *http.Client
	accessKey  string
	secretKey  string
	region     string
	// sessionToken is an optional AWS session token for S3 SigV4.
	sessionToken string
	// maxBytes caps the object size; 0 means defaultMaxBytes.
	maxBytes int64
}

// Compile-time interface check.
var _ loader.DocumentLoader = (*Loader)(nil)

// New creates a new cloud storage document loader.
func New(cfg config.ProviderConfig) (*Loader, error) {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	secretKey, _ := config.GetOption[string](cfg, "secret_key")
	sessionToken, _ := config.GetOption[string](cfg, "session_token")
	region, _ := config.GetOption[string](cfg, "region")
	if region == "" {
		region = "us-east-1"
	}
	if !regionPattern.MatchString(region) {
		return nil, core.NewError("cloudstorage.new", core.ErrInvalidInput, fmt.Sprintf("invalid region %q", region), nil)
	}
	var maxBytes int64
	if m, ok := config.GetOption[float64](cfg, "max_bytes"); ok && m > 0 {
		maxBytes = int64(m)
	}

	return &Loader{
		httpClient: &http.Client{
			Timeout:       timeout,
			CheckRedirect: checkRedirect,
		},
		accessKey:    cfg.APIKey,
		secretKey:    secretKey,
		region:       region,
		sessionToken: sessionToken,
		maxBytes:     maxBytes,
	}, nil
}

// checkRedirect refuses redirects that leave the storage providers' hosts,
// so a redirect can never carry a request (or credentials) elsewhere.
func checkRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= 5 {
		return fmt.Errorf("cloudstorage: too many redirects")
	}
	if req.URL.Scheme != "https" || !isProviderHost(req.URL.Hostname()) {
		return fmt.Errorf("cloudstorage: refusing redirect to %q", req.URL.Host)
	}
	return nil
}

// isProviderHost reports whether host belongs to one of the supported
// storage providers.
func isProviderHost(host string) bool {
	return hostAllowed("s3", host, "") || hostAllowed("gcs", host, "") || hostAllowed("azure", host, "")
}

// hostAllowed reports whether host is a legitimate endpoint for provider.
// For S3 an empty region accepts any regional endpoint.
func hostAllowed(provider, host, region string) bool {
	host = strings.ToLower(host)
	switch provider {
	case "s3":
		if region != "" {
			return strings.HasSuffix(host, ".s3."+region+".amazonaws.com")
		}
		return strings.HasSuffix(host, ".amazonaws.com")
	case "gcs":
		return host == "storage.googleapis.com"
	case "azure":
		return strings.HasSuffix(host, ".blob.core.windows.net")
	default:
		return false
	}
}

// parseSource parses the storage URL and returns provider, bucket, and key.
// For Azure the first segment is the storage account and the key is
// "<container>/<blob>". Bucket and account names are validated against the
// provider's naming rules because they become part of the request host.
func parseSource(source string) (provider, bucket, key string, err error) {
	var rest string
	var pattern *regexp.Regexp
	switch {
	case strings.HasPrefix(source, "s3://"):
		provider, rest, pattern = "s3", strings.TrimPrefix(source, "s3://"), s3BucketPattern
	case strings.HasPrefix(source, "gs://"):
		provider, rest, pattern = "gcs", strings.TrimPrefix(source, "gs://"), gcsBucketPattern
	case strings.HasPrefix(source, "az://"):
		provider, rest, pattern = "azure", strings.TrimPrefix(source, "az://"), azureAccountPattern
	default:
		return "", "", "", fmt.Errorf("cloudstorage: unsupported URL scheme in %q (expected s3://, gs://, or az://)", source)
	}
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) < 2 || parts[1] == "" {
		return "", "", "", fmt.Errorf("cloudstorage: invalid %s URL %q (expected %s)", displayName[provider], source, expectedForm(provider))
	}
	if !pattern.MatchString(parts[0]) {
		return "", "", "", core.NewError("cloudstorage.load", core.ErrInvalidInput,
			fmt.Sprintf("invalid %s bucket/account name %q", provider, parts[0]), nil)
	}
	return provider, parts[0], parts[1], nil
}

// displayName is the human-readable provider name used in errors.
var displayName = map[string]string{"s3": "S3", "gcs": "GCS", "azure": "Azure"}

func expectedForm(provider string) string {
	switch provider {
	case "s3":
		return "s3://bucket/key"
	case "gcs":
		return "gs://bucket/key"
	default:
		return "az://account/container/blob"
	}
}

// buildURL constructs the HTTP URL for the storage object. The host is built
// from the validated bucket/account only; the key is carried in the path and
// escaped, so it can never change the host or add a query string.
func (l *Loader) buildURL(provider, bucket, key string) string {
	u := url.URL{Scheme: "https", Path: "/" + key}
	switch provider {
	case "s3":
		u.Host = fmt.Sprintf("%s.s3.%s.amazonaws.com", bucket, l.region)
	case "gcs":
		u.Host = "storage.googleapis.com"
		u.Path = "/" + bucket + "/" + key
	case "azure":
		u.Host = bucket + ".blob.core.windows.net"
	default:
		return ""
	}
	return u.String()
}

// Load downloads a file from cloud storage and returns it as a document.
func (l *Loader) Load(ctx context.Context, source string) ([]schema.Document, error) {
	if source == "" {
		return nil, fmt.Errorf("cloudstorage: source URL is required")
	}

	provider, bucket, key, err := parseSource(source)
	if err != nil {
		return nil, err
	}

	rawURL := l.buildURL(provider, bucket, key)
	if rawURL == "" {
		return nil, fmt.Errorf("cloudstorage: unable to build URL for %q", source)
	}
	// Defence in depth: never send a request (or credentials) to a host
	// that is not the provider's endpoint.
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme != "https" || !hostAllowed(provider, parsed.Hostname(), l.region) {
		return nil, core.NewError("cloudstorage.load", core.ErrInvalidInput,
			fmt.Sprintf("refusing to fetch %q: host is not a %s endpoint", source, provider), nil)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("cloudstorage: create request: %w", err)
	}
	if err := l.authorize(ctx, provider, req); err != nil {
		return nil, err
	}

	resp, err := l.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cloudstorage: fetch %q: %w", source, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("cloudstorage: fetch %q failed (status %d): %s", source, resp.StatusCode, string(body))
	}

	limit := l.maxBytes
	if limit <= 0 {
		limit = defaultMaxBytes
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, fmt.Errorf("cloudstorage: read %q: %w", source, err)
	}
	if int64(len(data)) > limit {
		return nil, core.NewError("cloudstorage.load", core.ErrInvalidInput,
			fmt.Sprintf("object %q exceeds the %d-byte limit", source, limit), nil)
	}

	content := string(data)
	if content == "" {
		return nil, nil
	}

	// Extract filename from key.
	filename := key
	if idx := strings.LastIndex(key, "/"); idx >= 0 {
		filename = key[idx+1:]
	}

	meta := map[string]any{
		"source":   source,
		"loader":   "cloudstorage",
		"provider": provider,
		"bucket":   bucket,
		"key":      key,
		"filename": filename,
	}

	return []schema.Document{{
		ID:       source,
		Content:  content,
		Metadata: meta,
	}}, nil
}

// authorize adds provider credentials to req. S3 requests are signed with
// SigV4 (APIKey = access key ID, Options["secret_key"], optional
// Options["session_token"]). GCS and Azure send APIKey as an OAuth bearer
// token; Azure also gets the x-ms-version header bearer auth requires.
// Without an APIKey the request is anonymous.
func (l *Loader) authorize(ctx context.Context, provider string, req *http.Request) error {
	if l.accessKey == "" {
		return nil
	}
	switch provider {
	case "s3":
		if l.secretKey == "" {
			return core.NewError("cloudstorage.load", core.ErrInvalidInput,
				"s3 requires Options[\"secret_key\"] with the access key for SigV4 signing", nil)
		}
		req.Header.Set("x-amz-content-sha256", emptyPayloadHash)
		creds := aws.Credentials{AccessKeyID: l.accessKey, SecretAccessKey: l.secretKey, SessionToken: l.sessionToken}
		if err := v4.NewSigner().SignHTTP(ctx, creds, req, emptyPayloadHash, "s3", l.region, time.Now()); err != nil {
			return fmt.Errorf("cloudstorage: sign s3 request: %w", err)
		}
	case "azure":
		req.Header.Set("x-ms-version", azureAPIVersion)
		req.Header.Set("Authorization", "Bearer "+l.accessKey)
	case "gcs":
		req.Header.Set("Authorization", "Bearer "+l.accessKey)
	}
	return nil
}
