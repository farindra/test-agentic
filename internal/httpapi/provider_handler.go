package httpapi

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"

	"test-agentic/internal/aiprovider"
	"test-agentic/internal/store"
)

// providerView: API key TIDAK PERNAH dibalikin utuh ke frontend setelah
// tersimpan — cukup penanda "sudah keisi" + 4 karakter terakhir, biar UI bisa
// nunjukin "•••• ab12" tanpa nyimpen secret di browser/network log.
type providerView struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Type          string `json:"type"`
	BaseURL       string `json:"base_url"`
	DefaultModel  string `json:"default_model"`
	IsActive      bool   `json:"is_active"`
	APIKeySet     bool   `json:"api_key_set"`
	APIKeyPreview string `json:"api_key_preview,omitempty"`
}

func toProviderView(p store.Provider) providerView {
	v := providerView{
		ID: p.ID, Name: p.Name, Type: string(p.Type), BaseURL: p.BaseURL,
		DefaultModel: p.DefaultModel, IsActive: p.IsActive, APIKeySet: p.APIKey != "",
	}
	if n := len(p.APIKey); n >= 4 {
		v.APIKeyPreview = "••••" + p.APIKey[n-4:]
	} else if n > 0 {
		v.APIKeyPreview = "••••"
	}
	return v
}

func (a *API) listProviders(c *fiber.Ctx) error {
	ctx, cancel := ctx15(c)
	defer cancel()
	rows, err := a.st.ListProviders(ctx)
	if err != nil {
		return errJSON(c, fiber.StatusInternalServerError, err)
	}
	out := make([]providerView, 0, len(rows))
	for _, p := range rows {
		out = append(out, toProviderView(p))
	}
	return c.JSON(fiber.Map{"providers": out})
}

func (a *API) getProvider(c *fiber.Ctx) error {
	ctx, cancel := ctx15(c)
	defer cancel()
	p, err := a.st.GetProvider(ctx, c.Params("id"))
	if errors.Is(err, store.ErrNotFound) {
		return errJSON(c, fiber.StatusNotFound, err)
	}
	if err != nil {
		return errJSON(c, fiber.StatusInternalServerError, err)
	}
	return c.JSON(toProviderView(*p))
}

type providerReq struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	BaseURL      string `json:"base_url"`
	APIKey       string `json:"api_key"`
	DefaultModel string `json:"default_model"`
	IsActive     *bool  `json:"is_active"`
}

func (a *API) createProvider(c *fiber.Ctx) error {
	var req providerReq
	if err := c.BodyParser(&req); err != nil || strings.TrimSpace(req.Name) == "" || req.Type == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name & type wajib"})
	}
	ctx, cancel := ctx15(c)
	defer cancel()
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	p, err := a.st.CreateProvider(ctx, store.Provider{
		Name: req.Name, Type: store.ProviderType(req.Type), BaseURL: req.BaseURL,
		APIKey: req.APIKey, DefaultModel: req.DefaultModel, IsActive: isActive,
	})
	if err != nil {
		return errJSON(c, fiber.StatusInternalServerError, err)
	}
	return c.Status(fiber.StatusCreated).JSON(toProviderView(*p))
}

func (a *API) updateProvider(c *fiber.Ctx) error {
	var req providerReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "body tidak valid"})
	}
	ctx, cancel := ctx15(c)
	defer cancel()

	existing, err := a.st.GetProvider(ctx, c.Params("id"))
	if errors.Is(err, store.ErrNotFound) {
		return errJSON(c, fiber.StatusNotFound, err)
	}
	if err != nil {
		return errJSON(c, fiber.StatusInternalServerError, err)
	}

	apiKey := existing.APIKey
	if req.APIKey != "" {
		apiKey = req.APIKey // kosong = dipertahankan; UI gak wajib re-submit secret tiap edit
	}
	isActive := existing.IsActive
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	updated, err := a.st.UpdateProvider(ctx, store.Provider{
		ID: existing.ID, Name: req.Name, Type: store.ProviderType(req.Type), BaseURL: req.BaseURL,
		APIKey: apiKey, DefaultModel: req.DefaultModel, IsActive: isActive,
	})
	if err != nil {
		return errJSON(c, fiber.StatusInternalServerError, err)
	}
	return c.JSON(toProviderView(*updated))
}

func (a *API) deleteProvider(c *fiber.Ctx) error {
	ctx, cancel := ctx15(c)
	defer cancel()
	if err := a.st.DeleteProvider(ctx, c.Params("id")); errors.Is(err, store.ErrNotFound) {
		return errJSON(c, fiber.StatusNotFound, err)
	} else if err != nil {
		return errJSON(c, fiber.StatusInternalServerError, err)
	}
	return c.JSON(fiber.Map{"deleted": true})
}

func (a *API) testProvider(c *fiber.Ctx) error {
	ctx, cancel := ctx30(c)
	defer cancel()
	reply, err := a.orch.TestProvider(ctx, c.Params("id"))
	if err != nil {
		// 400, bukan 502: Cloudflare di depan origin nge-intercept status 5xx
		// dan nimpa body-nya sama halaman error generik sendiri.
		return errJSON(c, fiber.StatusBadRequest, err)
	}
	return c.JSON(fiber.Map{"ok": true, "reply": reply})
}

type listModelsReq struct {
	Type       string  `json:"type"`
	BaseURL    string  `json:"base_url"`
	APIKey     string  `json:"api_key"`
	ProviderID *string `json:"provider_id"` // dikirim pas EDIT — fallback ambil key tersimpan kalau api_key dikosongin
}

// listProviderModels ngambil daftar model langsung dari API provider, buat
// diisiin ke dropdown "Model Default" di UI (alih-alih user ngetik manual
// dan gampang typo). Dipanggil dari modal create/edit — SEBELUM provider
// disimpan — jadi nerima config lewat body, bukan lookup by id.
func (a *API) listProviderModels(c *fiber.Ctx) error {
	var req listModelsReq
	if err := c.BodyParser(&req); err != nil || req.Type == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "type wajib"})
	}
	ctx, cancel := ctx15(c)
	defer cancel()

	apiKey := req.APIKey
	if apiKey == "" && req.ProviderID != nil && *req.ProviderID != "" {
		existing, err := a.st.GetProvider(ctx, *req.ProviderID)
		if err == nil {
			apiKey = existing.APIKey
		}
	}

	client, err := aiprovider.New(aiprovider.Config{Kind: aiprovider.Kind(req.Type), BaseURL: req.BaseURL, APIKey: apiKey})
	if err != nil {
		return errJSON(c, fiber.StatusBadRequest, err)
	}
	models, err := client.ListModels(ctx)
	if err != nil {
		return errJSON(c, fiber.StatusBadRequest, err)
	}
	return c.JSON(fiber.Map{"models": models})
}
