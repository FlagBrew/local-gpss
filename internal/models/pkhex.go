package models

type GpssConsoleArgs struct {
	Mode       string
	Pokemon    string
	Generation string
	Version    string
}

type GpssErrorReply struct {
	Error string `json:"error"`
}

type GpssLegalityCheckReply struct {
	Legal  bool     `json:"legal"`
	Report []string `json:"report"`
}

type GpssAutoLegalityReply struct {
	Legal   bool     `json:"legal"`
	Success bool     `json:"success"`
	Ran     bool     `json:"ran"`
	Report  []string `json:"report"`
	Pokemon *string  `json:"pokemon"`
}
