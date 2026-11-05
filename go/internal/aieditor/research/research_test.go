package research

import (
	"context"
	"net"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBlockedIP(t *testing.T) {
	for _, s := range []string{"127.0.0.1", "10.0.0.1", "192.168.1.1", "169.254.169.254", "100.64.0.1", "198.18.0.1", "::1", "fc00::1"} {
		assert.True(t, blockedIP(net.ParseIP(s)), s)
	}
	for _, s := range []string{"8.8.8.8", "1.1.1.1", "2606:4700:4700::1111"} {
		assert.False(t, blockedIP(net.ParseIP(s)), s)
	}
}

func TestFetchTextBlocksLoopback(t *testing.T) {
	_, err := FetchText(context.Background(), "http://127.0.0.1:80/")
	require.Error(t, err)
	_, err = FetchText(context.Background(), "file:///etc/passwd")
	require.Error(t, err)
}

type fakeProvider struct {
	results map[string][]Result
	err     error
}

func (f *fakeProvider) Name() string { return "fake" }
func (f *fakeProvider) Search(_ context.Context, q string, _ int) ([]Result, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.results[q], nil
}

func TestCollectDeduplicates(t *testing.T) {
	p := &fakeProvider{results: map[string][]Result{
		"a": {{Title: "Bir", URL: "https://example.com/1"}, {Title: "İki", URL: "https://example.com/2"}},
		"b": {{Title: "İki tekrar", URL: "https://example.com/2"}, {Title: "Kötü", URL: "javascript:alert(1)"}},
	}}
	got, err := Collect(context.Background(), p, []string{"a", "b"}, 8, false)
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "example.com", got[0].Publisher)
}

func TestBraveLive(t *testing.T) {
	if os.Getenv("BRAVE_LIVE") != "1" {
		t.Skip("BRAVE_LIVE=1 değil")
	}
	p := FromEnv(nil)
	items, err := p.Search(context.Background(), "theblock.com.tr", 3)
	require.NoError(t, err)
	require.NotEmpty(t, items)
	assert.NotEmpty(t, items[0].URL)
}
