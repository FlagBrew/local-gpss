package models

type Config struct {
	FancyScreen bool           `json:"fancy_screen"`
	Database    DatabaseConfig `json:"database"`
	HTTP        HTTPConfig     `json:"http"`
	Misc        MiscConfig     `json:"misc"`
}

type DatabaseConfig struct {
	DBType           string `json:"db_type" validate:"required,oneof=sqlite postgres mysql"`
	ConnectionString string `json:"connection_string" validate:"required"`
}

type HTTPConfig struct {
	Port          int    `json:"port" validate:"required,min=1,max=65535"`
	ListeningAddr string `json:"listening_addr" validate:"required"`
}

type MiscConfig struct {
	RecheckLegality    bool `json:"recheck_legality"`
	MigrateOriginalDb  bool `json:"migrate_original_db"`
	DownloadOriginalDb bool `json:"download_original_db"`
}
