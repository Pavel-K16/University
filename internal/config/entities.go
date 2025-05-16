package config

type TimeConds struct {
	Tau float64 `json:"tau"`
	T0  float64 `json:"t0"`
	T   float64 `json:"t"`
}

type Body struct {
	X0 float64 `json:"x0"`
	V0 float64 `json:"v0"`
	K  float64 `json:"k"`
	D  float64 `json:"d"`
	M  float64 `json:"m"`
}

type ConnParams struct {
	K12 float64 `json:"k12"`
	D12 float64 `json:"d12"`
}

type BodiesConds struct {
	Body1      Body       `json:"body1"`
	Body2      Body       `json:"body2"`
	ConnParams ConnParams `json:"connParams"`
	IsCoupled  bool       `json:"isCoupled"`
}
