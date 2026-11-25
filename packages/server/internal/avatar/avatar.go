package avatar

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/huntersxy/xqecz/server/internal/cache"
	_ "golang.org/x/image/webp"
)

type Subject byte

const (
	User Subject = iota + 1
	Guest

	Route = "/avatars/"

	tokenDataSize = 9
	signatureSize = 16
	maxImageBytes = 256 << 10
	maxImageEdge  = 256
)

var (
	qqMailPattern    = regexp.MustCompile(`^(\d{5,11})@qq\.com$`)
	errImageMissing  = errors.New("avatar image unavailable")
	errImageRejected = errors.New("avatar image invalid")
)

type Proxy struct {
	origin string
	key    []byte
	client *http.Client
	cache  *cache.Client
}

func New(origin, signingKey string, redis *cache.Client) *Proxy {
	key, err := hex.DecodeString(signingKey)
	if err != nil || len(key) != 32 {
		return nil
	}
	if origin != "" {
		address, err := url.Parse(origin)
		if err != nil || (address.Scheme != "https" && address.Scheme != "http") ||
			address.Host == "" || address.User != nil || address.RawQuery != "" || address.Fragment != "" ||
			(address.Path != "" && address.Path != "/") {
			return nil
		}
		origin = strings.TrimRight(origin, "/")
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxConnsPerHost = 8
	transport.MaxIdleConnsPerHost = 8
	return &Proxy{
		origin: origin,
		key:    key,
		client: &http.Client{
			Transport: transport,
			Timeout:   5 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		cache: redis,
	}
}

func (p *Proxy) URL(subject Subject, id uint64, email *string) string {
	if p == nil || id == 0 || email == nil || strings.TrimSpace(*email) == "" {
		return ""
	}
	var token [tokenDataSize + signatureSize]byte
	token[0] = byte(subject)
	binary.BigEndian.PutUint64(token[1:tokenDataSize], id)
	copy(token[tokenDataSize:], p.sign("avatar-ref-v1", token[:tokenDataSize]))
	return p.origin + "/api" + Route + base64.RawURLEncoding.EncodeToString(token[:])
}

func (p *Proxy) Verify(token string) (Subject, uint64, bool) {
	if p == nil || len(token) != base64.RawURLEncoding.EncodedLen(tokenDataSize+signatureSize) {
		return 0, 0, false
	}
	data, err := base64.RawURLEncoding.Strict().DecodeString(token)
	if err != nil || len(data) != tokenDataSize+signatureSize ||
		!hmac.Equal(data[tokenDataSize:], p.sign("avatar-ref-v1", data[:tokenDataSize])[:signatureSize]) {
		return 0, 0, false
	}
	subject, id := Subject(data[0]), binary.BigEndian.Uint64(data[1:tokenDataSize])
	return subject, id, id != 0 && (subject == User || subject == Guest)
}

func (p *Proxy) sign(purpose string, data []byte) []byte {
	mac := hmac.New(sha256.New, p.key)
	mac.Write([]byte(purpose))
	mac.Write(data)
	return mac.Sum(nil)
}

func (p *Proxy) Image(ctx context.Context, email string) ([]byte, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return nil, errImageMissing
	}
	upstream := providerURL(email)
	key := "avatar_image:v1:" + hex.EncodeToString(p.sign("avatar-image-v1", []byte(email)))
	return cache.GetOrSetJSON(ctx, p.cache, key, 2*time.Hour, func() ([]byte, error) {
		image, err := p.fetchImage(ctx, upstream)
		if p.cache == nil {
			return image, err
		}
		lastKnown := key + ":last"
		if err == nil {
			_ = p.cache.SetJSON(ctx, lastKnown, image, 30*24*time.Hour)
			return image, nil
		}
		if hit, _ := p.cache.GetJSON(ctx, lastKnown, &image); hit {
			return image, nil
		}
		return nil, err
	})
}

func providerURL(email string) string {
	if match := qqMailPattern.FindStringSubmatch(email); match != nil {
		return "https://q.qlogo.cn/headimg_dl?dst_uin=" + match[1] + "&spec=100"
	}
	sum := md5.Sum([]byte(email))
	return "https://www.gravatar.com/avatar/" + hex.EncodeToString(sum[:]) + "?d=identicon&s=80"
}

func (p *Proxy) fetchImage(ctx context.Context, upstream string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, upstream, nil)
	if err != nil {
		return nil, errImageMissing
	}
	req.Header.Set("Accept", "image/png, image/jpeg, image/webp, image/gif")
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, errImageMissing
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK || resp.ContentLength > maxImageBytes {
		return nil, errImageMissing
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxImageBytes+1))
	if err != nil || len(body) > maxImageBytes {
		return nil, errImageRejected
	}
	size, format, err := image.DecodeConfig(bytes.NewReader(body))
	if err != nil || size.Width < 1 || size.Height < 1 || size.Width > maxImageEdge || size.Height > maxImageEdge {
		return nil, errImageRejected
	}
	switch format {
	case "png", "jpeg", "webp", "gif":
	default:
		return nil, errImageRejected
	}
	decoded, _, err := image.Decode(bytes.NewReader(body))
	if err != nil {
		return nil, errImageRejected
	}
	var output bytes.Buffer
	if err := png.Encode(&output, decoded); err != nil {
		return nil, errImageRejected
	}
	return output.Bytes(), nil
}
