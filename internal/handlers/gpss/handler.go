package gpss

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/FlagBrew/local-gpss/internal/database/ent"
	"github.com/FlagBrew/local-gpss/internal/database/ent/bundle"
	"github.com/FlagBrew/local-gpss/internal/database/ent/pokemon"
	"github.com/FlagBrew/local-gpss/internal/database/ent/predicate"
	"github.com/FlagBrew/local-gpss/internal/models"
	"github.com/FlagBrew/local-gpss/internal/utils"
	"github.com/apex/log"
	"github.com/go-chi/chi/v5"
	"github.com/lrstanley/chix"
)

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) Route(r chi.Router) {
	r.Post("/search/{type}", h.list)
	r.Post("/upload/pokemon", h.uploadPokemon)
	r.Post("/upload/bundle", h.uploadBundle)
	r.Get("/download/{type}/{code}", h.download)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	entityType := chi.URLParam(r, "type")
	if entityType == "" {
		chix.JSON(w, r, http.StatusBadRequest, chix.M{"error": "missing entity type"})
		return
	}

	var payload listRequest
	if chix.Error(w, r, chix.Bind(r, &payload)) {
		return
	}

	// Set the defaults
	page := 1
	limit := 30

	if entityType == "bundles" {
		// default for bundles should be 5
		limit = 5
	}

	if r.URL.Query().Get("page") != "" {
		parsedPage, err := strconv.Atoi(r.URL.Query().Get("page"))
		if err == nil && parsedPage > 0 {
			page = parsedPage
		}
	}

	if r.URL.Query().Get("amount") != "" {
		parsedAmount, err := strconv.Atoi(r.URL.Query().Get("amount"))
		if err == nil && parsedAmount < 101 && parsedAmount > 0 {
			limit = parsedAmount
		}
	}

	var gens []string
	sortField := "latest"

	if payload.SortField != "" {
		sortField = payload.SortField
	}

	orderDir := sql.OrderAsc()

	if payload.SortDirection {
		orderDir = sql.OrderDesc()
	}

	// Handle the generations array if any are provided
	if payload.Generations != nil {
		for _, gen := range payload.Generations {
			switch gen {
			case "LGPE":
				gens = append(gens, "7.1")
				break
			case "BDSP":
				gens = append(gens, "8.2")
				break
			case "PLA":
				gens = append(gens, "9.1")
				break
			default:
				if _, err := strconv.Atoi(gen); err == nil {
					gens = append(gens, gen)
					break
				}
			}
		}
	}

	logger := log.FromContext(r.Context())
	db := ent.FromContext(r.Context())
	if db == nil {
		logger.Error("db is nil")
		chix.JSON(w, r, http.StatusInternalServerError, chix.M{"error": "failed to connect to database"})
		return
	}

	switch entityType {
	case "pokemon":
		query := db.Pokemon.Query()
		var args []predicate.Pokemon

		if len(gens) > 0 {
			args = append(args, pokemon.GenerationIn(gens...))
		}

		if payload.LegalOnly {
			args = append(args, pokemon.Legal(true))
		}

		var orderField pokemon.OrderOption

		switch sortField {
		case "popularity":
			orderField = pokemon.ByDownloadCount(orderDir)
		case "latest":
			fallthrough
		default:
			orderField = pokemon.ByUploadDatetime(orderDir)
		}

		query.Where(pokemon.And(args...)).Order(orderField)

		amount, err := query.Count(r.Context())
		if err != nil {
			logger.WithError(err).Error("failed to count the pokemon in the query")
			chix.JSON(w, r, http.StatusInternalServerError, chix.M{"error": "failed to list pokemon"})
			return
		}

		mons, err := query.Limit(limit).Offset((page - 1) * limit).All(r.Context())
		if err != nil {
			logger.WithError(err).Error("failed to list pokemon")
			chix.JSON(w, r, http.StatusInternalServerError, chix.M{"error": "failed to list pokemon"})
			return
		}

		resp := gpssPokemonListResponse{
			Total:   amount,
			Page:    page,
			Pages:   int(math.Ceil(float64(amount) / float64(limit))),
			Pokemon: []gpssPokemon{},
		}

		for _, mon := range mons {
			resp.Pokemon = append(resp.Pokemon, gpssPokemon{
				Legal:      mon.Legal,
				Generation: mon.Generation,
				Code:       mon.DownloadCode,
				Base64:     mon.Base64,
			})
		}

		chix.JSON(w, r, http.StatusOK, resp)
		return
	case "bundle", "bundles":
		query := db.Bundle.Query()

		var args []predicate.Bundle
		minGen := "1"
		maxGen := "10"

		if len(gens) > 0 {
			var nums []json.Number
			for _, gen := range gens {
				nums = append(nums, json.Number(gen))
			}

			// now sort the list
			slices.Sort(nums)

			// Get the first and last items in the list
			minGen = nums[0].String()
			maxGen = nums[len(nums)-1].String()

			args = append(args, bundle.MaxGenLTE(maxGen), bundle.MinGenGTE(minGen))
		}

		if payload.LegalOnly {
			args = append(args, bundle.Legal(true))
		}

		var orderField bundle.OrderOption

		switch sortField {
		case "popularity":
			orderField = bundle.ByDownloadCount(orderDir)
		case "latest":
			fallthrough
		default:
			orderField = bundle.ByUploadDatetime(orderDir)
		}

		query.Where(bundle.And(args...)).Order(orderField)

		amount, err := query.Count(r.Context())
		if err != nil {
			logger.WithError(err).Error("failed to count the bundles in the query")
			chix.JSON(w, r, http.StatusInternalServerError, chix.M{"error": "failed to list bundles"})
			return
		}

		bundles, err := query.Limit(limit).Offset((page - 1) * limit).WithPokemons().All(r.Context())
		if err != nil {
			logger.WithError(err).Error("failed to list bundles")
			chix.JSON(w, r, http.StatusInternalServerError, chix.M{"error": "failed to list bundles"})
			return
		}

		resp := gpssBundleListResponse{
			Total:   amount,
			Page:    page,
			Pages:   int(math.Ceil(float64(amount) / float64(limit))),
			Bundles: []gpssBundle{},
		}

		for _, bun := range bundles {
			tmpBun := gpssBundle{
				Legal:         bun.Legal,
				MinGen:        bun.MinGen,
				MaxGen:        bun.MaxGen,
				Patreon:       false,
				Count:         len(bun.Edges.Pokemons),
				DownloadCode:  bun.DownloadCode,
				DownloadCodes: []string{},
				Pokemons:      []gpssBundlePokemon{},
			}

			var seenGens []json.Number
			for _, mon := range bun.Edges.Pokemons {
				seenGens = append(seenGens, json.Number(mon.Generation))
				tmpBun.DownloadCodes = append(tmpBun.DownloadCodes, mon.DownloadCode)
				tmpBun.Pokemons = append(tmpBun.Pokemons, gpssBundlePokemon{
					Legal:      mon.Legal,
					Generation: mon.Generation,
					Base64:     mon.Base64,
				})
			}

			slices.Sort(seenGens)
			// Noticed that some of the min/max gens on bundles are wrong, so let's re-calculate it.
			tmpBun.MinGen = seenGens[0].String()
			tmpBun.MaxGen = seenGens[len(seenGens)-1].String()

			resp.Bundles = append(resp.Bundles, tmpBun)
		}

		chix.JSON(w, r, http.StatusOK, resp)
		return
	default:
		chix.JSON(w, r, http.StatusNotFound, chix.M{"error": "not found"})
		return
	}
}

func (h *Handler) download(w http.ResponseWriter, r *http.Request) {
	entityType := chi.URLParam(r, "type")
	if entityType == "" {
		chix.JSON(w, r, http.StatusBadRequest, chix.M{"error": "missing entity type"})
		return
	}

	downloadCode := chi.URLParam(r, "code")
	if downloadCode == "" {
		chix.JSON(w, r, http.StatusBadRequest, chix.M{"error": "missing download code"})
		return
	}

	logger := log.FromContext(r.Context())
	db := ent.FromContext(r.Context())
	if db == nil {
		logger.Error("db is nil")
		chix.JSON(w, r, http.StatusInternalServerError, chix.M{"error": "failed to connect to database"})
		return
	}

	switch entityType {
	case "pokemon":
		result, err := db.Pokemon.Query().Where(pokemon.DownloadCode(downloadCode)).First(r.Context())
		if err != nil {
			if ent.IsNotFound(err) {
				chix.JSON(w, r, http.StatusNotFound, chix.M{"error": "pokemon not found"})
				return
			}
			logger.WithError(err).WithField("download_code", downloadCode).Error("failed to find pokemon")
			chix.JSON(w, r, http.StatusInternalServerError, chix.M{"error": "failed to get pokemon"})
			return
		}

		// Increment the download count by 1
		_, err = result.Update().AddDownloadCount(1).Save(r.Context())
		if err != nil {
			logger.WithError(err).WithField("download_code", downloadCode).Error("failed to update download count")
			chix.JSON(w, r, http.StatusInternalServerError, chix.M{"error": "failed to get pokemon"})
			return
		}
		// Since PKSM just clones the B64 from the list endpoint, we don't actually have to return anything
		chix.JSON(w, r, http.StatusOK, chix.M{})
		return
	case "bundle", "bundles":
		result, err := db.Bundle.Query().WithPokemons().Where(bundle.DownloadCode(downloadCode)).First(r.Context())
		if err != nil {
			if ent.IsNotFound(err) {
				chix.JSON(w, r, http.StatusNotFound, chix.M{"error": "bundle not found"})
				return
			}
			logger.WithError(err).WithField("download_code", downloadCode).Error("failed to find bundle")
			chix.JSON(w, r, http.StatusInternalServerError, chix.M{"error": "failed to get bundle"})
			return
		}

		// Increment the download count by 1
		_, err = result.Update().AddDownloadCount(1).Save(r.Context())
		if err != nil {
			logger.WithError(err).WithField("download_code", downloadCode).Error("failed to update download count")
			chix.JSON(w, r, http.StatusInternalServerError, chix.M{"error": "failed to get bundle"})
			return
		}

		// We also need to increment the download counts of all the pokemon

		mons, err := result.QueryPokemons().All(r.Context())
		if err != nil {
			logger.WithError(err).WithField("download_code", downloadCode).Error("failed to get pokemons from the bundle")
			chix.JSON(w, r, http.StatusInternalServerError, chix.M{"error": "failed to get bundle"})
			return
		}

		for _, mon := range mons {
			_, err = mon.Update().AddDownloadCount(1).Save(r.Context())
			if err != nil {
				logger.WithError(err).WithField("download_code", downloadCode).Error("failed to update download count for pokemon in bundle")
				chix.JSON(w, r, http.StatusInternalServerError, chix.M{"error": "failed to get bundle"})
			}
		}

		// Since PKSM just clones the B64 from the list endpoint, we don't actually have to return anything
		chix.JSON(w, r, http.StatusOK, chix.M{})
		return
	default:
		chix.JSON(w, r, http.StatusBadRequest, chix.M{"error": "unknown entity type"})
		return
	}
}

func (h *Handler) uploadPokemon(w http.ResponseWriter, r *http.Request) {
	logger := log.FromContext(r.Context())
	db := ent.FromContext(r.Context())

	// No point in executing GpssConsole if we have no database
	if db == nil {
		logger.Error("db is nil")
		chix.JSON(w, r, http.StatusInternalServerError, chix.M{"error": "db is nil"})
		return
	}

	// We can re-use code from the legality endpoint to make it easier to set up.
	args, statusCode, err := utils.PrepareCall(r, "legality")
	if err != nil {
		chix.JSON(w, r, statusCode, chix.M{"error": err.Error()})
		return
	}

	// Check to see if the base64 already exists in the database
	pkmn, err := db.Pokemon.Query().Where(pokemon.Base64(args.Pokemon)).First(r.Context())
	if err != nil {
		if !ent.IsNotFound(err) {
			logger.WithError(err).Error("failed to check database for existing pokemon")
			chix.JSON(w, r, http.StatusInternalServerError, chix.M{"error": "failed to upload pokemon"})
			return
		}
	} else {
		// Return the download code
		chix.JSON(w, r, http.StatusOK, chix.M{"code": pkmn.DownloadCode})
		return
	}

	pkmn, err = h.uploadPkmn(r, db, logger, *args)
	if err != nil {
		chix.JSON(w, r, http.StatusInternalServerError, chix.M{"error": "failed to upload pokemon"})
		return
	}

	chix.JSON(w, r, http.StatusOK, chix.M{"code": pkmn.DownloadCode})
}

func (h *Handler) uploadBundle(w http.ResponseWriter, r *http.Request) {
	logger := log.FromContext(r.Context())
	db := ent.FromContext(r.Context())
	// No point in executing GpssConsole if we have no database
	if db == nil {
		logger.Error("db is nil")
		chix.JSON(w, r, http.StatusInternalServerError, chix.M{"error": "failed to connect to database"})
		return
	}

	// Because the approach for bundles is different, we can't rely on the shared code from the legality check.
	countHeader := r.Header.Get("count")
	if countHeader == "" {
		chix.JSON(w, r, http.StatusBadRequest, chix.M{"error": "missing count from header"})
		return
	}

	count, err := strconv.Atoi(countHeader)
	if err != nil {
		chix.JSON(w, r, http.StatusBadRequest, chix.M{"error": "invalid count header"})
		return
	}

	if count < 1 || count > 6 {
		chix.JSON(w, r, http.StatusBadRequest, chix.M{"error": "count must be between 1 and 6"})
		return
	}

	generations := strings.Split(r.Header.Get("generations"), ",")
	if len(generations) != count {
		chix.JSON(w, r, http.StatusBadRequest, chix.M{"error": "missing generations header or invalid amount"})
		return
	}

	// Set the limit to 5 MB
	err = r.ParseMultipartForm(5 * 1024 * 1024)
	if err != nil {
		logger.WithError(err).Error("failed to parse multipart form")
		chix.JSON(w, r, http.StatusInternalServerError, chix.M{"error": "failed to upload bundle"})
	}

	tx, err := db.Tx(r.Context())
	if err != nil {
		logger.WithError(err).Error("failed to begin transaction")
		chix.JSON(w, r, http.StatusInternalServerError, chix.M{"error": "failed to upload bundle"})
		return
	}

	bundleLegal := true
	var mons []*ent.Pokemon
	for i := 0; i < count; i++ {
		pkmn, _, err := r.FormFile(fmt.Sprintf("pkmn%d", i+1))
		if err != nil {
			tx.Rollback()
			logger.WithError(err).Error("failed to get pokemon from bundle")
			chix.JSON(w, r, http.StatusInternalServerError, chix.M{"error": "failed to upload bundle"})
			return
		}

		defer pkmn.Close()

		// Base64 encode the pokemon
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, pkmn); err != nil {
			tx.Rollback()
			logger.WithError(err).Error("failed to copy pokemon data")
			chix.JSON(w, r, http.StatusInternalServerError, chix.M{"error": "failed to upload bundle"})
			return
		}

		pkmn.Close()
		b64Str := base64.StdEncoding.EncodeToString(buf.Bytes())

		// Check to see if the mon already exists

		mon, err := db.Pokemon.Query().Where(pokemon.Base64(b64Str)).First(r.Context())
		if err != nil {
			if !ent.IsNotFound(err) {
				tx.Rollback()
				logger.WithError(err).Error("failed to search for pokemon")
				chix.JSON(w, r, http.StatusInternalServerError, chix.M{"error": "failed to upload bundle"})
				return
			}
		} else {
			// mon exists, we can move onto the next.
			mons = append(mons, mon)
			continue
		}

		args := models.GpssConsoleArgs{
			Mode:       "legality",
			Generation: generations[i],
			Pokemon:    b64Str,
		}

		mon, err = h.uploadPkmn(r, tx.Client(), logger, args)
		if err != nil {
			tx.Rollback()
			logger.WithError(err).Error("failed to upload pokemon data")
			chix.JSON(w, r, http.StatusInternalServerError, chix.M{"error": "failed to upload bundle"})
			return
		}

		if !mon.Legal {
			bundleLegal = false
		}

		mons = append(mons, mon)
	}

	// Sort the generations slice
	slices.Sort(generations)

	ids := make([]int, len(mons))
	for i, mon := range mons {
		ids[i] = mon.ID
	}

	// Check to see if we have a bundle already
	existingBun, err := db.Bundle.Query().WithPokemons().Where(bundle.And(bundle.HasPokemonsWith(pokemon.IDIn(ids...)), bundle.MinGen(generations[0]), bundle.MaxGen(generations[len(generations)-1]))).First(r.Context())
	if err != nil && !ent.IsNotFound(err) {
		tx.Rollback()
		logger.WithError(err).Error("failed to search for pokemon")
		chix.JSON(w, r, http.StatusInternalServerError, chix.M{"error": "failed to upload bundle"})
		return
	} else if existingBun != nil && len(existingBun.Edges.Pokemons) == count {
		tx.Commit()
		chix.JSON(w, r, http.StatusOK, chix.M{"code": existingBun.DownloadCode})
		return
	}

	// Now let's create the bundle
	downloadCode, err := utils.GenerateDownloadCode(r.Context(), "bundle")
	if err != nil {
		logger.WithError(err).Error("failed to generate bundle download code")
		chix.JSON(w, r, http.StatusInternalServerError, chix.M{"error": "failed to upload bundle"})
		return
	}
	newBundle, err := tx.Bundle.Create().SetMinGen(generations[0]).SetMaxGen(generations[len(mons)-1]).
		SetLegal(bundleLegal).AddPokemons(mons...).SetUploadDatetime(time.Now()).SetDownloadCode(downloadCode).Save(r.Context())

	if err != nil {
		tx.Rollback()
		logger.WithError(err).Error("failed to upload bundle")
		chix.JSON(w, r, http.StatusInternalServerError, chix.M{"error": "failed to upload bundle"})
		return
	}

	tx.Commit()
	chix.JSON(w, r, http.StatusOK, chix.M{"code": newBundle.DownloadCode})

}

func (h *Handler) uploadPkmn(r *http.Request, db *ent.Client, logger log.Interface, args models.GpssConsoleArgs) (*ent.Pokemon, error) {
	// We call out to the same function as the legality check endpoint does as we need to do two things
	// 1. Make sure the file sent over is an actual Pokémon
	// 2. Check the legality status.
	result, err := utils.ExecGpssConsole[models.GpssLegalityCheckReply](r.Context(), args)
	if err != nil {
		logger.WithError(err).Error("failed to communicate with GpssConsole")
		return nil, err
	}

	downloadCode, err := utils.GenerateDownloadCode(r.Context(), "pokemon")
	if err != nil {
		logger.WithError(err).Error("failed to generate download code")
		return nil, err
	}

	pkmn, err := db.Pokemon.Create().
		SetUploadDatetime(time.Now()).
		SetGeneration(args.Generation).
		SetLegal(result.Legal).
		SetDownloadCode(downloadCode).
		SetBase64(args.Pokemon).Save(r.Context())
	if err != nil {
		logger.WithError(err).Error("failed to insert pokemon into database")
		return nil, err
	}

	return pkmn, nil
}
