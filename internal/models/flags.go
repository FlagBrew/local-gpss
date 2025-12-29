package models

type Flags struct {
	Mode string `short:"m" long:"mode" env:"MODE" required:"true" description:"The mode Local GPSS is running in: cli/docker" default:"cli"`
}
