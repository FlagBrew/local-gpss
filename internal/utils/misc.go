package utils

import (
	"context"
	"errors"
	"math/rand"
	"strconv"

	"github.com/FlagBrew/local-gpss/internal/database/ent"
	"github.com/FlagBrew/local-gpss/internal/database/ent/bundle"
	"github.com/FlagBrew/local-gpss/internal/database/ent/pokemon"
)

func GenerateDownloadCode(ctx context.Context, kind string) (string, error) {
	db := ent.FromContext(ctx)
	if db == nil {
		return "", errors.New("db is nil")
	}

	var downloadCode string
	for {
		downloadCode = strconv.Itoa(rand.Intn(9) + 1)
		for i := 0; i < 9; i++ {
			downloadCode += strconv.Itoa(rand.Intn(10))
		}

		var exists bool
		var err error

		if kind == "pokemon" {
			exists, err = db.Pokemon.Query().Where(pokemon.DownloadCode(downloadCode)).Exist(ctx)
			if err != nil {
				return "", err
			}
		} else {
			exists, err = db.Bundle.Query().Where(bundle.DownloadCode(downloadCode)).Exist(ctx)
			if err != nil {
				return "", err
			}
		}

		if !exists {
			break
		}
	}
	return downloadCode, nil
}
