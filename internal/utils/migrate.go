package utils

import (
	"context"
	"errors"
	"os"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/FlagBrew/local-gpss/internal/database/ent"
	"github.com/FlagBrew/local-gpss/internal/models"
	"github.com/apex/log"
)

type oldPokemon struct {
	ID             int
	UploadDateTime time.Time
	DownloadCode   string
	DownloadCount  int
	Generation     string
	Legal          bool
	Base64         string
}

type oldBundle struct {
	ID             int
	DownloadCode   string
	UploadDateTime time.Time
	DownloadCount  int
	Legal          bool
	MinGen         string
	MaxGen         string
}

type oldBundlePokemon struct {
	PokemonID int
	BundleID  int
}

func MigrateOriginalDb(ctx context.Context, cfg *models.Config) {
	logger := log.FromContext(ctx)
	db := ent.FromContext(ctx)
	if db == nil {
		logger.Error("DB missing from context")
		return
	}

	// Check if the file exists
	if _, err := os.Stat("gpss.db"); errors.Is(err, os.ErrNotExist) {
		logger.Warn("Old Database doesn't exist")
		return
	}

	oldDb, err := sql.Open("sqlite", "file:gpss.db?_pragma=foreign_keys(1)")
	if err != nil {
		logger.WithError(err).Error("failed to open old database")
		return
	}
	defer oldDb.Close()

	var oldPokemons []oldPokemon

	rows, err := oldDb.QueryContext(ctx, "SELECT * FROM pokemon")
	if err != nil {
		logger.WithError(err).Error("failed to fetch old pokemons")
		return
	}

	defer rows.Close()
	for rows.Next() {
		var pokemon oldPokemon
		err = rows.Scan(&pokemon.ID, &pokemon.UploadDateTime, &pokemon.DownloadCode, &pokemon.DownloadCount, &pokemon.Generation, &pokemon.Legal, &pokemon.Base64)
		if err != nil {
			logger.WithError(err).Error("failed to fetch pokemon")
			return
		}

		oldPokemons = append(oldPokemons, pokemon)
	}

	if err = rows.Err(); err != nil {
		logger.WithError(err).Error("failed to fetch pokemons")
		return
	}

	var oldBundles []oldBundle

	rows, err = oldDb.QueryContext(ctx, "SELECT * FROM bundle")
	if err != nil {
		logger.WithError(err).Error("failed to fetch old bundles")
		return
	}

	defer rows.Close()
	for rows.Next() {
		var bundle oldBundle
		err = rows.Scan(&bundle.ID, &bundle.DownloadCode, &bundle.UploadDateTime, &bundle.DownloadCount, &bundle.Legal, &bundle.MinGen, &bundle.MaxGen)
		if err != nil {
			logger.WithError(err).Error("failed to fetch bundle")
			return
		}

		oldBundles = append(oldBundles, bundle)
	}

	if err = rows.Err(); err != nil {
		logger.WithError(err).Error("failed to fetch bundles")
		return
	}

	var oldPokemonBundles []oldBundlePokemon

	rows, err = oldDb.QueryContext(ctx, "SELECT pokemon_id, bundle_id FROM bundle_pokemon")
	if err != nil {
		logger.WithError(err).Error("failed to fetch old bundles")
		return
	}

	defer rows.Close()
	for rows.Next() {
		var bp oldBundlePokemon
		err = rows.Scan(&bp.PokemonID, &bp.BundleID)
		if err != nil {
			logger.WithError(err).Error("failed to fetch pokemon-bundle")
			return
		}

		oldPokemonBundles = append(oldPokemonBundles, bp)
	}

	if err = rows.Err(); err != nil {
		logger.WithError(err).Error("failed to fetch pokemon-bundles")
		return
	}

	// Now that we have all the records, we need to do a bulk creation into ent go
	tx, err := db.Tx(ctx)
	if err != nil {
		logger.WithError(err).Error("failed to start transaction")
		return
	}

	pkmnMap := map[int]*ent.Pokemon{}

	for _, oldPokemon := range oldPokemons {
		newPkmn, err := tx.Pokemon.Create().
			SetID(oldPokemon.ID).
			SetUploadDatetime(oldPokemon.UploadDateTime).
			SetDownloadCode(oldPokemon.DownloadCode).
			SetDownloadCount(oldPokemon.DownloadCount).
			SetGeneration(oldPokemon.Generation).
			SetLegal(oldPokemon.Legal).
			SetBase64(oldPokemon.Base64).Save(ctx)

		if err != nil {
			logger.WithError(err).Error("failed to save pokemon")
			tx.Rollback()
			return
		}
		pkmnMap[oldPokemon.ID] = newPkmn
	}

	bundleMap := map[int]*ent.Bundle{}
	for _, ob := range oldBundles {
		newBundle, err := tx.Bundle.Create().
			SetID(ob.ID).
			SetDownloadCode(ob.DownloadCode).
			SetUploadDatetime(ob.UploadDateTime).
			SetDownloadCount(ob.DownloadCount).
			SetLegal(ob.Legal).
			SetMinGen(ob.MinGen).
			SetMaxGen(ob.MaxGen).Save(ctx)

		if err != nil {
			logger.WithError(err).Error("failed to save bundle")
			tx.Rollback()
			return
		}

		bundleMap[ob.ID] = newBundle
	}

	for _, ob := range oldPokemonBundles {
		b, ok := bundleMap[ob.BundleID]
		if !ok {
			logger.Error("failed to fetch bundle from map")
			tx.Rollback()
			return
		}

		p, ok := pkmnMap[ob.PokemonID]
		if !ok {
			logger.Error("failed to fetch pokemon from map")
			tx.Rollback()
			return
		}

		_, err = tx.Pokemon.UpdateOne(p).AddBundleIDs(b.ID).Save(ctx)
		if err != nil {
			logger.WithError(err).Error("failed to save pokemon")
			tx.Rollback()
			return
		}
	}

	// Commit the data now
	err = tx.Commit()
	if err != nil {
		logger.WithError(err).Error("failed to commit transaction")
		return
	}

	// Update the config to not migrate the db anymore
	os.ReadFile("config.json")
	cfg.Misc.MigrateOriginalDb = false
	SetConfig(ctx, cfg)

	// Remove the old DB
	err = os.Remove("gpss.db")
	if err != nil {
		logger.WithError(err).Error("failed to remove old database")
		return
	}
}
