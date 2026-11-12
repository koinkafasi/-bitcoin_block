package repo

import (
	"context"
	"time"
)

// goCities 81 il (plaka, ad, slug, enlem, boylam) — yaklaşık koordinatlar.
var goCities = []struct {
	plaka int
	name  string
	slug  string
	lat   float64
	lon   float64
}{
	{1, "Adana", "adana", 37.00, 35.32}, {2, "Adıyaman", "adiyaman", 37.76, 38.28},
	{3, "Afyonkarahisar", "afyonkarahisar", 38.76, 30.54}, {4, "Ağrı", "agri", 39.72, 43.05},
	{5, "Amasya", "amasya", 40.65, 35.83}, {6, "Ankara", "ankara", 39.93, 32.86},
	{7, "Antalya", "antalya", 36.89, 30.71}, {8, "Artvin", "artvin", 41.18, 41.82},
	{9, "Aydın", "aydin", 37.85, 27.84}, {10, "Balıkesir", "balikesir", 39.65, 27.89},
	{11, "Bilecik", "bilecik", 40.15, 29.99}, {12, "Bingöl", "bingol", 38.88, 40.49},
	{13, "Bitlis", "bitlis", 38.40, 42.11}, {14, "Bolu", "bolu", 40.74, 31.61},
	{15, "Burdur", "burdur", 37.72, 30.29}, {16, "Bursa", "bursa", 40.19, 29.06},
	{17, "Çanakkale", "canakkale", 40.15, 26.41}, {18, "Çankırı", "cankiri", 40.60, 33.62},
	{19, "Çorum", "corum", 40.55, 34.95}, {20, "Denizli", "denizli", 37.78, 29.09},
	{21, "Diyarbakır", "diyarbakir", 37.91, 40.24}, {22, "Edirne", "edirne", 41.68, 26.56},
	{23, "Elazığ", "elazig", 38.68, 39.22}, {24, "Erzincan", "erzincan", 39.75, 39.49},
	{25, "Erzurum", "erzurum", 39.90, 41.27}, {26, "Eskişehir", "eskisehir", 39.78, 30.52},
	{27, "Gaziantep", "gaziantep", 37.07, 37.38}, {28, "Giresun", "giresun", 40.91, 38.39},
	{29, "Gümüşhane", "gumushane", 40.46, 39.48}, {30, "Hakkari", "hakkari", 37.58, 43.74},
	{31, "Hatay", "hatay", 36.20, 36.16}, {32, "Isparta", "isparta", 37.76, 30.55},
	{33, "Mersin", "mersin", 36.80, 34.63}, {34, "İstanbul", "istanbul", 41.01, 28.98},
	{35, "İzmir", "izmir", 38.42, 27.14}, {36, "Kars", "kars", 40.60, 43.10},
	{37, "Kastamonu", "kastamonu", 41.38, 33.78}, {38, "Kayseri", "kayseri", 38.72, 35.48},
	{39, "Kırklareli", "kirklareli", 41.73, 27.22}, {40, "Kırşehir", "kirsehir", 39.15, 34.16},
	{41, "Kocaeli", "kocaeli", 40.77, 29.94}, {42, "Konya", "konya", 37.87, 32.48},
	{43, "Kütahya", "kutahya", 39.42, 29.98}, {44, "Malatya", "malatya", 38.35, 38.31},
	{45, "Manisa", "manisa", 38.61, 27.43}, {46, "Kahramanmaraş", "kahramanmaras", 37.58, 36.93},
	{47, "Mardin", "mardin", 37.31, 40.74}, {48, "Muğla", "mugla", 37.22, 28.36},
	{49, "Muş", "mus", 38.75, 41.49}, {50, "Nevşehir", "nevsehir", 38.63, 34.71},
	{51, "Niğde", "nigde", 37.97, 34.68}, {52, "Ordu", "ordu", 40.98, 37.88},
	{53, "Rize", "rize", 41.02, 40.52}, {54, "Sakarya", "sakarya", 40.69, 30.44},
	{55, "Samsun", "samsun", 41.29, 36.33}, {56, "Siirt", "siirt", 37.93, 41.94},
	{57, "Sinop", "sinop", 42.03, 35.15}, {58, "Sivas", "sivas", 39.75, 37.02},
	{59, "Tekirdağ", "tekirdag", 40.98, 27.51}, {60, "Tokat", "tokat", 40.31, 36.55},
	{61, "Trabzon", "trabzon", 41.00, 39.72}, {62, "Tunceli", "tunceli", 39.10, 39.55},
	{63, "Şanlıurfa", "sanliurfa", 37.16, 38.79}, {64, "Uşak", "usak", 38.68, 29.40},
	{65, "Van", "van", 38.49, 43.38}, {66, "Yozgat", "yozgat", 39.82, 34.81},
	{67, "Zonguldak", "zonguldak", 41.45, 31.79}, {68, "Aksaray", "aksaray", 38.37, 34.03},
	{69, "Bayburt", "bayburt", 40.26, 40.22}, {70, "Karaman", "karaman", 37.18, 33.22},
	{71, "Kırıkkale", "kirikkale", 39.85, 33.51}, {72, "Batman", "batman", 37.88, 41.13},
	{73, "Şırnak", "sirnak", 37.52, 42.46}, {74, "Bartın", "bartin", 41.64, 32.34},
	{75, "Ardahan", "ardahan", 41.11, 42.70}, {76, "Iğdır", "igdir", 39.92, 44.04},
	{77, "Yalova", "yalova", 40.65, 29.27}, {78, "Karabük", "karabuk", 41.21, 32.63},
	{79, "Kilis", "kilis", 36.72, 37.12}, {80, "Osmaniye", "osmaniye", 37.07, 36.25},
	{81, "Düzce", "duzce", 40.84, 31.16},
}

// City şehir kaydıdır.
type City struct {
	Plaka int
	Name  string
	Slug  string
	Lat   float64
	Lon   float64
}

// EnsureCities kataloğu yoksa doldurur (idempotant).
func EnsureCities(ctx context.Context, pool DB) error {
	for i, c := range goCities {
		_, err := pool.Exec(ctx, `
			INSERT INTO cities (plaka, name, slug, lat, lon, sort) VALUES ($1,$2,$3,$4,$5,$6)
			ON CONFLICT (plaka) DO UPDATE SET name=EXCLUDED.name, lat=EXCLUDED.lat, lon=EXCLUDED.lon`,
			c.plaka, c.name, c.slug, c.lat, c.lon, i+1)
		if err != nil {
			return err
		}
	}
	return nil
}

// ListCities 81 ili döndürür.
func ListCities(ctx context.Context, pool DB) ([]City, error) {
	rows, err := pool.Query(ctx, `SELECT plaka, name, slug, lat, lon FROM cities ORDER BY sort, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []City
	for rows.Next() {
		var c City
		if err := rows.Scan(&c.Plaka, &c.Name, &c.Slug, &c.Lat, &c.Lon); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// GetCitySlug slug'la şehir döndürür.
func GetCitySlug(ctx context.Context, pool DB, slug string) (City, error) {
	var c City
	err := pool.QueryRow(ctx, `SELECT plaka, name, slug, lat, lon FROM cities WHERE slug = $1`, slug).
		Scan(&c.Plaka, &c.Name, &c.Slug, &c.Lat, &c.Lon)
	return c, err
}

// GetCityByPlaka plaka ile şehir döndürür.
func GetCityByPlaka(ctx context.Context, pool DB, plaka int) (City, error) {
	var c City
	err := pool.QueryRow(ctx, `SELECT plaka, name, slug, lat, lon FROM cities WHERE plaka = $1`, plaka).
		Scan(&c.Plaka, &c.Name, &c.Slug, &c.Lat, &c.Lon)
	return c, err
}

// CacheRow servis önbellek satırıdır.
type CacheRow struct {
	Kind      string
	Key       string
	Data      []byte
	UpdatedAt time.Time
}

// CacheSet servis verisini önbelleğe yazar.
func CacheSet(ctx context.Context, pool DB, kind, key string, data []byte) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO services_cache (kind, key, data) VALUES ($1,$2,$3::jsonb)
		ON CONFLICT (kind, key) DO UPDATE SET data=EXCLUDED.data, updated_at=now()`,
		kind, key, data)
	return err
}

// CacheGet servis verisini okur.
func CacheGet(ctx context.Context, pool DB, kind, key string) ([]byte, error) {
	var b []byte
	err := pool.QueryRow(ctx, `SELECT data FROM services_cache WHERE kind=$1 AND key=$2`, kind, key).Scan(&b)
	return b, err
}

// CacheList kind bazında tüm satırları döndürür.
func CacheList(ctx context.Context, pool DB, kind string) ([]CacheRow, error) {
	rows, err := pool.Query(ctx, `SELECT kind, key, data, updated_at FROM services_cache WHERE kind=$1`, kind)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CacheRow
	for rows.Next() {
		var r CacheRow
		if err := rows.Scan(&r.Kind, &r.Key, &r.Data, &r.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func PurgeServiceCache(ctx context.Context, pool DB, days int) error {
	if days <= 0 {
		days = 7
	}
	_, err := pool.Exec(ctx,
		`DELETE FROM services_cache WHERE updated_at < now() - ($1::int || ' days')::interval`, days)
	return err
}
