package gpss

import (
	"encoding/json"
	"math"
	"net/http"
	"slices"
	"strconv"

	"entgo.io/ent/dialect/sql"
	"github.com/FlagBrew/local-gpss/internal/database/ent"
	"github.com/FlagBrew/local-gpss/internal/database/ent/bundle"
	"github.com/FlagBrew/local-gpss/internal/database/ent/pokemon"
	"github.com/FlagBrew/local-gpss/internal/database/ent/predicate"
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
	r.Post("/upload/{type}", h.upload)
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

	if r.URL.Query().Get("page") != "" {
		parsedPage, err := strconv.Atoi(r.URL.Query().Get("page"))
		if err == nil {
			page = parsedPage
		}
	}

	if r.URL.Query().Get("amount") != "" {
		parsedAmount, err := strconv.Atoi(r.URL.Query().Get("amount"))
		if err == nil && parsedAmount < 101 && parsedAmount > 0 {
			limit = parsedAmount
		}
	}

	gens := []string{}
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
		chix.JSON(w, r, http.StatusInternalServerError, chix.M{"error": "db is nil"})
		return
	}

	switch entityType {
	case "pokemon":
		query := db.Pokemon.Query()
		args := []predicate.Pokemon{}

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

		args := []predicate.Bundle{}
		minGen := "1"
		maxGen := "10"

		if len(gens) > 0 {
			nums := []json.Number{}
			for _, gen := range gens {
				nums = append(nums, json.Number(gen))
			}

			// now sort the list
			slices.Sort(nums)

			// Get the first and last items in the list
			minGen = nums[0].String()
			maxGen = nums[len(nums)-1].String()

			args = append(args, bundle.MaxGen(maxGen), bundle.MinGen(minGen))
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

			seenGens := []json.Number{}
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
		chix.JSON(w, r, http.StatusInternalServerError, chix.M{"error": "db is nil"})
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
			chix.JSON(w, r, http.StatusInternalServerError, chix.M{"error": "failed to get pokemon"})
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

func (h *Handler) upload(w http.ResponseWriter, r *http.Request) {
}
