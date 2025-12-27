package gpss

type gpssPokemonListResponse struct {
	Page    int           `json:"page"`
	Pages   int           `json:"pages"`
	Total   int           `json:"total"`
	Pokemon []gpssPokemon `json:"pokemon"`
}

type gpssPokemon struct {
	Legal      bool   `json:"legal"`
	Base64     string `json:"base_64"`
	Code       string `json:"code"`
	Generation string `json:"generation"`
}

type gpssBundleListResponse struct {
	Page    int          `json:"page"`
	Pages   int          `json:"pages"`
	Total   int          `json:"total"`
	Bundles []gpssBundle `json:"bundles"`
}

type gpssBundlePokemon struct {
	Legal      bool   `json:"legality"`
	Base64     string `json:"base_64"`
	Generation string `json:"generation"`
}

type gpssBundle struct {
	Pokemons      []gpssBundlePokemon `json:"pokemons"`
	DownloadCodes []string            `json:"download_codes"`
	DownloadCode  string              `json:"download_code"`
	Patreon       bool                `json:"patreon"`
	MinGen        string              `json:"min_gen"`
	MaxGen        string              `json:"max_gen"`
	Count         int                 `json:"count"`
	Legal         bool                `json:"legality"`
}
