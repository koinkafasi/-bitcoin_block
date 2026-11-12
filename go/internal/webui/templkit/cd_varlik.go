package templkit

var VarlikSurum = "1"

// cdVarlik sürüm parametresini üretir (boşsa "1").
func cdVarlik() string {
	if VarlikSurum == "" {
		return "1"
	}
	return VarlikSurum
}
