package transcribe

import "context"

// ProviderNameOf sağlayıcının görünen adını döndürür. Dynamic sağlayıcıda
// gerçek (yapılandırılmış) ad çözülür; iş kayıtlarına "dynamic" yazılmasını
// önler.
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
