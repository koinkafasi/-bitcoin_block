// Package live, yayındaki içerik değişikliklerini SSE istemcilerine dağıtan
// küçük, süreç içi bir yayın merkezi sağlar.
package live

import (
	"sync"
	"time"
)

// PresenceUser, aynı içeriği açan editörlerden biridir.
type PresenceUser struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// Event, yayınlanan içerik hakkında istemciye giden küçük sinyaldir. İçerik
// metni taşımaz; istemci güncel içeriği ayrı endpoint'ten çeker.
type Event struct {
	SiteID    int64          `json:"site_id,omitempty"`
	ContentID int64          `json:"content_id"`
	Version   int            `json:"version,omitempty"`
	Status    string         `json:"status,omitempty"`
	Slug      string         `json:"slug,omitempty"`
	Kind      string         `json:"kind,omitempty"` // content | removed | saved
	UserID    int64          `json:"user_id,omitempty"`
	UserName  string         `json:"user_name,omitempty"`
	Action    string         `json:"action,omitempty"`
	Users     []PresenceUser `json:"users,omitempty"`
	At        int64          `json:"at,omitempty"`
}

// Hub, içerik kimliğine göre abonelikleri tutar.
type Hub struct {
	mu   sync.RWMutex
	subs map[int64]map[chan Event]struct{}
}

func NewHub() *Hub {
	return &Hub{subs: map[int64]map[chan Event]struct{}{}}
}

// Default, tek süreçli kurulumda HTTP sunucusu ve scheduler tarafından
// paylaşılan yayın merkezidir. Çoklu instance'a geçilirse Postgres
// LISTEN/NOTIFY veya Redis pub/sub ile değiştirilmelidir.
var Default = NewHub()

// EditorDefault, yönetim panelindeki editörler arasındaki kayıt/versiyon
// sinyallerini taşır. Public SSE'den ayrıdır.
var EditorDefault = NewHub()

// Subscribe belirli bir içerik için olay kanalı açar. Dönen fonksiyon
// aboneliği kapatır ve kanalı kapatır.
func (h *Hub) Subscribe(contentID int64) (<-chan Event, func()) {
	ch := make(chan Event, 8)
	h.mu.Lock()
	if h.subs[contentID] == nil {
		h.subs[contentID] = map[chan Event]struct{}{}
	}
	h.subs[contentID][ch] = struct{}{}
	h.mu.Unlock()

	cancel := func() {
		h.mu.Lock()
		if set := h.subs[contentID]; set != nil {
			if _, ok := set[ch]; ok {
				delete(set, ch)
				close(ch)
			}
			if len(set) == 0 {
				delete(h.subs, contentID)
			}
		}
		h.mu.Unlock()
	}
	return ch, cancel
}

// Publish olayı abonelere bloklamadan dağıtır. Yavaş istemcinin kuyruğu
// dolarsa olay düşürülür; istemci yeniden bağlandığında snapshot ile
// güncel sürümü alır.
func (h *Hub) Publish(ev Event) {
	if ev.At == 0 {
		ev.At = time.Now().Unix()
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.subs[ev.ContentID] {
		select {
		case ch <- ev:
		default:
		}
	}
}

// SubscriberCount testler ve gözlemlenebilirlik için abone sayısını döner.
func (h *Hub) SubscriberCount(contentID int64) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.subs[contentID])
}
