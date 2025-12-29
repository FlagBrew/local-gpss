package gpss

type listRequest struct {
	Generations   []string `json:"generations" form:"generations"`
	LegalOnly     bool     `json:"legal_only" form:"legal_only"`
	SortDirection bool     `json:"sort_direction" form:"sort_direction"`
	SortField     string   `json:"sort_field" form:"sort_field"`
}
