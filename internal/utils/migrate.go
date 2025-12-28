package utils

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"sync"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/FlagBrew/local-gpss/internal/database/ent"
	"github.com/FlagBrew/local-gpss/internal/database/ent/bundle"
	"github.com/FlagBrew/local-gpss/internal/models"
	"github.com/apex/log"
	"golang.org/x/sync/errgroup"
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

type pokemonBinding struct {
	OldId int
	NewId int
}

type bundleBinding struct {
	OldId int
	NewId int
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

	pkmnMap := sync.Map{}
	pkmnBindingMap := sync.Map{}
	eg, _ := errgroup.WithContext(ctx)
	eg.SetLimit(30)
	for i, oldPkmn := range oldPokemons {
		fmt.Printf("%d/%d\r", i, len(oldPokemons))

		eg.Go(func() error {
			// Call GpssConsole to get the latest info
			result, err := ExecGpssConsole[models.GpssLegalityCheckReply](ctx, models.GpssConsoleArgs{
				Mode:       "legality",
				Pokemon:    oldPkmn.Base64,
				Generation: oldPkmn.Generation,
			})
			if err != nil {
				logger.WithError(err).Error("failed to fetch pokemon")
				return nil
			}

			oldPokemons[i].Legal = result.Legal
			return nil
		})

	}

	eg.Wait()

	for _, oldPkmn := range oldPokemons {
		newPkmn, err := tx.Pokemon.Create().
			SetUploadDatetime(oldPkmn.UploadDateTime).
			SetDownloadCode(oldPkmn.DownloadCode).
			SetDownloadCount(oldPkmn.DownloadCount).
			SetGeneration(oldPkmn.Generation).
			SetLegal(oldPkmn.Legal).
			SetBase64(oldPkmn.Base64).Save(ctx)

		if err != nil {
			logger.WithError(err).Error("failed to save pokemon")
			tx.Rollback()
			return
		}
		pkmnMap.Store(newPkmn.ID, newPkmn)
		pkmnBindingMap.Store(oldPkmn.ID, pokemonBinding{
			OldId: oldPkmn.ID,
			NewId: newPkmn.ID,
		})
	}

	bundleMap := map[int]*ent.Bundle{}
	bundleBindingMap := map[int]bundleBinding{}
	for _, ob := range oldBundles {
		newBundle, err := tx.Bundle.Create().
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

		bundleMap[newBundle.ID] = newBundle
		bundleBindingMap[ob.ID] = bundleBinding{
			OldId: ob.ID,
			NewId: newBundle.ID,
		}
	}

	genMap := map[int][]string{}
	for _, ob := range oldPokemonBundles {
		// get the bindings
		loadedVal, ok := pkmnBindingMap.Load(ob.PokemonID)
		if !ok {
			logger.Error("failed to fetch pokemon-bundle")
			tx.Rollback()
			return
		}

		oldP, ok := loadedVal.(pokemonBinding)
		if !ok {
			logger.Error("failed to cast pokemon-bundle")
			tx.Rollback()
			return
		}

		oldB, ok := bundleBindingMap[ob.BundleID]
		if !ok {
			logger.Error("failed to fetch bundle")
			tx.Rollback()
			return
		}

		b, ok := bundleMap[oldB.NewId]
		if !ok {
			logger.Error("failed to fetch bundle from map")
			tx.Rollback()
			return
		}

		p, ok := pkmnMap.Load(oldP.NewId)
		if !ok {
			logger.Error("failed to fetch pokemon from map")
			tx.Rollback()
			return
		}

		p2, ok := p.(*ent.Pokemon)
		if !ok {
			logger.Error("failed to fetch pokemon from map")
			tx.Rollback()
		}

		if p2.Legal != b.Legal {
			_, err = tx.Bundle.UpdateOne(b).SetLegal(false).Save(ctx)
			if err != nil {
				logger.WithError(err).Error("failed to update legal status in bundle")
				tx.Rollback()
				return
			}
		}

		_, err = tx.Pokemon.UpdateOne(p2).AddBundleIDs(b.ID).Save(ctx)
		if err != nil {
			logger.WithError(err).Error("failed to save pokemon")
			tx.Rollback()
			return
		}

		genMap[b.ID] = append(genMap[b.ID], p2.Generation)
	}

	for k, g := range genMap {
		slices.Sort(g)

		_, err = tx.Bundle.Update().SetMinGen(g[0]).SetMaxGen(g[len(g)-1]).Where(bundle.ID(k)).Save(ctx)
		if err != nil {
			logger.WithError(err).Error("failed to correct bundle info")
			tx.Rollback()
		}
	}

	// Commit the data now
	err = tx.Commit()
	if err != nil {
		logger.WithError(err).Error("failed to commit transaction")
		return
	}

	// Now we need to reset the primary key IDs, we do this by getting the latest IDs
	db.Pokemon.Query()

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
