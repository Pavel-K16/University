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
