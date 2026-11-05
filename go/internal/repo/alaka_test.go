package repo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Kripto alaka kapısı.
//
// ÖLÇÜM 1 (5 Kasım 2026): ana sayfa manşeti "Federal Reserve Board issues
// enforcement actions with former employee of Northstar Bank…" idi.
//
// ÖLÇÜM 2: ilk sürümde `exchange`/`token`/`custody` de terim listesindeydi;
// SEC'in 25 bülteninin 25'i kapıyı geçti (filtre işlevsiz). Aşağıdaki
// "kripto dışı" listesi GERÇEK SEC/Fed/ECB başlıklarıdır.
func TestKriptoAlakali(t *testing.T) {
	kripto := []string{
		"SEC Issues “Innovation Exemption” to Facilitate the Trading of Tokenized NMS Stock",
		"SEC approves spot bitcoin ETF applications",
		"Federal Reserve issues guidance on stablecoin reserves",
		"ECB publishes digital euro progress report",
		"Tokenization pilot enters next phase",
		"Blockchain-based settlement platform launched",
		"Virtual asset service providers must register",
		"Kripto para düzenlemesi Resmî Gazete'de",
		"Dijital varlık hizmet sağlayıcıları için yeni kurallar",
		"Crypto exchange registration deadline extended",
	}
	for _, s := range kripto {
		assert.True(t, kriptoAlakali(s), "kripto sayılmalı: %q", s)
	}

	// GERÇEK başlıklar (SEC/Fed/ECB, 5 Kasım 2026) — hiçbiri kripto değil.
	alakasiz := []string{
		"Federal Reserve Board issues enforcement actions with former employee of Northstar Bank, former employee of American Express Travel Related Services Company",
		"Federal Reserve Board announces termination of enforcement action with SNB Bancshares and Bank of Eufaula",
		"Monetary policy decisions",
		"Isabel Schnabel: Navigating stormy waters: reflections on monetary policy",
		"ECB wage tracker suggests slightly higher wage growth in first three quarters of 2027",
		"Survey on the Access to Finance of Enterprises: lending conditions tightened",
		"SEC Grants Exemptive Relief from Certain Inline XBRL Filing or Submission Requirements",
		"SEC Proposes Rescission of Shareholder Proposal Rule and Reforms to Proxy Solicitation Process",
		"SEC Censures OTC Link LLC for Repeated Compliance Failures Related to Regulation SCI",
		"SEC Charges South Florida Resident and His Company for Alleged Investment Scheme Defrauding Law",
		"SEC Publishes Updated Market Statistics, Highlighting Increase in IPOs and Proceeds Raised",
		"SEC Charges Registered Investment Adviser Zoe Financial for Failure to Disclose Conflict of Int",
		"SEC Proposes Amendments to Expand Responsible Retailization of Private Markets",
		"SEC’s Division of Examinations Announces New Exam Handbook",
	}
	for _, s := range alakasiz {
		assert.False(t, kriptoAlakali(s), "kripto dışı sayılmalı: %q", s)
	}
}

// TestKaynakAlakaGerekli: kapı yalnız resmî kurumlarda.
func TestKaynakAlakaGerekli(t *testing.T) {
	// aggregator=true → kripto medyası (CoinDesk, Decrypt…): kapı YOK.
	assert.False(t, kaynakAlakaGerekli(true))
	// aggregator=false → resmî kurum (SEC, Fed, ECB): kapı VAR.
	assert.True(t, kaynakAlakaGerekli(false))
}
