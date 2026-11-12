package transcribe

import "context"

func ProviderNameOf(ctx context.Context, p Provider) string {
	if p == nil {
		return ""
	}
	if d, ok := p.(*Dynamic); ok {
		cfg := d.Resolve(ctx)
		if cfg.Provider == "" {
			return "assemblyai"
		}
		return cfg.Provider
	}
	return p.Name()
}
