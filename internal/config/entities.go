package config

type InitialConds struct {
	X0  float64 `json:"x0"`
	V0  float64 `json:"v0"`
	Tau float64 `json:"tau"`
	T0  float64 `json:"t0"`
	T   float64 `json:"t"`
	K   float64 `json:"k"`
	D   float64 `json:"d"`
	M   float64 `json:"m"`
}

type InitialCondsCoupled struct {
	X1 float64 `json:"x1"`
	V1 float64 `json:"v1"`
	K1 float64 `json:"k1"`
	D1 float64 `json:"d1"`
	M1 float64 `json:"m1"`
	X2 float64 `json:"x2"`
	V2 float64 `json:"v2"`
	K2 float64 `json:"k2"`
	D2 float64 `json:"d2"`
	M2 float64 `json:"m2"`

	K12 float64 `json:"k12"`
	D12 float64 `json:"d12"`
}
