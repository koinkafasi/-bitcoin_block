package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// ---- Ligler (puan durumu + fikstür) ----
//
// Puan durumu: ESPN (ücretsiz, anahtarsız). Fikstür: TheSportsDB eventsround
// (ücretsiz anahtar hafta başına tam maç listesi + logolar verir).

// League desteklenen bir futbol ligi.
type League struct {
	Key    string // superlig
	Name   string // Süper Lig
	TSDB   string // TheSportsDB lig id (fikstür)
	ESPN   string // ESPN lig kodu (puan durumu)
	Rounds int    // toplam hafta sayısı
}

// Leagues desteklenen ligler (sıra seçicide görünür).
var Leagues = []League{
	{"superlig", "Süper Lig", "4339", "tur.1", 34},
	{"bundesliga", "Bundesliga", "4331", "ger.1", 34},
	{"premierlig", "Premier Lig", "4328", "eng.1", 38},
	{"laliga", "La Liga", "4335", "esp.1", 38},
	{"seriea", "Serie A", "4332", "ita.1", 38},
	{"ligue1", "Ligue 1", "4334", "fra.1", 34},
}

// LeagueByKey anahtardan lig döndürür (yoksa Süper Lig).
func LeagueByKey(k string) League {
	for _, l := range Leagues {
		if l.Key == k {
			return l
		}
	}
	return Leagues[0]
}

func standingsKey(lg League) string { return "L:" + lg.Key }

// StandingsOfLeague bir ligin puan durumunu döndürür. Süper Lig için admin'in
// elle girdiği tablo (varsa) önceliklidir; yoksa önbellek/ESPN.
func StandingsOfLeague(ctx context.Context, pool DB, lg League) StandingsData {
	if lg.Key == "superlig" {
		if rows, ok := ManualStandings(ctx, pool); ok {
			return StandingsData{Rows: rows, League: lg.Name, Updated: time.Now()}
		}
	}
	key := standingsKey(lg)
	if b, err := CacheGet(ctx, pool, "puan", key); err == nil {
		var sd StandingsData
		if json.Unmarshal(b, &sd) == nil && len(sd.Rows) > 0 {
			return sd
		}
	}
	rows := fetchESPNStandings(lg.ESPN, 12*time.Second)
	sd := StandingsData{Rows: rows, League: lg.Name, Updated: time.Now()}
	if len(rows) > 0 {
		if b, err := json.Marshal(sd); err == nil {
			_ = CacheSet(ctx, pool, "puan", key, b)
		}
	}
	return sd
}

// ---- Fikstür (hafta bazlı, takım logolu) — TheSportsDB eventsround ----

// Fixture tek maç.
type Fixture struct {
	Date      string `json:"date"` // 2026-10-16
	Time      string `json:"time"` // 20:00
	Home      string `json:"home"`
	Away      string `json:"away"`
	HomeBadge string `json:"homeBadge"`
	AwayBadge string `json:"awayBadge"`
	HomeScore *int   `json:"homeScore"`
	AwayScore *int   `json:"awayScore"`
	Played    bool   `json:"played"`
}

// FixtureRound bir haftanın maçları.
type FixtureRound struct {
	Round   int       `json:"round"`
	Season  string    `json:"season"`
	Matches []Fixture `json:"matches"`
	Updated time.Time `json:"updated"`
}

// tsRoundEvent eventsround.php satırı.
type tsRoundEvent struct {
	Timestamp string `json:"strTimestamp"`
	Home      string `json:"strHomeTeam"`
	Away      string `json:"strAwayTeam"`
	HomeBadge string `json:"strHomeTeamBadge"`
	AwayBadge string `json:"strAwayTeamBadge"`
	HomeScore string `json:"intHomeScore"`
	AwayScore string `json:"intAwayScore"`
	Round     string `json:"intRound"`
	Season    string `json:"strSeason"`
}

func tsRound(key, league string, round int, season string, timeout time.Duration) []tsRoundEvent {
	path := fmt.Sprintf("eventsround.php?id=%s&r=%d&s=%s", league, round, season)
	raw, err := httpGetJSONUA(fmt.Sprintf(tsAPI, key, path), timeout)
	if err != nil {
		return nil
	}
	var resp struct {
		Events []tsRoundEvent `json:"events"`
	}
	if json.Unmarshal(raw, &resp) != nil {
		return nil
	}
	return resp.Events
}

func fixturesRoundKey(lg League, season string, round int) string {
	return lg.Key + ":" + season + ":" + strconv.Itoa(round)
}

func toFixtures(evs []tsRoundEvent, round int, season string) FixtureRound {
	fr := FixtureRound{Round: round, Season: season, Updated: time.Now()}
	for _, e := range evs {
		date, tm := e.Timestamp, ""
		if len(date) >= 16 {
			tm = date[11:16]
			date = date[:10]
		}
		f := Fixture{Date: date, Time: tm, Home: e.Home, Away: e.Away, HomeBadge: e.HomeBadge, AwayBadge: e.AwayBadge}
		if e.HomeScore != "" {
			if v, err := strconv.Atoi(e.HomeScore); err == nil {
				f.HomeScore = &v
				f.Played = true
			}
		}
		if e.AwayScore != "" {
			if v, err := strconv.Atoi(e.AwayScore); err == nil {
				f.AwayScore = &v
			}
		}
		fr.Matches = append(fr.Matches, f)
	}
	return fr
}

// FiksturRoundOf bir ligin/haftanın maçlarını döndürür; önbellekte yoksa çeker.
func FiksturRoundOf(ctx context.Context, pool DB, lg League, season string, round int) FixtureRound {
	if round < 1 {
		round = 1
	}
	key := fixturesRoundKey(lg, season, round)
	if b, err := CacheGet(ctx, pool, "fikstur-r", key); err == nil {
		var fr FixtureRound
		if json.Unmarshal(b, &fr) == nil && len(fr.Matches) > 0 {
			return fr
		}
	}
	evs := tsRound(tsAPIKey(ctx, pool), lg.TSDB, round, season, 12*time.Second)
	fr := toFixtures(evs, round, season)
	if len(fr.Matches) > 0 {
		if b, err := json.Marshal(fr); err == nil {
			_ = CacheSet(ctx, pool, "fikstur-r", key, b)
		}
	}
	return fr
}

// ---- meta (güncel sezon/hafta) ----

// FiksturMeta fikstür sayfası için sezon/hafta bilgisi.
type FiksturMeta struct {
	Season  string    `json:"season"` // 2026-2027
	Round   int       `json:"round"`  // güncel hafta
	Rounds  int       `json:"rounds"` // toplam hafta
	Updated time.Time `json:"updated"`
}

func tsAPIKey(ctx context.Context, pool DB) string {
	key := "123"
	if b, err := GetSetting(ctx, pool, "servis", "fikstur_key"); err == nil {
		if s := trimJSONString(string(b)); s != "" {
			key = s
		}
	}
	return key
}

// fetchLeagueMeta bir ligin güncel sezon/haftasını TheSportsDB'den okur ve
// başarılıysa önbelleğe yazar (başarısızsa yazmaz; sonraki istekte yeniden denenir).
func fetchLeagueMeta(ctx context.Context, pool DB, lg League) FiksturMeta {
	key := tsAPIKey(ctx, pool)
	season := seasonOf(time.Now())
	round := 0

	next := tsEvents(key, "eventsnextleague.php?id="+lg.TSDB, 10*time.Second)
	if len(next) > 0 {
		if next[0].Season != "" {
			season = next[0].Season
		}
		round = atoiSafe(next[0].Round)
	}
	if round <= 0 {
		past := tsEvents(key, "eventspastleague.php?id="+lg.TSDB, 10*time.Second)
		if len(past) > 0 {
			if past[0].Season != "" {
				season = past[0].Season
			}
			round = atoiSafe(past[0].Round)
		}
	}

	m := FiksturMeta{Season: season, Round: round, Rounds: lg.Rounds, Updated: time.Now()}
	if round > 0 {
		if b, err := json.Marshal(m); err == nil {
			_ = CacheSet(ctx, pool, "fikstur", "m:"+lg.Key, b)
		}
	} else {
		m.Round = 1
	}
	return m
}

// LeagueMetaOf önbellekten okur; yoksa çeker.
func LeagueMetaOf(ctx context.Context, pool DB, lg League) FiksturMeta {
	if b, err := CacheGet(ctx, pool, "fikstur", "m:"+lg.Key); err == nil {
		var m FiksturMeta
		if json.Unmarshal(b, &m) == nil && m.Round > 0 {
			return m
		}
	}
	return fetchLeagueMeta(ctx, pool, lg)
}

// RefreshFixtures tüm liglerin güncel haftasını ve etrafındaki maçları
// önbelleğe alır (aralarında bekleyerek; ücretsiz API rate-limit'ine uyar).
func RefreshFixtures(ctx context.Context, pool DB) error {
	key := tsAPIKey(ctx, pool)
	for _, lg := range Leagues {
		meta := fetchLeagueMeta(ctx, pool, lg)
		for r := meta.Round - 1; r <= meta.Round+2; r++ {
			if r < 1 {
				continue
			}
			evs := tsRound(key, lg.TSDB, r, meta.Season, 12*time.Second)
			if len(evs) == 0 {
				continue
			}
			fr := toFixtures(evs, r, meta.Season)
			// Fikstür rozetlerini yerelleştir (indirme yalnız ilk seferde).
			mirrorFixtureBadges(ctx, &fr)
			if b, err := json.Marshal(fr); err == nil {
				_ = CacheSet(ctx, pool, "fikstur-r", fixturesRoundKey(lg, meta.Season, r), b)
			}
		}
		time.Sleep(3 * time.Second) // TheSportsDB ücretsiz key: nazik aralık
	}
	return nil
}
