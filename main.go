package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/google/generative-ai-go/genai"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/lib/pq"
	"github.com/xuri/excelize/v2"
	"google.golang.org/api/option"
)

// =============================================================================
// MODELS (Structs for JSON and DB)
// =============================================================================
const (
	AppRoleAdmin     = "admin"
	AppRoleViewer    = "viewer"
	AppRoleWarehouse = "warehouse"
	AppRoleK3        = "k3"
	AppRoleK5        = "k5"
)
type LogEntry struct {
	Action    string    `json:"action"`    // e.g., "membuat issue", "menandai solved"
	User      string    `json:"user"`      // The username who performed the action
	Timestamp time.Time `json:"timestamp"` // When the action was performed
}

type Logs []LogEntry

func (l Logs) Value() (driver.Value, error) {
	if len(l) == 0 {
		return json.Marshal([]LogEntry{}) 
	}
	return json.Marshal(l)
}

func (l *Logs) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	if b == nil || string(b) == "null" {
		*l = []LogEntry{}
		return nil
	}
	return json.Unmarshal(b, &l)
}
type User struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
type CreateCommentPayload struct {
	SenderID         string   `json:"sender_id"`
	Text             string   `json:"text"`
	ReplyToCommentID *string  `json:"reply_to_comment_id,omitempty"`
	ReplyToUserID    *string  `json:"reply_to_user_id,omitempty"`
	Images           []string `json:"images"`
}
type IssueTitle struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
}
type IssueComment struct {
	ID               string    `json:"id"`
	IssueID          int       `json:"issue_id"`
	Sender           User      `json:"sender"`
	Text             string    `json:"text"`
	Timestamp        time.Time `json:"timestamp"`
	ReplyTo          *User     `json:"reply_to,omitempty"`
	ReplyToCommentID *string   `json:"reply_to_comment_id,omitempty"`
	IsEdited         bool      `json:"is_edited"`
	ImageUrls        []string  `json:"image_urls"`
}

type Chat struct {
	ID        int       `json:"chat_id"`
	PanelNoPp string    `json:"panel_no_pp"`
	CreatedAt time.Time `json:"created_at"`
}
type ChatMessage struct {
	ID            int        `json:"id"`
	ChatID        int        `json:"chat_id"`
	SenderUsername string    `json:"sender_username"`
	Text          *string    `json:"text"` // Pointer agar bisa null
	ImageData     *string    `json:"image_data,omitempty"` // Pointer dan omitempty
	RepliedIssueID *int       `json:"replied_issue_id,omitempty"` // Pointer dan omitempty
	CreatedAt     time.Time `json:"created_at"`
}
type Issue struct {
	ID          int       `json:"issue_id"`
	ChatID      int       `json:"chat_id"`
	Title       string    `json:"issue_title"` 
	Description string    `json:"issue_description"`
	Type      string    `json:"issue_type,omitempty"`
	Status    string    `json:"issue_status"`
	Logs      Logs      `json:"logs"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Photo struct {
	ID        int    `json:"photo_id"`
	IssueID   int    `json:"issue_id"`
	PhotoData string `json:"photo"` 
}

type IssueWithPhotos struct {
	Issue
	Photos []Photo `json:"photos"`
}

type customTime struct {
	time.Time
}
type PanelKeyInfo struct {
	NoPp    string `json:"no_pp"`
	NoPanel string `json:"no_panel"`
	Project string `json:"project"`
	NoWbs   string `json:"no_wbs"`
}
const ctLayout = "2006-01-02T15:04:05.999999"

func (ct *customTime) UnmarshalJSON(b []byte) (err error) {
	s := strings.Trim(string(b), "\"")
	if s == "null" || s == "" {
		ct.Time = time.Time{}
		return
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t, err = time.Parse(ctLayout, s)
	}
	if err == nil {
		ct.Time = t
	}
	return
}

func (ct *customTime) MarshalJSON() ([]byte, error) {
	if ct.Time.IsZero() {
		return []byte("null"), nil
	}
	return []byte(fmt.Sprintf("\"%s\"", ct.Time.Format(time.RFC3339))), nil
}

func (ct *customTime) Scan(value interface{}) error {
	if value == nil {
		ct.Time = time.Time{}
		return nil
	}
	t, ok := value.(time.Time)
	if !ok {
		return fmt.Errorf("could not scan type %T into customTime", value)
	}
	ct.Time = t
	return nil
}

func (ct customTime) Value() (driver.Value, error) {
	if ct.Time.IsZero() {
		return nil, nil
	}
	return ct.Time, nil
}

type Company struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
}
type CompanyAccount struct {
	Username  string `json:"username"`
	Password  string `json:"password"`
	CompanyID string `json:"company_id"`
}
type Panel struct {
	NoPp            string      `json:"no_pp"`
	NoPanel         *string     `json:"no_panel"`
	NoWbs           *string     `json:"no_wbs"`
	Project         *string     `json:"project"`
	PercentProgress *float64    `json:"percent_progress"`
	StartDate       *customTime `json:"start_date"`
	TargetDelivery  *customTime `json:"target_delivery"`
	StatusBusbarPcc *string     `json:"status_busbar_pcc"`
	StatusBusbarMcc *string     `json:"status_busbar_mcc"`
	StatusComponent *string     `json:"status_component"`
	StatusPalet     *string     `json:"status_palet"`
	StatusCorepart  *string     `json:"status_corepart"`
	AoBusbarPcc     *customTime `json:"ao_busbar_pcc"`
	AoBusbarMcc     *customTime `json:"ao_busbar_mcc"`
	CreatedBy       *string     `json:"created_by"`
	VendorID        *string     `json:"vendor_id"`
	IsClosed        bool        `json:"is_closed"`
	ClosedDate      *customTime `json:"closed_date"`
	PanelType       *string     `json:"panel_type,omitempty"`
	Remarks         *string     `json:"remarks,omitempty"`
	CloseDateBusbarPcc *customTime `json:"close_date_busbar_pcc,omitempty"`
	CloseDateBusbarMcc *customTime `json:"close_date_busbar_mcc,omitempty"`

}
type Busbar struct {
	ID        int     `json:"id"`
	PanelNoPp string  `json:"panel_no_pp"`
	Vendor    string  `json:"vendor"`
	Remarks   *string `json:"remarks"`
}
type Component struct {
	ID        int    `json:"id"`
	PanelNoPp string `json:"panel_no_pp"`
	Vendor    string `json:"vendor"`
}
type Palet struct {
	ID        int    `json:"id"`
	PanelNoPp string `json:"panel_no_pp"`
	Vendor    string `json:"vendor"`
}
type Corepart struct {
	ID        int    `json:"id"`
	PanelNoPp string `json:"panel_no_pp"`
	Vendor    string `json:"vendor"`
}
type PanelDisplayData struct {
	Panel                Panel           `json:"panel"`
	PanelVendorName      *string         `json:"panel_vendor_name"`
	PanelRemarks         *string         `json:"panel_remarks"`
	BusbarVendorNames    *string         `json:"busbar_vendor_names"`
	BusbarVendorIds      []string        `json:"busbar_vendor_ids"`
	BusbarRemarks        json.RawMessage `json:"busbar_remarks"` 
	ComponentVendorNames *string         `json:"component_vendor_names"`
	ComponentVendorIds   []string        `json:"component_vendor_ids"`
	PaletVendorNames     *string         `json:"palet_vendor_names"`
	PaletVendorIds       []string        `json:"palet_vendor_ids"`
	CorepartVendorNames  *string         `json:"corepart_vendor_names"`
	CorepartVendorIds    []string        `json:"corepart_vendor_ids"`
}

// =============================================================================
// DB INTERFACE
// =============================================================================
type DBTX interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
}

// =============================================================================
// APP & MAIN
// =============================================================================
type App struct {
	Router *mux.Router
	DB     *sql.DB
}

func (a *App) Initialize(dbUser, dbPassword, dbName, dbHost string) {
	dbSslMode := os.Getenv("DB_SSLMODE")
	if dbSslMode == "" {
		dbSslMode = "require"
	}
	connectionString := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=%s", dbUser, dbPassword, dbHost, dbName, dbSslMode)

	var err error
	a.DB, err = sql.Open("postgres", connectionString)
	if err != nil {
		log.Fatalf("Gagal membuka koneksi DB: %v", err)
	}

	a.DB.SetMaxOpenConns(25)
	a.DB.SetMaxIdleConns(25)
	a.DB.SetConnMaxLifetime(5 * time.Minute)

	err = a.DB.Ping()
	if err != nil {
		log.Fatalf("Tidak dapat terhubung ke database: %v", err)
	}
	log.Println("Berhasil terhubung ke database!")

	initDB(a.DB)
	a.Router = mux.NewRouter().StrictSlash(true)
	a.initializeRoutes()
}

func (a *App) Run(addr string) {
	// [FIX] Bungkus router Anda dengan middleware CORS di sini
	
	// Tentukan domain yang diizinkan. Tanda '*' berarti mengizinkan semua domain.
	// Untuk keamanan produksi, ganti '*' dengan domain web Flutter Anda (misal: "http://localhost:port", "https://nama-web-anda.com")
	allowedOrigins := handlers.AllowedOrigins([]string{"*"})
	
	// Tentukan metode HTTP yang diizinkan
	allowedMethods := handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"})
	
	// Tentukan header yang diizinkan
	allowedHeaders := handlers.AllowedHeaders([]string{"Content-Type", "Authorization"})

	log.Printf("Server berjalan di %s", addr)
	// Gunakan handler CORS yang sudah dikonfigurasi untuk menjalankan server
	log.Fatal(http.ListenAndServe(addr, handlers.CORS(allowedOrigins, allowedMethods, allowedHeaders)(a.Router)))
}

func main() {
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	dbHost := os.Getenv("DB_HOST")

	if dbUser == "" || dbPassword == "" || dbName == "" || dbHost == "" {
		log.Fatal("Variabel environment DB_USER, DB_PASSWORD, DB_NAME, dan DB_HOST harus di-set!")
	}

	app := App{}
	app.Initialize(dbUser, dbPassword, dbName, dbHost)
	app.Run(":8080")
}
// =============================================================================
// ROUTES
// =============================================================================
func (a *App) initializeRoutes() {
	// Auth & User Management
	a.Router.HandleFunc("/login", a.loginHandler).Methods("POST")
	a.Router.HandleFunc("/company-by-username/{username}", a.getCompanyByUsernameHandler).Methods("GET")
	a.Router.HandleFunc("/user/{username}/password", a.updatePasswordHandler).Methods("PUT")
	a.Router.HandleFunc("/company-with-account", a.insertCompanyWithAccountHandler).Methods("POST")
	a.Router.HandleFunc("/company-with-account/{username}", a.updateCompanyAndAccountHandler).Methods("PUT")
	a.Router.HandleFunc("/account/{username}", a.deleteCompanyAccountHandler).Methods("DELETE")
	a.Router.HandleFunc("/accounts", a.getAllCompanyAccountsHandler).Methods("GET")
	a.Router.HandleFunc("/account/exists/{username}", a.isUsernameTakenHandler).Methods("GET")
	a.Router.HandleFunc("/users/display", a.getAllUserAccountsForDisplayHandler).Methods("GET")
	a.Router.HandleFunc("/users/colleagues/display", a.getColleagueAccountsForDisplayHandler).Methods("GET")
	a.Router.HandleFunc("/accounts/search", a.searchUsernamesHandler).Methods("GET")


	// Company Management
	a.Router.HandleFunc("/company", a.insertCompanyHandler).Methods("POST")
	a.Router.HandleFunc("/company/{id}", a.getCompanyByIdHandler).Methods("GET")
	a.Router.HandleFunc("/company/{id}", a.updateCompanyHandler).Methods("PUT")
	a.Router.HandleFunc("/companies", a.getAllCompaniesHandler).Methods("GET")
	a.Router.HandleFunc("/company-by-name/{name}", a.getCompanyByNameHandler).Methods("GET")
	a.Router.HandleFunc("/vendors", a.getCompaniesByRoleHandler).Methods("GET")
	a.Router.HandleFunc("/companies/form-data", a.getUniqueCompanyDataForFormHandler).Methods("GET")


	// Panel Management
	a.Router.HandleFunc("/panels", a.getAllPanelsForDisplayHandler).Methods("GET")
	a.Router.HandleFunc("/panels", a.upsertPanelHandler).Methods("POST")
	a.Router.HandleFunc("/panel/status-ao-k5", a.upsertStatusAOK5).Methods("POST")
	a.Router.HandleFunc("/panel/status-whs", a.upsertStatusWHS).Methods("POST")

	a.Router.HandleFunc("/panels/bulk-delete", a.deletePanelsHandler).Methods("DELETE")
	a.Router.HandleFunc("/panels/{no_pp}", a.deletePanelHandler).Methods("DELETE")
	a.Router.HandleFunc("/panels/all", a.getAllPanelsHandler).Methods("GET")
	a.Router.HandleFunc("/panels/keys", a.getPanelKeysHandler).Methods("GET")
	a.Router.HandleFunc("/panel/{no_pp}", a.getPanelByNoPpHandler).Methods("GET")
	a.Router.HandleFunc("/panel/exists/no-pp/{no_pp}", a.isNoPpTakenHandler).Methods("GET")
	a.Router.HandleFunc("/panels/{old_no_pp}/change-pp", a.changePanelNoPpHandler).Methods("PUT")
	a.Router.HandleFunc("/panel/remark-vendor", a.upsertPanelRemarkHandler).Methods("POST")

	// Rute isPanelNumberUniqueHandler tidak lagi diperlukan karena no_panel tidak unik
	// a.Router.HandleFunc("/panel/exists/no-panel/{no_panel}", a.isPanelNumberUniqueHandler).Methods("GET")

	// Sub-Panel Parts Management
	
	a.Router.HandleFunc("/busbar", a.upsertGenericHandler("busbars", &Busbar{})).Methods("POST")
	a.Router.HandleFunc("/component", a.upsertGenericHandler("components", &Component{})).Methods("POST")
	a.Router.HandleFunc("/palet", a.upsertGenericHandler("palet", &Palet{})).Methods("POST")
	a.Router.HandleFunc("/corepart", a.upsertGenericHandler("corepart", &Corepart{})).Methods("POST")

	// Gunakan handler generik untuk semua operasi DELETE
	a.Router.HandleFunc("/busbar", a.deleteGenericRelationHandler("busbars")).Methods("DELETE")
	a.Router.HandleFunc("/palet", a.deleteGenericRelationHandler("palet")).Methods("DELETE")
	a.Router.HandleFunc("/corepart", a.deleteGenericRelationHandler("corepart")).Methods("DELETE")
	
	a.Router.HandleFunc("/busbar/remark-vendor", a.upsertBusbarRemarkandVendorHandler).Methods("POST")
	a.Router.HandleFunc("/busbars", a.getAllGenericHandler("busbars", func() interface{} { return &Busbar{} })).Methods("GET")
	a.Router.HandleFunc("/components", a.getAllGenericHandler("components", func() interface{} { return &Component{} })).Methods("GET")
	a.Router.HandleFunc("/palets", a.getAllGenericHandler("palet", func() interface{} { return &Palet{} })).Methods("GET")
	a.Router.HandleFunc("/coreparts", a.getAllGenericHandler("corepart", func() interface{} { return &Corepart{} })).Methods("GET")

	// Export & Import
	a.Router.HandleFunc("/export/filtered-data", a.getFilteredDataForExportHandler).Methods("GET")
	a.Router.HandleFunc("/export/custom", a.generateCustomExportJsonHandler).Methods("GET")
	a.Router.HandleFunc("/export/database", a.generateFilteredDatabaseJsonHandler).Methods("GET")
	a.Router.HandleFunc("/import/database", a.importDataHandler).Methods("POST")
	a.Router.HandleFunc("/import/template", a.generateImportTemplateHandler).Methods("GET")
	a.Router.HandleFunc("/import/custom", a.importFromCustomTemplateHandler).Methods("POST")

	// Issue Management Routes
	a.Router.HandleFunc("/panels/{no_pp}/issues", a.getIssuesByPanelHandler).Methods("GET")
	a.Router.HandleFunc("/panels/{no_pp}/issues", a.createIssueForPanelHandler).Methods("POST")
	a.Router.HandleFunc("/issues/{id}", a.getIssueByIDHandler).Methods("GET")
	a.Router.HandleFunc("/issues/{id}", a.updateIssueHandler).Methods("PUT")
	a.Router.HandleFunc("/issues/{id}", a.deleteIssueHandler).Methods("DELETE")
	a.Router.HandleFunc("/issue-titles", a.getAllIssueTitlesHandler).Methods("GET")
	a.Router.HandleFunc("/issue-titles", a.createIssueTitleHandler).Methods("POST")
	a.Router.HandleFunc("/issue-titles/{id}", a.updateIssueTitleHandler).Methods("PUT")
	a.Router.HandleFunc("/issue-titles/{id}", a.deleteIssueTitleHandler).Methods("DELETE")

	// Photo Management Routes
	a.Router.HandleFunc("/issues/{issue_id}/photos", a.addPhotoToIssueHandler).Methods("POST")
	a.Router.HandleFunc("/photos/{id}", a.deletePhotoHandler).Methods("DELETE")

	// Chat Message Routes
	a.Router.HandleFunc("/chats/{chat_id}/messages", a.getMessagesByChatIDHandler).Methods("GET")
	a.Router.HandleFunc("/chats/{chat_id}/messages", a.createMessageHandler).Methods("POST")

	// Issue Comment Routes
	a.Router.HandleFunc("/issues/{issue_id}/comments", a.getCommentsByIssueHandler).Methods("GET")
	a.Router.HandleFunc("/issues/{issue_id}/comments", a.createCommentHandler).Methods("POST")
	a.Router.HandleFunc("/issues/{issue_id}/ask-gemini", a.askGeminiHandler).Methods("POST")
	a.Router.HandleFunc("/panels/{no_pp}/ask-gemini", a.askGeminiAboutPanelHandler).Methods("POST") 


	a.Router.HandleFunc("/comments/{id}", a.updateCommentHandler).Methods("PUT")
	a.Router.HandleFunc("/comments/{id}", a.deleteCommentHandler).Methods("DELETE")
	fs := http.FileServer(http.Dir("./uploads/"))
	a.Router.PathPrefix("/uploads/").Handler(http.StripPrefix("/uploads/", fs))
}

// =============================================================================
// HANDLERS
// =============================================================================

func (a *App) insertCompanyHandler(w http.ResponseWriter, r *http.Request) {
	var c Company
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid payload: "+err.Error())
		return
	}

	query := `
		INSERT INTO companies (id, name, role) VALUES ($1, $2, $3)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			role = EXCLUDED.role`

	_, err := a.DB.Exec(query, c.ID, c.Name, c.Role)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" && strings.Contains(pqErr.Constraint, "companies_name_key") {
			respondWithError(w, http.StatusConflict, fmt.Sprintf("Nama perusahaan '%s' sudah ada.", c.Name))
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Gagal memasukkan perusahaan: "+err.Error())
		return
	}

	respondWithJSON(w, http.StatusCreated, c)
}
func (a *App) upsertPanelHandler(w http.ResponseWriter, r *http.Request) {
	var p Panel
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid payload: "+err.Error())
		return
	}

	query := `
		INSERT INTO panels (no_pp, no_panel, no_wbs, project, percent_progress, start_date, target_delivery, status_busbar_pcc, status_busbar_mcc, status_component, status_palet, status_corepart, ao_busbar_pcc, ao_busbar_mcc, created_by, vendor_id, is_closed, closed_date, panel_type)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
		ON CONFLICT (no_pp) DO UPDATE SET
		no_panel = EXCLUDED.no_panel, no_wbs = EXCLUDED.no_wbs, project = EXCLUDED.project, percent_progress = EXCLUDED.percent_progress, start_date = EXCLUDED.start_date, target_delivery = EXCLUDED.target_delivery, status_busbar_pcc = EXCLUDED.status_busbar_pcc, status_busbar_mcc = EXCLUDED.status_busbar_mcc, status_component = EXCLUDED.status_component, status_palet = EXCLUDED.status_palet, status_corepart = EXCLUDED.status_corepart, ao_busbar_pcc = EXCLUDED.ao_busbar_pcc, ao_busbar_mcc = EXCLUDED.ao_busbar_mcc, created_by = EXCLUDED.created_by, vendor_id = EXCLUDED.vendor_id, is_closed = EXCLUDED.is_closed, closed_date = EXCLUDED.closed_date, panel_type = EXCLUDED.panel_type`

	_, err := a.DB.Exec(query, p.NoPp, p.NoPanel, p.NoWbs, p.Project, p.PercentProgress, p.StartDate, p.TargetDelivery, p.StatusBusbarPcc, p.StatusBusbarMcc, p.StatusComponent, p.StatusPalet, p.StatusCorepart, p.AoBusbarPcc, p.AoBusbarMcc, p.CreatedBy, p.VendorID, p.IsClosed, p.ClosedDate, p.PanelType)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			respondWithError(w, http.StatusConflict, "Gagal menyimpan: Terdapat duplikasi pada No Panel.")
			return
		}
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondWithJSON(w, http.StatusCreated, p)
}

func (a *App) loginHandler(w http.ResponseWriter, r *http.Request) {
	var payload struct{ Username, Password string }
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid payload")
		return
	}
	var account CompanyAccount
	err := a.DB.QueryRow("SELECT password, company_id FROM company_accounts WHERE username = $1", payload.Username).Scan(&account.Password, &account.CompanyID)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Username atau password salah")
		return
	}
	if account.Password == payload.Password {
		var company Company
		err := a.DB.QueryRow("SELECT id, name, role FROM companies WHERE id = $1", account.CompanyID).Scan(&company.ID, &company.Name, &company.Role)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Company not found for user")
			return
		}
		respondWithJSON(w, http.StatusOK, company)
	} else {
		respondWithError(w, http.StatusUnauthorized, "Username atau password salah")
	}
}

func (a *App) updateStatusAOHandler(w http.ResponseWriter, r *http.Request) {
	// 1. Ambil no_pp dari URL, contoh: /panel/PP-123/status-ao
	vars := mux.Vars(r)
	noPp, ok := vars["no_pp"]
	if !ok {
		respondWithError(w, http.StatusBadRequest, "No. PP tidak ditemukan di URL")
		return
	}

	// 2. Siapkan struct untuk menampung data dari body JSON request
	var payload struct {
		StatusBusbarPcc *string     `json:"status_busbar_pcc"`
		StatusBusbarMcc *string     `json:"status_busbar_mcc"`
		StatusComponent *string     `json:"status_component"`
		AoBusbarPcc     *customTime `json:"ao_busbar_pcc"`
		AoBusbarMcc     *customTime `json:"ao_busbar_mcc"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondWithError(w, http.StatusBadRequest, "Payload tidak valid: "+err.Error())
		return
	}

	// 3. Bangun query UPDATE secara dinamis agar hanya field yang dikirim yang diupdate
	var updates []string
	var args []interface{}
	argCounter := 1

	if payload.StatusBusbarPcc != nil {
		updates = append(updates, fmt.Sprintf("status_busbar_pcc = $%d", argCounter))
		args = append(args, *payload.StatusBusbarPcc)
		argCounter++
	}
	if payload.StatusBusbarMcc != nil {
		updates = append(updates, fmt.Sprintf("status_busbar_mcc = $%d", argCounter))
		args = append(args, *payload.StatusBusbarMcc)
		argCounter++
	}
	if payload.StatusComponent != nil {
		updates = append(updates, fmt.Sprintf("status_component = $%d", argCounter))
		args = append(args, *payload.StatusComponent)
		argCounter++
	}
	if payload.AoBusbarPcc != nil {
		updates = append(updates, fmt.Sprintf("ao_busbar_pcc = $%d", argCounter))
		args = append(args, payload.AoBusbarPcc)
		argCounter++
	}
	if payload.AoBusbarMcc != nil {
		updates = append(updates, fmt.Sprintf("ao_busbar_mcc = $%d", argCounter))
		args = append(args, payload.AoBusbarMcc)
		argCounter++
	}

	// Jika tidak ada data yang perlu diupdate, hentikan proses
	if len(updates) == 0 {
		respondWithJSON(w, http.StatusOK, map[string]string{"message": "Tidak ada field yang diupdate"})
		return
	}

	// 4. Finalisasi query dengan menambahkan klausa WHERE
	query := fmt.Sprintf("UPDATE panels SET %s WHERE no_pp = $%d", strings.Join(updates, ", "), argCounter)
	args = append(args, noPp)

	// 5. Eksekusi query
	result, err := a.DB.Exec(query, args...)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Gagal mengupdate status: "+err.Error())
		return
	}

	// 6. Cek apakah ada baris yang terpengaruh
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		respondWithError(w, http.StatusNotFound, "Panel dengan No. PP tersebut tidak ditemukan")
		return
	}

	// 7. Kirim respon sukses
	respondWithJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

func (a *App) getCompanyByUsernameHandler(w http.ResponseWriter, r *http.Request) {
	username := mux.Vars(r)["username"]
	var company Company
	query := `SELECT c.id, c.name, c.role FROM companies c JOIN company_accounts ca ON c.id = ca.company_id WHERE ca.username = $1`
	err := a.DB.QueryRow(query, username).Scan(&company.ID, &company.Name, &company.Role)
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusNotFound, "User not found")
			return
		}
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondWithJSON(w, http.StatusOK, company)
}
func (a *App) updatePasswordHandler(w http.ResponseWriter, r *http.Request) {
	username := mux.Vars(r)["username"]
	var payload struct{ Password string `json:"password"` }
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid payload")
		return
	}
	res, err := a.DB.Exec("UPDATE company_accounts SET password = $1 WHERE username = $2", payload.Password, username)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	count, _ := res.RowsAffected()
	if count == 0 {
		respondWithError(w, http.StatusNotFound, "User not found")
		return
	}
	respondWithJSON(w, http.StatusOK, map[string]string{"status": "success"})
}
func (a *App) insertCompanyWithAccountHandler(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Company Company        `json:"company"`
		Account CompanyAccount `json:"account"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid payload: "+err.Error())
		return
	}
	if payload.Account.Username == "" {
		respondWithError(w, http.StatusBadRequest, "Username tidak boleh kosong.")
		return
	}
	if payload.Account.Password == "" {
		respondWithError(w, http.StatusBadRequest, "Password tidak boleh kosong.")
		return
	}

	tx, err := a.DB.Begin()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	_, err = tx.Exec("INSERT INTO companies (id, name, role) VALUES ($1, $2, $3) ON CONFLICT (id) DO NOTHING", payload.Company.ID, payload.Company.Name, payload.Company.Role)
	if err != nil {
		tx.Rollback()
		respondWithError(w, http.StatusInternalServerError, "Failed to insert company: "+err.Error())
		return
	}
	_, err = tx.Exec("INSERT INTO company_accounts (username, password, company_id) VALUES ($1, $2, $3) ON CONFLICT (username) DO UPDATE SET password = EXCLUDED.password, company_id = EXCLUDED.company_id", payload.Account.Username, payload.Account.Password, payload.Account.CompanyID)
	if err != nil {
		tx.Rollback()
		respondWithError(w, http.StatusInternalServerError, "Failed to insert account: "+err.Error())
		return
	}
	if err := tx.Commit(); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Transaction commit failed: "+err.Error())
		return
	}
	respondWithJSON(w, http.StatusCreated, payload)
}
func (a *App) updateCompanyAndAccountHandler(w http.ResponseWriter, r *http.Request) {
	username := mux.Vars(r)["username"]
	var payload struct {
		Company     Company `json:"company"`
		NewPassword *string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid payload: "+err.Error())
		return
	}

	tx, err := a.DB.Begin()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer tx.Rollback()

	var targetCompanyId string
	err = tx.QueryRow("SELECT id FROM companies WHERE name = $1", payload.Company.Name).Scan(&targetCompanyId)
	if err == sql.ErrNoRows {
		targetCompanyId = strings.ToLower(strings.ReplaceAll(payload.Company.Name, " ", "_"))
		_, err = tx.Exec("INSERT INTO companies (id, name, role) VALUES ($1, $2, $3)", targetCompanyId, payload.Company.Name, payload.Company.Role)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to create new company: "+err.Error())
			return
		}
	} else if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to find company: "+err.Error())
		return
	}

	if payload.NewPassword != nil && *payload.NewPassword != "" {
		_, err = tx.Exec("UPDATE company_accounts SET company_id = $1, password = $2 WHERE username = $3", targetCompanyId, *payload.NewPassword, username)
	} else {
		_, err = tx.Exec("UPDATE company_accounts SET company_id = $1 WHERE username = $2", targetCompanyId, username)
	}

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update account: "+err.Error())
		return
	}

	if err := tx.Commit(); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Transaction commit failed: "+err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"status": "success"})
}
func (a *App) deleteCompanyAccountHandler(w http.ResponseWriter, r *http.Request) {
	username := mux.Vars(r)["username"]
	res, err := a.DB.Exec("DELETE FROM company_accounts WHERE username = $1", username)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	count, _ := res.RowsAffected()
	if count == 0 {
		respondWithError(w, http.StatusNotFound, "User not found")
		return
	}
	respondWithJSON(w, http.StatusOK, map[string]string{"status": "success"})
}
func (a *App) getAllCompanyAccountsHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := a.DB.Query("SELECT username, password, company_id FROM company_accounts")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()
	var accounts []CompanyAccount
	for rows.Next() {
		var acc CompanyAccount
		if err := rows.Scan(&acc.Username, &acc.Password, &acc.CompanyID); err != nil {
			respondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
		accounts = append(accounts, acc)
	}
	respondWithJSON(w, http.StatusOK, accounts)
}
func (a *App) isUsernameTakenHandler(w http.ResponseWriter, r *http.Request) {
	username := mux.Vars(r)["username"]
	var exists bool
	err := a.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM company_accounts WHERE username = $1)", username).Scan(&exists)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// ---> PERUBAHAN DI SINI <---
	// Jangan kirim 404. Selalu kirim 200 OK dengan jawaban JSON.
	// Flutter akan tahu dari nilai boolean 'exists'.
	respondWithJSON(w, http.StatusOK, map[string]bool{"exists": exists})
}
func (a *App) getAllUserAccountsForDisplayHandler(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT ca.username, ca.company_id, c.name AS company_name, c.role
		FROM company_accounts ca
		JOIN companies c ON ca.company_id = c.id
		WHERE ca.username != 'admin'
		ORDER BY c.name, ca.username`
	rows, err := a.DB.Query(query)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var username, companyId, companyName, role string
		if err := rows.Scan(&username, &companyId, &companyName, &role); err != nil {
			respondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
		results = append(results, map[string]interface{}{
			"username":     username,
			"company_id":   companyId,
			"company_name": companyName,
			"role":         role,
		})
	}
	respondWithJSON(w, http.StatusOK, results)
}
func (a *App) getColleagueAccountsForDisplayHandler(w http.ResponseWriter, r *http.Request) {
	companyName := r.URL.Query().Get("company_name")
	currentUsername := r.URL.Query().Get("current_username")

	query := `
		SELECT ca.username, ca.company_id, c.name AS company_name, c.role
		FROM company_accounts ca
		JOIN companies c ON ca.company_id = c.id
		WHERE c.name = $1 AND ca.username != $2
		ORDER BY ca.username`
	rows, err := a.DB.Query(query, companyName, currentUsername)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var username, companyId, companyNameRes, role string
		if err := rows.Scan(&username, &companyId, &companyNameRes, &role); err != nil {
			respondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
		results = append(results, map[string]interface{}{
			"username":     username,
			"company_id":   companyId,
			"company_name": companyNameRes,
			"role":         role,
		})
	}
	respondWithJSON(w, http.StatusOK, results)
}

func (a *App) searchUsernamesHandler(w http.ResponseWriter, r *http.Request) {
	// 1. Ambil query pencarian dari URL (?q=...)
	query := r.URL.Query().Get("q")
	if query == "" {
		// Jika query kosong, kembalikan list kosong agar tidak membebani server
		respondWithJSON(w, http.StatusOK, []string{})
		return
	}

	// 2. Siapkan SQL query untuk mencari username yang cocok (case-insensitive)
	//    'LIKE' dengan '%' berarti "diawali dengan"
	sqlQuery := "SELECT username FROM company_accounts WHERE username ILIKE $1 LIMIT 10"

	// 3. Eksekusi query ke database
	rows, err := a.DB.Query(sqlQuery, query+"%")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	// 4. Kumpulkan hasil pencarian ke dalam sebuah slice
	var usernames []string
	for rows.Next() {
		var username string
		if err := rows.Scan(&username); err != nil {
			// Jika ada error saat scan, log dan lanjutkan ke baris berikutnya
			log.Printf("Error scanning username: %v", err)
			continue
		}
		usernames = append(usernames, username)
	}

	// 5. Kirim hasilnya sebagai JSON
	respondWithJSON(w, http.StatusOK, usernames)
}
func (a *App) getUniqueCompanyDataForFormHandler(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT id, name, role FROM companies
		WHERE role != 'admin'
		GROUP BY id, name, role ORDER BY name ASC`

	rows, err := a.DB.Query(query)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	var results []map[string]string
	for rows.Next() {
		var id, name, role string
		if err := rows.Scan(&id, &name, &role); err != nil {
			respondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
		results = append(results, map[string]string{"id": id, "name": name, "role": role})
	}
	respondWithJSON(w, http.StatusOK, results)
}
func (a *App) getCompanyByIdHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var c Company
	err := a.DB.QueryRow("SELECT id, name, role FROM companies WHERE id = $1", id).Scan(&c.ID, &c.Name, &c.Role)
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusNotFound, "Company not found")
			return
		}
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondWithJSON(w, http.StatusOK, c)
}
func (a *App) getAllCompaniesHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := a.DB.Query("SELECT id, name, role FROM companies ORDER BY name")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()
	var companies []Company
	for rows.Next() {
		var c Company
		if err := rows.Scan(&c.ID, &c.Name, &c.Role); err != nil {
			respondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
		companies = append(companies, c)
	}
	respondWithJSON(w, http.StatusOK, companies)
}
func (a *App) getCompanyByNameHandler(w http.ResponseWriter, r *http.Request) {
	encodedName := mux.Vars(r)["name"]
	name, err := url.PathUnescape(encodedName)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid name encoding")
		return
	}

	var c Company
	err = a.DB.QueryRow("SELECT id, name, role FROM companies WHERE name = $1", name).Scan(&c.ID, &c.Name, &c.Role)
	if err != nil {
		if err == sql.ErrNoRows {
			// [FIX] Jangan kirim error 404.
			// Kirim status 200 OK dengan body null/kosong.
			// Ini menandakan pencarian sukses, tapi data tidak ada.
			respondWithJSON(w, http.StatusOK, nil)
			return
		}
		// Ini untuk error database yang sesungguhnya.
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	// Jika perusahaan ditemukan, kirim datanya seperti biasa.
	respondWithJSON(w, http.StatusOK, c)
}
func (a *App) getCompaniesByRoleHandler(w http.ResponseWriter, r *http.Request) {
	role := r.URL.Query().Get("role")
	rows, err := a.DB.Query("SELECT id, name, role FROM companies WHERE role = $1 ORDER BY name", role)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()
	var companies []Company
	for rows.Next() {
		var c Company
		if err := rows.Scan(&c.ID, &c.Name, &c.Role); err != nil {
			respondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
		companies = append(companies, c)
	}
	respondWithJSON(w, http.StatusOK, companies)
}

func (a *App) getAllPanelsForDisplayHandler(w http.ResponseWriter, r *http.Request) {
	userRole := r.URL.Query().Get("role")
	companyId := r.URL.Query().Get("company_id")
	var panelIdsSubQuery string
	var args []interface{}
	argCounter := 1

	if userRole == "" {
		userRole = AppRoleAdmin
	}

	if userRole != AppRoleAdmin && userRole != AppRoleViewer {
		switch userRole {
		case AppRoleK3:
			// panelIdsSubQuery = fmt.Sprintf(`SELECT no_pp FROM panels WHERE vendor_id = $%d UNION SELECT panel_no_pp FROM palet WHERE vendor = $%d UNION SELECT panel_no_pp FROM corepart WHERE vendor = $%d`, argCounter, argCounter, argCounter)
			// args = append(args, companyId)
			panelIdsSubQuery = fmt.Sprintf(`
				SELECT no_pp FROM panels WHERE vendor_id = $%d OR vendor_id IS NULL
				UNION
				SELECT panel_no_pp FROM palet WHERE vendor = $%d
				UNION
				SELECT panel_no_pp FROM corepart WHERE vendor = $%d`,
				argCounter, argCounter, argCounter)
			args = append(args, companyId)
		case AppRoleK5:
			panelIdsSubQuery = fmt.Sprintf(`SELECT panel_no_pp FROM busbars WHERE vendor = $%d UNION SELECT no_pp FROM panels WHERE no_pp NOT IN (SELECT DISTINCT panel_no_pp FROM busbars)`, argCounter)
			args = append(args, companyId)
		case AppRoleWarehouse:
			panelIdsSubQuery = fmt.Sprintf(`SELECT panel_no_pp FROM components WHERE vendor = $%d UNION SELECT no_pp FROM panels WHERE no_pp NOT IN (SELECT DISTINCT panel_no_pp FROM components)`, argCounter)
			args = append(args, companyId)
		default:
			respondWithJSON(w, http.StatusOK, []PanelDisplayData{})
			return
		}
	} else {
		panelIdsSubQuery = "SELECT no_pp FROM panels"
	}

	// rawIDs := r.URL.Query().Get("raw_ids") == "true"

	
	finalQuery := fmt.Sprintf(`
		SELECT
			p.no_pp, p.no_panel, p.no_wbs, p.project, p.percent_progress, p.start_date, p.target_delivery,
			p.status_busbar_pcc, p.status_busbar_mcc, p.status_component, p.status_palet,
			p.status_corepart, p.ao_busbar_pcc, p.ao_busbar_mcc, p.created_by, p.vendor_id,
			p.is_closed, p.closed_date, p.panel_type, p.remarks,
			p.close_date_busbar_pcc, p.close_date_busbar_mcc,
			pu.name as panel_vendor_name,
			(SELECT STRING_AGG(c.name, ', ') FROM companies c JOIN busbars b ON c.id = b.vendor WHERE b.panel_no_pp = p.no_pp) as busbar_vendor_names,
			(SELECT STRING_AGG(c.id, ',') FROM companies c JOIN busbars b ON c.id = b.vendor WHERE b.panel_no_pp = p.no_pp) as busbar_vendor_ids,
			COALESCE((
				SELECT json_agg(json_build_object(
					'vendor_name', c.name,
					'remark', b.remarks,
					'vendor_id', b.vendor
				))
				FROM busbars b
				JOIN companies c ON b.vendor = c.id
				WHERE b.panel_no_pp = p.no_pp AND b.remarks IS NOT NULL AND b.remarks != ''
			), '[]'::json) as busbar_remarks,
			(SELECT STRING_AGG(c.name, ', ') FROM companies c JOIN components co ON c.id = co.vendor WHERE co.panel_no_pp = p.no_pp) as component_vendor_names,
			(SELECT STRING_AGG(c.id, ',') FROM companies c JOIN components co ON c.id = co.vendor WHERE co.panel_no_pp = p.no_pp) as component_vendor_ids,
			(SELECT STRING_AGG(c.name, ', ') FROM companies c JOIN palet pa ON c.id = pa.vendor WHERE pa.panel_no_pp = p.no_pp) as palet_vendor_names,
			(SELECT STRING_AGG(c.id, ',') FROM companies c JOIN palet pa ON c.id = pa.vendor WHERE pa.panel_no_pp = p.no_pp) as palet_vendor_ids,
			(SELECT STRING_AGG(c.name, ', ') FROM companies c JOIN corepart cp ON c.id = cp.vendor WHERE cp.panel_no_pp = p.no_pp) as corepart_vendor_names,
			(SELECT STRING_AGG(c.id, ',') FROM companies c JOIN corepart cp ON c.id = cp.vendor WHERE cp.panel_no_pp = p.no_pp) as corepart_vendor_ids
		FROM panels p
		LEFT JOIN companies pu ON p.vendor_id = pu.id
		WHERE p.no_pp IN (`+panelIdsSubQuery+`)
		ORDER BY p.start_date DESC`)


	rows, err := a.DB.Query(finalQuery, args...)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	var results []PanelDisplayData
	for rows.Next() {
		var pdd PanelDisplayData
		var busbarVendorIds, componentVendorIds, paletVendorIds, corepartVendorIds sql.NullString
		
		var panel Panel
		
		err := rows.Scan(
			&panel.NoPp, &panel.NoPanel, &panel.NoWbs, &panel.Project,
			&panel.PercentProgress, &panel.StartDate, &panel.TargetDelivery,
			&panel.StatusBusbarPcc, &panel.StatusBusbarMcc, &panel.StatusComponent,
			&panel.StatusPalet, &panel.StatusCorepart, &panel.AoBusbarPcc,
			&panel.AoBusbarMcc, &panel.CreatedBy, &panel.VendorID,
			&panel.IsClosed, &panel.ClosedDate, &panel.PanelType, &panel.Remarks,
			&panel.CloseDateBusbarPcc, &panel.CloseDateBusbarMcc,
			&pdd.PanelVendorName, &pdd.BusbarVendorNames, &busbarVendorIds, &pdd.BusbarRemarks,
			&pdd.ComponentVendorNames, &componentVendorIds, &pdd.PaletVendorNames, &paletVendorIds,
			&pdd.CorepartVendorNames, &corepartVendorIds,
		)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Error scanning row: "+err.Error())
			return
		}

		
		// if !rawIDs && strings.HasPrefix(panel.NoPp, "TEMP_PP_") {
		// 	panel.NoPp = ""
		// }

		pdd.Panel = panel
		pdd.BusbarVendorIds = splitIds(busbarVendorIds)
		pdd.ComponentVendorIds = splitIds(componentVendorIds)
		pdd.PaletVendorIds = splitIds(paletVendorIds)
		pdd.CorepartVendorIds = splitIds(corepartVendorIds)
		results = append(results, pdd)
	}
	respondWithJSON(w, http.StatusOK, results)
}
func (a *App) getPanelKeysHandler(w http.ResponseWriter, r *http.Request) {
	query := `SELECT no_pp, COALESCE(no_panel, ''), COALESCE(project, ''), COALESCE(no_wbs, '') FROM panels`
	rows, err := a.DB.Query(query)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	var keys []PanelKeyInfo
	for rows.Next() {
		var key PanelKeyInfo
		if err := rows.Scan(&key.NoPp, &key.NoPanel, &key.Project, &key.NoWbs); err != nil {
			respondWithError(w, http.StatusInternalServerError, "Gagal scan panel keys: "+err.Error())
			return
		}
		keys = append(keys, key)
	}
	respondWithJSON(w, http.StatusOK, keys)
}

func (a *App) deletePanelsHandler(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		NoPps []string `json:"no_pps"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid payload: "+err.Error())
		return
	}
	if len(payload.NoPps) == 0 {
		respondWithJSON(w, http.StatusOK, map[string]string{"status": "success", "message": "Tidak ada panel yang dipilih untuk dihapus"})
		return
	}

	query := "DELETE FROM panels WHERE no_pp = ANY($1)"
	result, err := a.DB.Exec(query, pq.Array(payload.NoPps))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Gagal menghapus panel: "+err.Error())
		return
	}
	rowsAffected, _ := result.RowsAffected()
	message := fmt.Sprintf("%d panel berhasil dihapus.", rowsAffected)
	respondWithJSON(w, http.StatusOK, map[string]string{"status": "success", "message": message})
}

func (a *App) deletePanelHandler(w http.ResponseWriter, r *http.Request) {
	noPp := mux.Vars(r)["no_pp"]

	query := "DELETE FROM panels WHERE no_pp = $1"
	result, err := a.DB.Exec(query, noPp)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		respondWithError(w, http.StatusNotFound, "Panel tidak ditemukan")
		return
	}
	respondWithJSON(w, http.StatusOK, map[string]string{"status": "success", "message": "Panel berhasil dihapus"})
}

func (a *App) getAllPanelsHandler(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT
			CASE WHEN no_pp LIKE 'TEMP_PP_%' THEN '' ELSE no_pp END AS no_pp,
			no_panel, no_wbs, project, percent_progress, start_date, target_delivery,
			status_busbar_pcc, status_busbar_mcc, status_component, status_palet,
			status_corepart, ao_busbar_pcc, ao_busbar_mcc, created_by, vendor_id,
			is_closed, closed_date, panel_type
		FROM panels`

	rows, err := a.DB.Query(query)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()
	var panels []Panel
	for rows.Next() {
		var p Panel
		if err := rows.Scan(&p.NoPp, &p.NoPanel, &p.NoWbs, &p.Project, &p.PercentProgress, &p.StartDate, &p.TargetDelivery, &p.StatusBusbarPcc, &p.StatusBusbarMcc, &p.StatusComponent, &p.StatusPalet, &p.StatusCorepart, &p.AoBusbarPcc, &p.AoBusbarMcc, &p.CreatedBy, &p.VendorID, &p.IsClosed, &p.ClosedDate, &p.PanelType); err != nil {
			respondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
		panels = append(panels, p)
	}
	respondWithJSON(w, http.StatusOK, panels)
}

func (a *App) getPanelByNoPpHandler(w http.ResponseWriter, r *http.Request) {
	noPp := mux.Vars(r)["no_pp"]
	var p Panel

	query := `
		SELECT
			CASE WHEN no_pp LIKE 'TEMP_PP_%' THEN '' ELSE no_pp END AS no_pp,
			no_panel, no_wbs, project, percent_progress, start_date, target_delivery,
			status_busbar_pcc, status_busbar_mcc, status_component, status_palet,
			status_corepart, ao_busbar_pcc, ao_busbar_mcc, created_by, vendor_id,
			is_closed, closed_date, panel_type
		FROM panels WHERE no_pp = $1`

	err := a.DB.QueryRow(query, noPp).Scan(
		&p.NoPp, &p.NoPanel, &p.NoWbs, &p.Project, &p.PercentProgress, &p.StartDate, &p.TargetDelivery, &p.StatusBusbarPcc, &p.StatusBusbarMcc, &p.StatusComponent, &p.StatusPalet, &p.StatusCorepart, &p.AoBusbarPcc, &p.AoBusbarMcc, &p.CreatedBy, &p.VendorID, &p.IsClosed, &p.ClosedDate, &p.PanelType)

	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusNotFound, "Panel tidak ditemukan")
			return
		}
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondWithJSON(w, http.StatusOK, p)
}
func (a *App) isNoPpTakenHandler(w http.ResponseWriter, r *http.Request) {
	noPp := mux.Vars(r)["no_pp"]
	var exists bool
	err := a.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM panels WHERE no_pp = $1)", noPp).Scan(&exists)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	// [FIX] Always respond with 200 OK and a clear JSON body.
	// This single line handles both cases (exists or not).
	respondWithJSON(w, http.StatusOK, map[string]bool{"exists": exists})
}

func (a *App) changePanelNoPpHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	oldNoPp, ok := vars["old_no_pp"]
	if !ok {
		respondWithError(w, http.StatusBadRequest, "No. PP lama tidak ditemukan di URL")
		return
	}

	var payloadData Panel
	if err := json.NewDecoder(r.Body).Decode(&payloadData); err != nil {
		if err.Error() == "EOF" {
			respondWithError(w, http.StatusBadRequest, "Payload tidak valid: Body request kosong.")
		} else {
			respondWithError(w, http.StatusBadRequest, "Payload tidak valid: "+err.Error())
		}
		return
	}

	tx, err := a.DB.Begin()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Gagal memulai transaksi: "+err.Error())
		return
	}
	defer tx.Rollback()

	// 1. Ambil data panel yang ada sekarang dari DB
	var existingPanel Panel
	querySelect := `
		SELECT no_pp, no_panel, no_wbs, project, percent_progress, start_date, target_delivery,
		status_busbar_pcc, status_busbar_mcc, status_component, status_palet, status_corepart,
		ao_busbar_pcc, ao_busbar_mcc, created_by, vendor_id, is_closed, closed_date, panel_type, remarks,
		close_date_busbar_pcc, close_date_busbar_mcc
		FROM panels WHERE no_pp = $1`
	err = tx.QueryRow(querySelect, oldNoPp).Scan(
		&existingPanel.NoPp, &existingPanel.NoPanel, &existingPanel.NoWbs, &existingPanel.Project,
		&existingPanel.PercentProgress, &existingPanel.StartDate, &existingPanel.TargetDelivery,
		&existingPanel.StatusBusbarPcc, &existingPanel.StatusBusbarMcc, &existingPanel.StatusComponent,
		&existingPanel.StatusPalet, &existingPanel.StatusCorepart, &existingPanel.AoBusbarPcc,
		&existingPanel.AoBusbarMcc, &existingPanel.CreatedBy, &existingPanel.VendorID,
		&existingPanel.IsClosed, &existingPanel.ClosedDate, &existingPanel.PanelType, &existingPanel.Remarks,
		&existingPanel.CloseDateBusbarPcc, &existingPanel.CloseDateBusbarMcc,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusNotFound, "Panel dengan No. PP lama tidak ditemukan")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Gagal mengambil data panel: "+err.Error())
		return
	}

	// 2. Timpa data lama dengan perubahan dari payload (jika ada)
	if payloadData.NoPanel != nil { existingPanel.NoPanel = payloadData.NoPanel }
	if payloadData.NoWbs != nil { existingPanel.NoWbs = payloadData.NoWbs }
	if payloadData.Project != nil { existingPanel.Project = payloadData.Project }
	if payloadData.PercentProgress != nil { existingPanel.PercentProgress = payloadData.PercentProgress }
	if payloadData.StartDate != nil { existingPanel.StartDate = payloadData.StartDate }
	if payloadData.TargetDelivery != nil { existingPanel.TargetDelivery = payloadData.TargetDelivery }
	if payloadData.StatusBusbarPcc != nil { existingPanel.StatusBusbarPcc = payloadData.StatusBusbarPcc }
	if payloadData.StatusBusbarMcc != nil { existingPanel.StatusBusbarMcc = payloadData.StatusBusbarMcc }
	if payloadData.StatusComponent != nil { existingPanel.StatusComponent = payloadData.StatusComponent }
	if payloadData.StatusPalet != nil { existingPanel.StatusPalet = payloadData.StatusPalet }
	if payloadData.StatusCorepart != nil { existingPanel.StatusCorepart = payloadData.StatusCorepart }
	if payloadData.AoBusbarPcc != nil { existingPanel.AoBusbarPcc = payloadData.AoBusbarPcc }
	if payloadData.AoBusbarMcc != nil { existingPanel.AoBusbarMcc = payloadData.AoBusbarMcc }
	if payloadData.VendorID != nil { existingPanel.VendorID = payloadData.VendorID }
	if payloadData.PanelType != nil { existingPanel.PanelType = payloadData.PanelType }
	if payloadData.Remarks != nil { existingPanel.Remarks = payloadData.Remarks } 
	existingPanel.CloseDateBusbarPcc = payloadData.CloseDateBusbarPcc
	existingPanel.CloseDateBusbarMcc = payloadData.CloseDateBusbarMcc
	existingPanel.IsClosed = payloadData.IsClosed
	existingPanel.ClosedDate = payloadData.ClosedDate
	existingPanel.NoPp = payloadData.NoPp

	// 3. Logika untuk mengubah Primary Key
	newNoPp := existingPanel.NoPp
	pkToUpdateWith := oldNoPp 

	if newNoPp != oldNoPp {
		if newNoPp == "" && !strings.HasPrefix(oldNoPp, "TEMP_PP_") {
			timestamp := time.Now().UnixNano() / int64(time.Millisecond)
			generatedTempNoPp := fmt.Sprintf("TEMP_PP_%d", timestamp)
			_, err = tx.Exec("UPDATE panels SET no_pp = $1 WHERE no_pp = $2", generatedTempNoPp, oldNoPp)
			if err != nil {
				respondWithError(w, http.StatusInternalServerError, "Gagal demote PK ke TEMP: "+err.Error())
				return
			}
			pkToUpdateWith = generatedTempNoPp
		} else if newNoPp != "" {
			_, err = tx.Exec("UPDATE panels SET no_pp = $1 WHERE no_pp = $2", newNoPp, oldNoPp)
			if err != nil {
				if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
					respondWithError(w, http.StatusConflict, fmt.Sprintf("No. PP '%s' sudah digunakan.", newNoPp))
					return
				}
				respondWithError(w, http.StatusInternalServerError, "Gagal update PK ke permanen: "+err.Error())
				return
			}
			pkToUpdateWith = newNoPp
		}
	}

	// 4. Lakukan satu kali UPDATE final dengan data yang sudah lengkap
	updateQuery := `
		UPDATE panels SET
			no_panel = $1, no_wbs = $2, project = $3, percent_progress = $4,
			start_date = $5, target_delivery = $6, status_busbar_pcc = $7,
			status_busbar_mcc = $8, status_component = $9, status_palet = $10,
			status_corepart = $11, ao_busbar_pcc = $12, ao_busbar_mcc = $13,
			vendor_id = $14, is_closed = $15, closed_date = $16, panel_type = $17, remarks = $18,
			close_date_busbar_pcc = $19, close_date_busbar_mcc = $20
	WHERE no_pp = $21`

	_, err = tx.Exec(updateQuery,
		existingPanel.NoPanel, existingPanel.NoWbs, existingPanel.Project, existingPanel.PercentProgress,
		existingPanel.StartDate, existingPanel.TargetDelivery, existingPanel.StatusBusbarPcc,
		existingPanel.StatusBusbarMcc, existingPanel.StatusComponent, existingPanel.StatusPalet,
		existingPanel.StatusCorepart, existingPanel.AoBusbarPcc, existingPanel.AoBusbarMcc,
		existingPanel.VendorID, existingPanel.IsClosed, existingPanel.ClosedDate, existingPanel.PanelType,
		existingPanel.Remarks, 
		existingPanel.CloseDateBusbarPcc, existingPanel.CloseDateBusbarMcc,
		pkToUpdateWith,
	)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Gagal update detail panel: "+err.Error())
		return
	}

	if err := tx.Commit(); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Gagal commit transaksi: "+err.Error())
		return
	}
	
	existingPanel.NoPp = pkToUpdateWith
	respondWithJSON(w, http.StatusOK, existingPanel)
}

func (a *App) upsertPanelRemarkHandler(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PanelNoPp string `json:"panel_no_pp"`
		Remarks   string `json:"remarks"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid payload")
		return
	}
	query := `UPDATE panels SET remarks = $1 WHERE no_pp = $2`
	_, err := a.DB.Exec(query, payload.Remarks, payload.PanelNoPp)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondWithJSON(w, http.StatusCreated, payload)
}

func (a *App) isPanelNumberUniqueHandler(w http.ResponseWriter, r *http.Request) {
	noPanel := mux.Vars(r)["no_panel"]
	currentNoPp := r.URL.Query().Get("current_no_pp")
	var query string
	var args []interface{}
	if currentNoPp != "" {
		query = "SELECT EXISTS(SELECT 1 FROM panels WHERE no_panel = $1 AND no_pp != $2)"
		args = append(args, noPanel, currentNoPp)
	} else {
		query = "SELECT EXISTS(SELECT 1 FROM panels WHERE no_panel = $1)"
		args = append(args, noPanel)
	}
	var exists bool
	err := a.DB.QueryRow(query, args...).Scan(&exists)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if exists {
		respondWithJSON(w, http.StatusOK, map[string]bool{"is_unique": false})
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}// main.go

// Fungsi generik untuk menghapus relasi panel-vendor
func (a *App) deleteGenericRelationHandler(tableName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			PanelNoPp string `json:"panel_no_pp"`
			Vendor    string `json:"vendor"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid payload: "+err.Error())
			return
		}

		if payload.PanelNoPp == "" || payload.Vendor == "" {
			respondWithError(w, http.StatusBadRequest, "panel_no_pp dan vendor wajib diisi")
			return
		}

		// Query dibangun secara dinamis berdasarkan nama tabel
		query := fmt.Sprintf("DELETE FROM %s WHERE panel_no_pp = $1 AND vendor = $2", tableName)
		result, err := a.DB.Exec(query, payload.PanelNoPp, payload.Vendor)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Gagal menghapus relasi %s: %v", tableName, err))
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			respondWithJSON(w, http.StatusOK, map[string]string{"status": "success", "message": "Tidak ada relasi yang cocok untuk dihapus"})
			return
		}

		respondWithJSON(w, http.StatusOK, map[string]string{"status": "success"})
	}
}

func (a *App) upsertGenericHandler(tableName string, model interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(model); err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid payload")
			return
		}
		var panelNoPp, vendor string
		switch v := model.(type) {
		case *Busbar:
			panelNoPp = v.PanelNoPp
			vendor = v.Vendor
		case *Component:
			panelNoPp = v.PanelNoPp
			vendor = v.Vendor
		case *Palet:
			panelNoPp = v.PanelNoPp
			vendor = v.Vendor
		case *Corepart:
			panelNoPp = v.PanelNoPp
			vendor = v.Vendor
		default:
			respondWithError(w, http.StatusInternalServerError, "Unsupported model type")
			return
		}
		query := "INSERT INTO " + tableName + " (panel_no_pp, vendor) VALUES ($1, $2) ON CONFLICT (panel_no_pp, vendor) DO NOTHING"
		_, err := a.DB.Exec(query, panelNoPp, vendor)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
		respondWithJSON(w, http.StatusCreated, model)
	}
}
func (a *App) deleteBusbarHandler(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PanelNoPp string `json:"panel_no_pp"`
		Vendor    string `json:"vendor"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid payload: "+err.Error())
		return
	}

	if payload.PanelNoPp == "" || payload.Vendor == "" {
		respondWithError(w, http.StatusBadRequest, "panel_no_pp and vendor are required")
		return
	}

	query := "DELETE FROM busbars WHERE panel_no_pp = $1 AND vendor = $2"
	result, err := a.DB.Exec(query, payload.PanelNoPp, payload.Vendor)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Gagal menghapus relasi busbar: "+err.Error())
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		// Ini bukan error, mungkin data sudah terhapus sebelumnya, jadi kembalikan sukses
		respondWithJSON(w, http.StatusOK, map[string]string{"status": "success", "message": "No matching busbar relation found to delete"})
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

func (a *App) upsertBusbarRemarkandVendorHandler(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PanelNoPp string `json:"panel_no_pp"`
		Vendor    string `json:"vendor"`
		Remarks   string `json:"remarks"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid payload")
		return
	}
	query := `
		INSERT INTO busbars (panel_no_pp, vendor, remarks) VALUES ($1, $2, $3)
		ON CONFLICT (panel_no_pp, vendor) DO UPDATE SET remarks = EXCLUDED.remarks`
	_, err := a.DB.Exec(query, payload.PanelNoPp, payload.Vendor, payload.Remarks)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondWithJSON(w, http.StatusCreated, payload)
}
func (a *App) upsertStatusAOK5(w http.ResponseWriter, r *http.Request) {
	// Definisikan struct untuk payload JSON yang dikirim oleh klien (role K5)
	var payload struct {
		PanelNoPp       string      `json:"panel_no_pp"`
		VendorID        string      `json:"vendor"` // Field ini ada di payload tapi tidak digunakan untuk update DB
		AoBusbarPcc     *customTime `json:"ao_busbar_pcc"`
		AoBusbarMcc     *customTime `json:"ao_busbar_mcc"`
		StatusBusbarPcc *string     `json:"status_busbar_pcc"`
		StatusBusbarMcc *string     `json:"status_busbar_mcc"`
	}

	// Decode body request ke dalam struct payload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid payload: "+err.Error())
		return
	}

	// Validasi dasar bahwa No. PP harus ada
	if payload.PanelNoPp == "" {
		respondWithError(w, http.StatusBadRequest, "panel_no_pp is required")
		return
	}

	// Siapkan untuk membangun query UPDATE secara dinamis
	var updates []string
	var args []interface{}
	argCounter := 1

	// Tambahkan field ke query hanya jika nilainya tidak nil di payload
	
	// if payload.VendorID != "" {
	// 	updates = append(updates, fmt.Sprintf("vendor_id = $%d", argCounter))
	// 	args = append(args, payload.VendorID)
	// 	argCounter++	
	// }
	if payload.StatusBusbarPcc != nil {
		updates = append(updates, fmt.Sprintf("status_busbar_pcc = $%d", argCounter))
		args = append(args, *payload.StatusBusbarPcc)
		argCounter++
	}
	if payload.StatusBusbarMcc != nil {
		updates = append(updates, fmt.Sprintf("status_busbar_mcc = $%d", argCounter))
		args = append(args, *payload.StatusBusbarMcc)
		argCounter++
	}
	if payload.AoBusbarPcc != nil {
		updates = append(updates, fmt.Sprintf("ao_busbar_pcc = $%d", argCounter))
		args = append(args, payload.AoBusbarPcc)
		argCounter++
	}
	if payload.AoBusbarMcc != nil {
		updates = append(updates, fmt.Sprintf("ao_busbar_mcc = $%d", argCounter))
		args = append(args, payload.AoBusbarMcc)
		argCounter++
	}

	// Jika tidak ada field yang perlu diupdate, kirim respon sukses
	if len(updates) == 0 {
		respondWithJSON(w, http.StatusOK, map[string]string{"message": "No fields to update."})
		return
	}

	// Finalisasi query dengan klausa WHERE dan eksekusi
	query := fmt.Sprintf("UPDATE panels SET %s WHERE no_pp = $%d", strings.Join(updates, ", "), argCounter)
	args = append(args, payload.PanelNoPp)

	result, err := a.DB.Exec(query, args...)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update panel status for K5: "+err.Error())
		return
	}

	// Cek apakah ada baris yang berhasil diupdate
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		respondWithError(w, http.StatusNotFound, "Panel with the given no_pp not found.")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

func (a *App) upsertStatusWHS(w http.ResponseWriter, r *http.Request) {
	// Definisikan struct untuk payload JSON yang dikirim oleh klien (role Warehouse)
	var payload struct {
		PanelNoPp       string  `json:"panel_no_pp"`
		VendorID        string  `json:"vendor"` // Field ini ada di payload tapi tidak digunakan untuk update DB
		StatusComponent *string `json:"status_component"`
	}

	// Decode body request ke dalam struct payload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid payload: "+err.Error())
		return
	}

	// Validasi dasar
	if payload.PanelNoPp == "" {
		respondWithError(w, http.StatusBadRequest, "panel_no_pp is required")
		return
	}
	if payload.StatusComponent == nil {
		respondWithError(w, http.StatusBadRequest, "status_component is required")
		return
	}

	// Query UPDATE yang sederhana dan langsung
	query := `UPDATE panels SET status_component = $1 WHERE no_pp = $2`

	result, err := a.DB.Exec(query, *payload.StatusComponent, payload.PanelNoPp)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update panel component status: "+err.Error())
		return
	}

	// Cek apakah ada baris yang berhasil diupdate
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		respondWithError(w, http.StatusNotFound, "Panel with the given no_pp not found.")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"status": "success"})
}
func (a *App) getAllGenericHandler(tableName string, modelFactory func() interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var query string
		if tableName == "busbars" {
			query = "SELECT id, panel_no_pp, vendor, remarks FROM " + tableName
		} else {
			query = "SELECT id, panel_no_pp, vendor FROM " + tableName
		}

		rows, err := a.DB.Query(query)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer rows.Close()

		var results []interface{}
		for rows.Next() {
			model := modelFactory()
			switch m := model.(type) {
			case *Busbar:
				if err := rows.Scan(&m.ID, &m.PanelNoPp, &m.Vendor, &m.Remarks); err != nil {
					log.Println(err)
					continue
				}
			case *Component:
				if err := rows.Scan(&m.ID, &m.PanelNoPp, &m.Vendor); err != nil {
					log.Println(err)
					continue
				}
			case *Palet:
				if err := rows.Scan(&m.ID, &m.PanelNoPp, &m.Vendor); err != nil {
					log.Println(err)
					continue
				}
			case *Corepart:
				if err := rows.Scan(&m.ID, &m.PanelNoPp, &m.Vendor); err != nil {
					log.Println(err)
					continue
				}
			}
			results = append(results, model)
		}
		respondWithJSON(w, http.StatusOK, results)
	}
}
func (a *App) updateCompanyHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var payload struct {
		Name string `json:"name"`
		Role string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid payload: "+err.Error())
		return
	}
	if payload.Name == "" {
		respondWithError(w, http.StatusBadRequest, "Nama perusahaan tidak boleh kosong")
		return
	}

	query := `UPDATE companies SET name = $1, role = $2 WHERE id = $3`
	_, err := a.DB.Exec(query, payload.Name, payload.Role, id)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" { // unique_violation
			respondWithError(w, http.StatusConflict, "Nama perusahaan '"+payload.Name+"' sudah digunakan.")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Gagal memperbarui perusahaan: "+err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"status": "success"})
}
// file: main.go

func (a *App) getFilteredDataForExport(r *http.Request) (map[string]interface{}, error) {
	queryParams := r.URL.Query()
	result := make(map[string]interface{})

	tx, err := a.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// [PERBAIKAN] Query dasar sekarang menggunakan JOIN untuk filter vendor
	panelQuery := `
		SELECT p.* FROM panels p
		LEFT JOIN busbars b ON p.no_pp = b.panel_no_pp
		LEFT JOIN components c ON p.no_pp = c.panel_no_pp
		LEFT JOIN palet pa ON p.no_pp = pa.panel_no_pp
		LEFT JOIN corepart cp ON p.no_pp = cp.panel_no_pp
		WHERE 1=1
	`
	args := []interface{}{}
	argCounter := 1

	// -- Helper Functions --
	addDateFilter := func(query, col, start, end string) (string, []interface{}) {
		// ... (fungsi ini tidak berubah)
		localArgs := []interface{}{}
		if start != "" {
			query += fmt.Sprintf(" AND %s >= $%d", col, argCounter)
			localArgs = append(localArgs, start)
			argCounter++
		}
		if end != "" {
			endTime, err := time.Parse(time.RFC3339, end)
			if err == nil {
				end = endTime.Add(24 * time.Hour).Format(time.RFC3339)
				query += fmt.Sprintf(" AND %s < $%d", col, argCounter)
				localArgs = append(localArgs, end)
				argCounter++
			}
		}
		return query, localArgs
	}

	addListFilter := func(query, col, values string) (string, []interface{}) {
		// ... (fungsi ini tidak berubah)
		localArgs := []interface{}{}
		if values != "" {
			list := strings.Split(values, ",")
			if len(list) > 0 {
				// Khusus untuk 'Belum Diatur'
				if len(list) == 1 && list[0] == "Belum Diatur" {
					query += fmt.Sprintf(" AND (%s IS NULL OR %s = '')", col, col)
				} else {
					hasBelumDiatur := false
					var filteredList []string
					for _, item := range list {
						if item == "Belum Diatur" {
							hasBelumDiatur = true
						} else {
							filteredList = append(filteredList, item)
						}
					}
					
					var condition string
					if len(filteredList) > 0 {
						condition = fmt.Sprintf("%s = ANY($%d)", col, argCounter)
						localArgs = append(localArgs, pq.Array(filteredList))
						argCounter++
					}

					if hasBelumDiatur {
						if condition != "" {
							condition = fmt.Sprintf("(%s OR %s IS NULL OR %s = '')", condition, col, col)
						} else {
							condition = fmt.Sprintf("(%s IS NULL OR %s = '')", col, col)
						}
					}
					query += " AND " + condition
				}
			}
		}
		return query, localArgs
	}

	addVendorFilter := func(query, col, values string) (string, []interface{}) {
		localArgs := []interface{}{}
		if values != "" {
			list := strings.Split(values, ",")
			if len(list) > 0 {
				query += fmt.Sprintf(" AND %s IN (SELECT unnest($%d::text[]))", col, argCounter)
				localArgs = append(localArgs, pq.Array(list))
				argCounter++
			}
		}
		return query, localArgs
	}

	// -- Menerapkan Semua Filter --
	var newArgs []interface{}
	
	// Filter Tanggal
	panelQuery, newArgs = addDateFilter(panelQuery, "p.start_date", queryParams.Get("start_date_start"), queryParams.Get("start_date_end"))
	args = append(args, newArgs...)
	panelQuery, newArgs = addDateFilter(panelQuery, "p.target_delivery", queryParams.Get("delivery_date_start"), queryParams.Get("delivery_date_end"))
	args = append(args, newArgs...)
	panelQuery, newArgs = addDateFilter(panelQuery, "p.closed_date", queryParams.Get("closed_date_start"), queryParams.Get("closed_date_end"))
	args = append(args, newArgs...)

	// Filter Tipe & Status
	panelQuery, newArgs = addListFilter(panelQuery, "p.panel_type", queryParams.Get("panel_types"))
	args = append(args, newArgs...)
	panelQuery, newArgs = addListFilter(panelQuery, "p.status_busbar_pcc", queryParams.Get("pcc_statuses"))
	args = append(args, newArgs...)
	panelQuery, newArgs = addListFilter(panelQuery, "p.status_busbar_mcc", queryParams.Get("mcc_statuses"))
	args = append(args, newArgs...)
	panelQuery, newArgs = addListFilter(panelQuery, "p.status_component", queryParams.Get("component_statuses"))
	args = append(args, newArgs...)
	panelQuery, newArgs = addListFilter(panelQuery, "p.status_palet", queryParams.Get("palet_statuses"))
	args = append(args, newArgs...)
	panelQuery, newArgs = addListFilter(panelQuery, "p.status_corepart", queryParams.Get("corepart_statuses"))
	args = append(args, newArgs...)
	
	// Filter Vendor (menggunakan JOIN)
	panelQuery, newArgs = addVendorFilter(panelQuery, "p.vendor_id", queryParams.Get("panel_vendors"))
	args = append(args, newArgs...)
	panelQuery, newArgs = addVendorFilter(panelQuery, "b.vendor", queryParams.Get("busbar_vendors"))
	args = append(args, newArgs...)
	panelQuery, newArgs = addVendorFilter(panelQuery, "c.vendor", queryParams.Get("component_vendors"))
	args = append(args, newArgs...)
	panelQuery, newArgs = addVendorFilter(panelQuery, "pa.vendor", queryParams.Get("palet_vendors"))
	args = append(args, newArgs...)
	panelQuery, newArgs = addVendorFilter(panelQuery, "cp.vendor", queryParams.Get("corepart_vendors"))
	args = append(args, newArgs...)
	
	// Filter Status Panel berdasarkan persentase
	if statuses := queryParams.Get("panel_statuses"); statuses != "" {
		statusList := strings.Split(statuses, ",")
		var statusConditions []string
		for _, status := range statusList {
			switch status {
			case "progressRed":
				statusConditions = append(statusConditions, "(p.percent_progress < 50)")
			case "progressOrange":
				statusConditions = append(statusConditions, "(p.percent_progress >= 50 AND p.percent_progress < 75)")
			case "progressBlue":
				statusConditions = append(statusConditions, "(p.percent_progress >= 75 AND p.percent_progress < 100)")
			case "readyToDelivery":
				statusConditions = append(statusConditions, "(p.percent_progress >= 100 AND p.is_closed = false)")
			case "closed":
				statusConditions = append(statusConditions, "(p.is_closed = true)")
			}
		}
		if len(statusConditions) > 0 {
			panelQuery += " AND (" + strings.Join(statusConditions, " OR ") + ")"
		}
	}
	
	// Filter Arsip
	if includeArchived, err := strconv.ParseBool(queryParams.Get("include_archived")); err == nil && !includeArchived {
		panelQuery += " AND (p.is_closed = false OR p.closed_date > NOW() - INTERVAL '2 days')"
	}


	// Akhiri query dengan GROUP BY untuk menghilangkan duplikat karena JOIN
	panelQuery += " GROUP BY p.no_pp"
	
	// Eksekusi query
	rows, err := tx.Query(panelQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("gagal query panel: %w", err)
	}

	// ... (Sisa fungsi untuk mem-parsing hasil dan mengambil data terkait tidak berubah)
	var panels []Panel
	panelIdSet := make(map[string]bool)
	for rows.Next() {
		var p Panel
		if err := rows.Scan(&p.NoPp, &p.NoPanel, &p.NoWbs, &p.Project, &p.PercentProgress, &p.StartDate, &p.TargetDelivery, &p.StatusBusbarPcc, &p.StatusBusbarMcc, &p.StatusComponent, &p.StatusPalet, &p.StatusCorepart, &p.AoBusbarPcc, &p.AoBusbarMcc, &p.CreatedBy, &p.VendorID, &p.IsClosed, &p.ClosedDate, &p.PanelType); err != nil {
			log.Printf("Peringatan: Gagal scan panel row: %v", err)
			continue
		}
		panels = append(panels, p)
		panelIdSet[p.NoPp] = true
	}
	rows.Close()
	result["panels"] = panels

	relevantPanelIds := []string{}
	for id := range panelIdSet {
		relevantPanelIds = append(relevantPanelIds, id)
	}

	if len(relevantPanelIds) > 0 {
		result["busbars"], _ = fetchInAs(tx, "busbars", "panel_no_pp", relevantPanelIds, func() interface{} { return &Busbar{} })
		result["components"], _ = fetchInAs(tx, "components", "panel_no_pp", relevantPanelIds, func() interface{} { return &Component{} })
		result["palet"], _ = fetchInAs(tx, "palet", "panel_no_pp", relevantPanelIds, func() interface{} { return &Palet{} })
		result["corepart"], _ = fetchInAs(tx, "corepart", "panel_no_pp", relevantPanelIds, func() interface{} { return &Corepart{} })
	} else {
		result["busbars"], result["components"], result["palet"], result["corepart"] = []Busbar{}, []Component{}, []Palet{}, []Corepart{}
	}

	result["companies"], _ = fetchAllAs(tx, "companies", func() interface{} { return &Company{} })
	result["companyAccounts"], _ = fetchAllAs(tx, "company_accounts", func() interface{} { return &CompanyAccount{} })

	return result, nil
}

func (a *App) getFilteredDataForExportHandler(w http.ResponseWriter, r *http.Request) {
	data, err := a.getFilteredDataForExport(r) 
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondWithJSON(w, http.StatusOK, data)
}
// file: main.go

func (a *App) generateCustomExportJsonHandler(w http.ResponseWriter, r *http.Request) {
	// Ambil parameter spesifik untuk export custom
	includePanels, _ := strconv.ParseBool(r.URL.Query().Get("panels"))
	includeUsers, _ := strconv.ParseBool(r.URL.Query().Get("users"))

	// [PERBAIKAN KUNCI]
	// Panggil fungsi getFilteredDataForExport dengan seluruh request 'r'.
	// Fungsi ini sekarang akan membaca semua parameter filter (tanggal, vendor, status, dll) dari URL.
	data, err := a.getFilteredDataForExport(r)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Mengambil semua data 'companies' untuk mapping ID ke Nama
	// Ini tidak perlu difilter karena kita butuh semua perusahaan sebagai referensi
	allCompaniesResult, err := fetchAllAs(a.DB, "companies", func() interface{} { return &Company{} })
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Gagal mengambil data companies: "+err.Error())
		return
	}
	allCompanies := allCompaniesResult.([]interface{})

	companyMap := make(map[string]Company)
	for _, c := range allCompanies {
		company := c.(*Company)
		companyMap[company.ID] = *company
	}

	// Siapkan JSON final yang akan dikirim
	jsonData := make(map[string]interface{})

	// Logika internal untuk membangun struktur JSON tidak berubah.
	// Sekarang ia akan otomatis bekerja dengan data panel yang sudah terfilter.
	if includePanels {
		panels := data["panels"].([]Panel) // Konversi ke slice of Panel
		busbars := data["busbars"].([]interface{})
		panelBusbarMap := make(map[string]string)
		for _, b := range busbars {
			busbar := b.(*Busbar)
			if company, ok := companyMap[busbar.Vendor]; ok {
				if existing, ok := panelBusbarMap[busbar.PanelNoPp]; ok {
					panelBusbarMap[busbar.PanelNoPp] = existing + ", " + company.Name
				} else {
					panelBusbarMap[busbar.PanelNoPp] = company.Name
				}
			}
		}

		var panelData []map[string]interface{}
		for _, panel := range panels { // Langsung iterate dari slice of Panel
			panelVendorName := ""
			if panel.VendorID != nil {
				if company, ok := companyMap[*panel.VendorID]; ok {
					panelVendorName = company.Name
				}
			}
			panelData = append(panelData, map[string]interface{}{
				"PP Panel":                 panel.NoPp,
				"Panel No":                 panel.NoPanel,
				"WBS":                      panel.NoWbs,
				"PROJECT":                  panel.Project,
				"Plan Start":               panel.StartDate,
				"Actual Delivery ke SEC":   panel.TargetDelivery,
				"Panel":                    panelVendorName,
				"Busbar":                   panelBusbarMap[panel.NoPp],
				"Progres Panel":            panel.PercentProgress,
				"Status Corepart":          panel.StatusCorepart,
				"Status Palet":             panel.StatusPalet,
				"Status Busbar PCC":        panel.StatusBusbarPcc,
				"Status Busbar MCC":        panel.StatusBusbarMcc,
				"AO Busbar PCC":            panel.AoBusbarPcc,
				"AO Busbar MCC":            panel.AoBusbarMcc,
			})
		}
		jsonData["panel_data"] = panelData
	}

	if includeUsers {
		accounts := data["companyAccounts"].([]interface{})
		var userData []map[string]interface{}
		for _, acc := range accounts {
			account := acc.(*CompanyAccount)
			companyName := ""
			companyRole := ""
			if company, ok := companyMap[account.CompanyID]; ok {
				companyName = company.Name
				companyRole = company.Role
			}
			userData = append(userData, map[string]interface{}{
				"Username":     account.Username,
				"Password":     account.Password,
				"Company":      companyName,
				"Company Role": companyRole,
			})
		}
		jsonData["user_data"] = userData
	}

	respondWithJSON(w, http.StatusOK, jsonData)
}
// file: main.go

func (a *App) generateFilteredDatabaseJsonHandler(w http.ResponseWriter, r *http.Request) {
	// Ambil parameter tabel mana saja yang mau di-export
	tablesParam := r.URL.Query().Get("tables")
	tablesToInclude := strings.Split(tablesParam, ",")

	// [PERBAIKAN KUNCI]
	// Panggil fungsi getFilteredDataForExport dengan seluruh request 'r'.
	// Ini akan mengembalikan data yang sudah difilter berdasarkan parameter URL
	// seperti tanggal, vendor, status, dll.
	data, err := a.getFilteredDataForExport(r)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Buat map JSON final
	jsonData := make(map[string]interface{})
	
	// Logika untuk memilih tabel mana yang akan dimasukkan ke JSON final tidak berubah.
	// Sekarang ia akan bekerja dengan data yang sudah terfilter.
	for _, table := range tablesToInclude {
		// Normalisasi nama tabel untuk pencocokan yang lebih fleksibel
		normalizedTable := strings.ToLower(strings.ReplaceAll(table, " ", ""))
		switch normalizedTable {
		case "companies":
			jsonData["companies"] = data["companies"]
		case "companyaccounts":
			// Nama tabel di JSON adalah 'company_accounts'
			jsonData["company_accounts"] = data["companyAccounts"]
		case "panels":
			jsonData["panels"] = data["panels"]
		case "busbars":
			jsonData["busbars"] = data["busbars"]
		case "components":
			jsonData["components"] = data["components"]
		case "palet":
			jsonData["palet"] = data["palet"]
		case "corepart":
			jsonData["corepart"] = data["corepart"]
		}
	}
	
	respondWithJSON(w, http.StatusOK, jsonData)
}
func (a *App) importDataHandler(w http.ResponseWriter, r *http.Request) {
	var data map[string][]map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid JSON payload: "+err.Error())
		return
	}

	tx, err := a.DB.Begin()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer tx.Rollback()

	tableOrder := []string{"companies", "company_accounts", "panels", "busbars", "components", "palet", "corepart"}
	for _, tableName := range tableOrder {
		if items, ok := data[tableName]; ok {
			for _, itemData := range items {
				cleanMapData(itemData)
				if err := insertMap(tx, tableName, itemData); err != nil {
					respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to import to %s: %v", tableName, err))
					return
				}
			}
		}
	}

	if err := tx.Commit(); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Transaction commit failed: "+err.Error())
		return
	}
	respondWithJSON(w, http.StatusCreated, map[string]string{"status": "success", "message": "Data imported successfully"})
}
func (a *App) generateImportTemplateHandler(w http.ResponseWriter, r *http.Request) {
	dataType := r.URL.Query().Get("dataType")
	format := r.URL.Query().Get("format")

	var fileBytes []byte
	var extension string
	var err error

	if format == "json" {
		extension = "json"
		jsonData := generateJsonTemplate(dataType)
		fileBytes, err = json.MarshalIndent(jsonData, "", "  ")
	} else {
		extension = "xlsx"
		excelFile := generateExcelTemplate(dataType)
		buffer, err_excel := excelFile.WriteToBuffer()
		if err_excel == nil {
			fileBytes = buffer.Bytes()
		} else {
			err = err_excel
		}
	}

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to generate template file: "+err.Error())
		return
	}

	encodedBytes := base64.StdEncoding.EncodeToString(fileBytes)
	respondWithJSON(w, http.StatusOK, map[string]string{
		"bytes":     encodedBytes,
		"extension": extension,
	})
}


// normalizeVendorName membersihkan nama vendor ke format dasar untuk pencocokan.
// Contoh: "PT. G.A.A 2" -> "GAA"
func normalizeVendorName(name string) string {
	// 1. Hapus Prefiks Umum (PT, CV, etc.)
	prefixes := []string{"PT.", "PT", "CV.", "CV", "UD.", "UD"}
	tempName := strings.ToUpper(name)
	for _, prefix := range prefixes {
		if strings.HasPrefix(tempName, prefix+" ") {
			tempName = strings.TrimSpace(strings.TrimPrefix(tempName, prefix+" "))
			break
		}
	}

	// 2. Hapus semua karakter non-alfabet
	var result strings.Builder
	for _, r := range tempName {
		if r >= 'A' && r <= 'Z' {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// generateAcronym membuat akronim dari nama vendor.
// Contoh: "PT Global Abc Acd" -> "GAA"
func generateAcronym(name string) string {
	// 1. Hapus Prefiks
	prefixes := []string{"PT.", "PT", "CV.", "CV", "UD.", "UD"}
	tempName := strings.ToUpper(name)
	for _, prefix := range prefixes {
		if strings.HasPrefix(tempName, prefix+" ") {
			tempName = strings.TrimSpace(strings.TrimPrefix(tempName, prefix+" "))
			break
		}
	}

	// 2. Buat Akronim dari sisa kata
	parts := strings.Fields(tempName) // Memisahkan berdasarkan spasi
	if len(parts) <= 1 {
		return "" // Bukan nama multi-kata, tidak ada akronim
	}
	var acronym strings.Builder
	for _, part := range parts {
		if len(part) > 0 {
			acronym.WriteRune([]rune(part)[0])
		}
	}

	// Hanya kembalikan jika benar-benar sebuah akronim (2+ huruf)
	if acronym.Len() > 1 {
		return acronym.String()
	}
	return ""
}

func (a *App) importFromCustomTemplateHandler(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Data             map[string][]map[string]interface{} `json:"data"`
		LoggedInUsername *string                           `json:"loggedInUsername"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid payload: "+err.Error())
		return
	}

	tx, err := a.DB.Begin()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer tx.Rollback()

	// --- [MODIFIKASI] Ambil role dan company ID dari user yang login ---
	var loggedInUserCompanyId string
	var loggedInUserRole string
	if payload.LoggedInUsername != nil && *payload.LoggedInUsername != "" {
		query := `SELECT c.id, c.role FROM companies c JOIN company_accounts ca ON c.id = ca.company_id WHERE ca.username = $1`
		err := tx.QueryRow(query, *payload.LoggedInUsername).Scan(&loggedInUserCompanyId, &loggedInUserRole)
		if err != nil && err != sql.ErrNoRows {
			// Ini adalah error database sungguhan, bukan user tidak ditemukan
			respondWithError(w, http.StatusInternalServerError, "Gagal mengambil data user: "+err.Error())
			return
		}
	}

	var errors []string
	dataProcessed := false

	// --- Pre-computation & Caching Vendor Names ---
	vendorIdMap := make(map[string]bool)
	normalizedNameToIdMap := make(map[string]string)
	acronymToIdMap := make(map[string]string)

	rows, err := tx.Query("SELECT id, name FROM companies")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Gagal mengambil data companies: "+err.Error())
		return
	}
	defer rows.Close()

	for rows.Next() {
		var c Company
		if err := rows.Scan(&c.ID, &c.Name); err != nil {
			respondWithError(w, http.StatusInternalServerError, "Gagal scan company: "+err.Error())
			return
		}
		vendorIdMap[c.ID] = true
		normalized := normalizeVendorName(c.Name)
		if normalized != "" {
			normalizedNameToIdMap[normalized] = c.ID
		}
		acronym := generateAcronym(c.Name)
		if acronym != "" {
			acronymToIdMap[acronym] = c.ID
		}
	}

	resolveVendor := func(vendorInput string) string {
		trimmedInput := strings.TrimSpace(vendorInput)
		if trimmedInput == "" {
			return ""
		}
		if vendorIdMap[trimmedInput] {
			return trimmedInput
		}
		normalizedInput := normalizeVendorName(trimmedInput)
		if id, ok := normalizedNameToIdMap[normalizedInput]; ok {
			return id
		}
		if id, ok := acronymToIdMap[normalizedInput]; ok {
			return id
		}
		acronymInput := generateAcronym(trimmedInput)
		if id, ok := acronymToIdMap[acronymInput]; ok {
			return id
		}
		return ""
	}

	if userSheetData, ok := payload.Data["user"]; ok {
		for i, row := range userSheetData {
			cleanMapData(row)
			rowNum := i + 2
			username := getValueCaseInsensitive(row, "Username")
			companyName := getValueCaseInsensitive(row, "Company")
			companyRole := strings.ToLower(getValueCaseInsensitive(row, "Company Role"))
			password := getValueCaseInsensitive(row, "Password")
			if password == "" {
				password = "123"
			}

			if username == "" {
				errors = append(errors, fmt.Sprintf("User baris %d: Kolom 'Username' wajib diisi.", rowNum))
				continue
			}
			if companyName == "" {
				errors = append(errors, fmt.Sprintf("User baris %d: Kolom 'Company' wajib diisi.", rowNum))
				continue
			}
			var companyId string
			err := tx.QueryRow("SELECT id FROM companies WHERE LOWER(name) = LOWER($1)", strings.TrimSpace(companyName)).Scan(&companyId)
			if err == sql.ErrNoRows {
				companyId = strings.ToLower(strings.ReplaceAll(companyName, " ", "_"))
				_, errInsert := tx.Exec("INSERT INTO companies (id, name, role) VALUES ($1, $2, $3) ON CONFLICT(id) DO NOTHING", companyId, companyName, companyRole)
				if errInsert != nil {
					errors = append(errors, fmt.Sprintf("User baris %d: Gagal membuat company baru '%s': %v", rowNum, companyName, errInsert))
					continue
				}
			} else if err != nil {
				errors = append(errors, fmt.Sprintf("User baris %d: Error mencari company: %v", rowNum, err))
				continue
			}
			_, err = tx.Exec("INSERT INTO company_accounts (username, password, company_id) VALUES ($1, $2, $3) ON CONFLICT (username) DO UPDATE SET password = EXCLUDED.password, company_id = EXCLUDED.company_id", username, password, companyId)
			if err != nil {
				errors = append(errors, fmt.Sprintf("User baris %d: Gagal memasukkan/update akun '%s': %v", rowNum, username, err))
				continue
			}
			dataProcessed = true
		}
	}

	if panelSheetData, ok := payload.Data["panel"]; ok {
		// [PERBAIKAN 1] Logika baru untuk memastikan Warehouse selalu ada
		var warehouseCompanyId string
		err := tx.QueryRow("SELECT id FROM companies WHERE name = 'Warehouse'").Scan(&warehouseCompanyId)
		if err == sql.ErrNoRows {
			// Jika tidak ada, buat baru
			warehouseCompanyId = "warehouse" // ID default
			_, errInsert := tx.Exec(
				"INSERT INTO companies (id, name, role) VALUES ($1, $2, $3) ON CONFLICT(id) DO NOTHING",
				warehouseCompanyId,
				"Warehouse",
				AppRoleWarehouse,
			)
			if errInsert != nil {
				errors = append(errors, fmt.Sprintf("Kritis: Gagal membuat company 'Warehouse' secara otomatis: %v", errInsert))
			}
		} else if err != nil {
			errors = append(errors, fmt.Sprintf("Kritis: Gagal mencari company 'Warehouse': %v", err))
		}

		for i, row := range panelSheetData {
			cleanMapData(row)
			rowNum := i + 2
			noPpRaw := getValueCaseInsensitive(row, "PP Panel")
			noPanel := getValueCaseInsensitive(row, "Panel No")
			project := getValueCaseInsensitive(row, "PROJECT")
			noWbs := getValueCaseInsensitive(row, "WBS")

			var noPp string
			if f, err := strconv.ParseFloat(noPpRaw, 64); err == nil {
				noPp = fmt.Sprintf("%d", int(f))
			} else {
				noPp = noPpRaw
			}

			var oldTempNoPp string
			if noPp != "" && (noPanel != "" || project != "" || noWbs != "") {
				findTempQuery := `SELECT no_pp FROM panels WHERE no_pp LIKE 'TEMP_PP_%' AND no_panel = $1 AND project = $2 AND no_wbs = $3`
				err := tx.QueryRow(findTempQuery, noPanel, project, noWbs).Scan(&oldTempNoPp)
				if err != nil && err != sql.ErrNoRows {
					errors = append(errors, fmt.Sprintf("Panel baris %d: Gagal mencari data sementara: %v", rowNum, err))
					continue
				}

				if oldTempNoPp != "" {
					_, err := tx.Exec("DELETE FROM panels WHERE no_pp = $1", oldTempNoPp)
					if err != nil {
						errors = append(errors, fmt.Sprintf("Panel baris %d: Gagal menghapus data sementara %s: %v", rowNum, oldTempNoPp, err))
						continue
					}
				}
			}

			if noPp == "" {
				timestamp := time.Now().UnixNano() / int64(time.Millisecond)
				noPp = fmt.Sprintf("TEMP_PP_%d_%d", rowNum, timestamp)
			}

			// --- [MODIFIKASI] Logika baru untuk vendor Panel dan Busbar berdasarkan role ---
			var panelVendorId, busbarVendorId sql.NullString

			if loggedInUserRole == AppRoleK3 {
				// LOGIKA KHUSUS UNTUK K3
				// 1. Panel Vendor otomatis diisi dengan company K3 yang login
				panelVendorId = sql.NullString{String: loggedInUserCompanyId, Valid: true}
				// 2. Kolom Busbar diabaikan, busbarVendorId dibiarkan null (tidak valid)
			} else {
				// LOGIKA STANDAR UNTUK ROLE LAIN
				panelVendorInput := getValueCaseInsensitive(row, "Panel")
				busbarVendorInput := getValueCaseInsensitive(row, "Busbar")

				if panelVendorInput != "" {
					resolvedId := resolveVendor(panelVendorInput)
					if resolvedId == "" {
						errors = append(errors, fmt.Sprintf("Panel baris %d: Vendor Panel '%s' tidak ditemukan.", rowNum, panelVendorInput))
					} else {
						panelVendorId = sql.NullString{String: resolvedId, Valid: true}
					}
				}

				if busbarVendorInput != "" {
					resolvedId := resolveVendor(busbarVendorInput)
					if resolvedId == "" {
						errors = append(errors, fmt.Sprintf("Panel baris %d: Vendor Busbar '%s' tidak ditemukan.", rowNum, busbarVendorInput))
					} else {
						busbarVendorId = sql.NullString{String: resolvedId, Valid: true}
					}
				}
			}
			// --- AKHIR MODIFIKASI ---

			lastErrorIndex := len(errors) - 1
			if lastErrorIndex >= 0 && strings.Contains(errors[lastErrorIndex], fmt.Sprintf("baris %d", rowNum)) {
				continue
			}

			actualDeliveryDate := parseDate(getValueCaseInsensitive(row, "Actual Delivery ke SEC"))
			var progress float64 = 0.0
			// Jika tanggal delivery ada, set progress ke 100
			if actualDeliveryDate != nil {
				progress = 100.0
			}

			panelMap := map[string]interface{}{
				"no_pp":             noPp,
				"no_panel":          getValueCaseInsensitive(row, "Panel No"),
				"no_wbs":            getValueCaseInsensitive(row, "WBS"),
				"project":           getValueCaseInsensitive(row, "PROJECT"),
				"target_delivery":   parseDate(getValueCaseInsensitive(row, "Plan Start")),
				"closed_date":       actualDeliveryDate,
				"is_closed":         actualDeliveryDate != nil, // is_closed juga bergantung pada tanggal ini
				"vendor_id":         panelVendorId,
				"created_by":        "import",
				"percent_progress":  progress, // [PERBAIKAN] Gunakan variabel progress yang sudah dihitung
				"status_busbar_pcc": "On Progress",
				"status_busbar_mcc": "On Progress",
				"status_component":  "Open",
				"status_palet":      "Open",
				"status_corepart":   "Open",
			}

			if payload.LoggedInUsername != nil {
				panelMap["created_by"] = *payload.LoggedInUsername
			}

			if err := insertMap(tx, "panels", panelMap); err != nil {
				errors = append(errors, fmt.Sprintf("Panel baris %d (%s): Gagal menyimpan: %v", rowNum, noPp, err))
				continue
			}
			if panelVendorId.Valid {
				if err := insertMap(tx, "palet", map[string]interface{}{"panel_no_pp": noPp, "vendor": panelVendorId.String}); err != nil {
					errors = append(errors, fmt.Sprintf("Panel baris %d: Gagal link palet: %v", rowNum, err))
				}
				if err := insertMap(tx, "corepart", map[string]interface{}{"panel_no_pp": noPp, "vendor": panelVendorId.String}); err != nil {
					errors = append(errors, fmt.Sprintf("Panel baris %d: Gagal link corepart: %v", rowNum, err))
				}
			}
			if warehouseCompanyId != "" {
				if err := insertMap(tx, "components", map[string]interface{}{"panel_no_pp": noPp, "vendor": warehouseCompanyId}); err != nil {
					errors = append(errors, fmt.Sprintf("Panel baris %d: Gagal link component: %v", rowNum, err))
				}
			}
			if busbarVendorId.Valid {
				if err := insertMap(tx, "busbars", map[string]interface{}{"panel_no_pp": noPp, "vendor": busbarVendorId.String}); err != nil {
					errors = append(errors, fmt.Sprintf("Panel baris %d: Gagal link busbar: %v", rowNum, err))
				}
			}

			dataProcessed = true
		}
	}
	if len(errors) > 0 {
		log.Printf("Import validation error details: %s", strings.Join(errors, " | "))
		finalMessage := "Impor dibatalkan karena error berikut:\n- " + strings.Join(errors, "\n- ")
		respondWithError(w, http.StatusBadRequest, finalMessage)
		return
	}

	if !dataProcessed {
		respondWithError(w, http.StatusBadRequest, "Tidak ada data valid yang ditemukan di dalam sheet 'Panel' atau 'User'.")
		return
	}

	if err := tx.Commit(); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Transaction commit failed: "+err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"message": "Impor berhasil diselesaikan! 🎉"})
}

// getOrCreateChatByPanel is a helper function to find or create a chat for a panel.
func (a *App) getOrCreateChatByPanel(panelNoPp string, tx *sql.Tx) (int, error) {
	var chatID int
	// Check if chat exists
	err := tx.QueryRow("SELECT id FROM chats WHERE panel_no_pp = $1", panelNoPp).Scan(&chatID)
	if err == sql.ErrNoRows {
		// Chat does not exist, create it
		err = tx.QueryRow("INSERT INTO chats (panel_no_pp) VALUES ($1) RETURNING id", panelNoPp).Scan(&chatID)
		if err != nil {
			return 0, fmt.Errorf("failed to create chat: %w", err)
		}
	} else if err != nil {
		return 0, fmt.Errorf("failed to query chat: %w", err)
	}
	return chatID, nil
}
func (a *App) createIssueForPanelHandler(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
	panelNoPp, ok := vars["no_pp"]
	if !ok {
		respondWithError(w, http.StatusBadRequest, "Panel No PP is required")
		return
	}
	var payload struct {
		Description string   `json:"issue_description"`
		Title       string   `json:"issue_title"`
		CreatedBy   string   `json:"created_by"`
		Photos      []string `json:"photos"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload: "+err.Error())
		return
	}
	if payload.Title == "" {
		respondWithError(w, http.StatusBadRequest, "Root cause tidak boleh kosong")
		return
	}
	tx, err := a.DB.Begin()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to start transaction")
		return
	}
	defer tx.Rollback()
	chatID, err := a.getOrCreateChatByPanel(panelNoPp, tx)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	initialLog := Logs{
		{Action: "membuat issue", User: payload.CreatedBy, Timestamp: time.Now()},
	}
	var issueID int
	query := `INSERT INTO issues (chat_id, title, description, created_by, logs, status)
			VALUES ($1, $2, $3, $4, $5, 'unsolved') RETURNING id`
	err = tx.QueryRow(query, chatID, payload.Title, payload.Description, payload.CreatedBy, initialLog).Scan(&issueID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create issue: "+err.Error())
		return
	}
	for _, photoBase64 := range payload.Photos {
		_, err := tx.Exec("INSERT INTO photos (issue_id, photo_data) VALUES ($1, $2)", issueID, photoBase64)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to save photo: "+err.Error())
			return
		}
	}

	commentText := fmt.Sprintf(payload.Title)
	if payload.Description != "" {
		commentText = fmt.Sprintf("%s: %s", payload.Title, payload.Description)
	}

	var imageUrls []string
	for _, photoBase64 := range payload.Photos {
		parts := strings.Split(photoBase64, ",")
		if len(parts) != 2 { continue }
		imgData, err := base64.StdEncoding.DecodeString(parts[1])
		if err != nil { continue }
		filename := fmt.Sprintf("%s.jpg", uuid.New().String())
		filepath := fmt.Sprintf("uploads/%s", filename)
		err = os.WriteFile(filepath, imgData, 0644)
		if err != nil { continue }
		imageUrls = append(imageUrls, "/"+filepath)
	}
	imageUrlsJSON, _ := json.Marshal(imageUrls)
	newCommentID := uuid.New().String()

	// --- ▼▼▼ PERUBAHAN DI SINI ▼▼▼ ---
	// Tambahkan is_system_comment ke query
	commentQuery := `
		INSERT INTO issue_comments (id, issue_id, sender_id, text, image_urls, is_system_comment)
		VALUES ($1, $2, $3, $4, $5, TRUE)` // Set nilainya menjadi TRUE
	_, err = tx.Exec(commentQuery, newCommentID, issueID, payload.CreatedBy, commentText, imageUrlsJSON)
	// --- ▲▲▲ AKHIR PERUBAHAN ▲▲▲
	
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create initial comment: "+err.Error())
		return
	}

	if err := tx.Commit(); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to commit transaction")
		return
	}
	respondWithJSON(w, http.StatusCreated, map[string]int{"issue_id": issueID})
}
func (a *App) getIssuesByPanelHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	panelNoPp := vars["no_pp"]

	var chatID int
	err := a.DB.QueryRow("SELECT id FROM chats WHERE panel_no_pp = $1", panelNoPp).Scan(&chatID)
	if err == sql.ErrNoRows {
		respondWithJSON(w, http.StatusOK, []IssueWithPhotos{}) // Return empty list
		return
	}
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to find chat for panel: "+err.Error())
		return
	}

	// Query utama untuk mendapatkan semua isu
	rows, err := a.DB.Query("SELECT id, chat_id, title, description, status, logs, created_by, created_at, updated_at FROM issues WHERE chat_id = $1 ORDER BY created_at DESC", chatID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	// Gunakan map untuk mapping issue ID ke issue-nya
	issueMap := make(map[int]*IssueWithPhotos)
	var issueIDs []int

	for rows.Next() {
		var issue Issue
		if err := rows.Scan(&issue.ID, &issue.ChatID, &issue.Title, &issue.Description, &issue.Status, &issue.Logs, &issue.CreatedBy, &issue.CreatedAt, &issue.UpdatedAt); err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to scan issue: "+err.Error())
			return
		}
		issueIDs = append(issueIDs, issue.ID)
		issueMap[issue.ID] = &IssueWithPhotos{Issue: issue, Photos: []Photo{}} // Inisialisasi dengan slice foto kosong
	}
	
	if len(issueIDs) == 0 {
		respondWithJSON(w, http.StatusOK, []IssueWithPhotos{})
		return
	}

	// Query kedua untuk mendapatkan semua foto yang relevan sekaligus
	photoRows, err := a.DB.Query("SELECT id, issue_id, photo_data FROM photos WHERE issue_id = ANY($1)", pq.Array(issueIDs))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to retrieve photos: "+err.Error())
		return
	}
	defer photoRows.Close()

	// Gabungkan foto ke dalam map isu
	for photoRows.Next() {
		var p Photo
		if err := photoRows.Scan(&p.ID, &p.IssueID, &p.PhotoData); err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to scan photo: "+err.Error())
			return
		}
		if issue, ok := issueMap[p.IssueID]; ok {
			issue.Photos = append(issue.Photos, p)
		}
	}

	// Konversi map kembali ke slice untuk respons JSON
	var issuesWithPhotos []IssueWithPhotos
	for _, id := range issueIDs { // Iterasi sesuai urutan asli
		issuesWithPhotos = append(issuesWithPhotos, *issueMap[id])
	}
	
	respondWithJSON(w, http.StatusOK, issuesWithPhotos)
}
// getIssueByIDHandler retrieves a single issue with its photos.
func (a *App) getIssueByIDHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid issue ID")
		return
	}

	var issue Issue
	// ▼▼▼ FIX: REMOVED "issue_type" FROM THE QUERY ▼▼▼
	err = a.DB.QueryRow("SELECT id, chat_id, title, description, status, logs, created_by, created_at, updated_at FROM issues WHERE id = $1", id).Scan(&issue.ID, &issue.ChatID, &issue.Title, &issue.Description, &issue.Status, &issue.Logs, &issue.CreatedBy, &issue.CreatedAt, &issue.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusNotFound, "Issue not found")
		} else {
			respondWithError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	rows, err := a.DB.Query("SELECT id, issue_id, photo_data FROM photos WHERE issue_id = $1", id)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to retrieve photos: "+err.Error())
		return
	}
	defer rows.Close()

	var photos []Photo
	for rows.Next() {
		var p Photo
		if err := rows.Scan(&p.ID, &p.IssueID, &p.PhotoData); err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to scan photo: "+err.Error())
			return
		}
		photos = append(photos, p)
	}

	response := IssueWithPhotos{Issue: issue, Photos: photos}
	respondWithJSON(w, http.StatusOK, response)
}
func (a *App) updateIssueHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	issueID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid issue ID")
		return
	}
	var payload struct {
		Description string `json:"issue_description"`
		Title       string `json:"issue_title"`
		Status      string `json:"issue_status"`
		UpdatedBy   string `json:"updated_by"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid payload: "+err.Error())
		return
	}
	if payload.Title == "" {
		respondWithError(w, http.StatusBadRequest, "Tipe Masalah tidak boleh kosong")
		return
	}

	// --- ▼▼▼ PERUBAHAN DIMULAI DI SINI ▼▼▼ ---
	tx, err := a.DB.Begin()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Gagal memulai transaksi")
		return
	}
	defer tx.Rollback()

	var currentLogs Logs
	var currentStatus string
	err = tx.QueryRow("SELECT logs, status FROM issues WHERE id = $1", issueID).Scan(&currentLogs, &currentStatus)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Issue not found or failed to get logs: "+err.Error())
		return
	}
	
	logAction := "mengubah issue"
	if payload.Status != currentStatus {
		if payload.Status == "solved" {
			logAction = "menandai solved"
		} else if payload.Status == "unsolved" {
			logAction = "membuka kembali issue"
		}
	}
	newLogEntry := LogEntry{Action: logAction, User: payload.UpdatedBy, Timestamp: time.Now()}
	updatedLogs := append(currentLogs, newLogEntry)

	// 1. Update tabel 'issues' seperti biasa
	query := `UPDATE issues SET title = $1, description = $2, status = $3, logs = $4 WHERE id = $5`
	res, err := tx.Exec(query, payload.Title, payload.Description, payload.Status, updatedLogs, issueID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update issue: "+err.Error())
		return
	}
	count, _ := res.RowsAffected()
	if count == 0 {
		respondWithError(w, http.StatusNotFound, "Issue not found")
		return
	}

	// 2. Buat teks komentar yang baru
	updatedCommentText := fmt.Sprintf("**%s**", payload.Title)
	if payload.Description != "" {
		updatedCommentText = fmt.Sprintf("**%s**: %s", payload.Title, payload.Description)
	}

	// 3. Update komentar sistem yang sesuai
	// Kita juga set is_edited menjadi true untuk menandakan ada perubahan
	commentQuery := `
		UPDATE issue_comments 
		SET text = $1, is_edited = TRUE
		WHERE issue_id = $2 AND is_system_comment = TRUE`
	_, err = tx.Exec(commentQuery, updatedCommentText, issueID)
	if err != nil {
		// Transaksi akan di-rollback jika ini gagal
		respondWithError(w, http.StatusInternalServerError, "Failed to update system comment: "+err.Error())
		return
	}

	// 4. Commit transaksi jika semua berhasil
	if err := tx.Commit(); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Gagal commit transaksi")
		return
	}
	
	// --- ▲▲▲ AKHIR PERUBAHAN ▲▲▲

	respondWithJSON(w, http.StatusOK, map[string]string{"status": "success"})
}
// deleteIssueHandler deletes an issue and its associated photos.
func (a *App) deleteIssueHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid issue ID")
		return
	}

	res, err := a.DB.Exec("DELETE FROM issues WHERE id = $1", id)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	count, _ := res.RowsAffected()
	if count == 0 {
		respondWithError(w, http.StatusNotFound, "Issue not found")
		return
	}
	respondWithJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// addPhotoToIssueHandler adds a new photo to an issue.
func (a *App) addPhotoToIssueHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	issueID, err := strconv.Atoi(vars["issue_id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid issue ID")
		return
	}

	var payload struct {
		PhotoData string `json:"photo"` // Base64 string
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid payload: "+err.Error())
		return
	}

	var photoID int
	err = a.DB.QueryRow("INSERT INTO photos (issue_id, photo_data) VALUES ($1, $2) RETURNING id", issueID, payload.PhotoData).Scan(&photoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to save photo: "+err.Error())
		return
	}
	respondWithJSON(w, http.StatusCreated, map[string]int{"photo_id": photoID})
}

// deletePhotoHandler removes a photo.
func (a *App) deletePhotoHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid photo ID")
		return
	}

	res, err := a.DB.Exec("DELETE FROM photos WHERE id = $1", id)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	count, _ := res.RowsAffected()
	if count == 0 {
		respondWithError(w, http.StatusNotFound, "Photo not found")
		return
	}
	respondWithJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func cleanMapData(data map[string]interface{}) {
	for key, value := range data {
		// Cek apakah nilainya adalah string
		if s, ok := value.(string); ok {
			trimmedS := strings.TrimSpace(s)
			if trimmedS == "" {
				// Jika stringnya kosong setelah di-trim, hapus dari map
				delete(data, key)
			} else {
				// Jika tidak, update map dengan nilai yang sudah bersih
				data[key] = trimmedS
			}
		}
	}
}

// =============================================================================
// HELPERS & DB INIT
// =============================================================================
func jsonContentTypeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		next.ServeHTTP(w, r)
	})
}
func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}
func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.WriteHeader(code)
	w.Write(response)
}
func splitIds(ns sql.NullString) []string {
	if !ns.Valid || ns.String == "" {
		return []string{}
	}
	return strings.Split(ns.String, ",")
}
func initDB(db *sql.DB) {
	createTablesSQL := `
	CREATE TABLE IF NOT EXISTS companies ( id TEXT PRIMARY KEY, name TEXT UNIQUE NOT NULL, role TEXT NOT NULL );
	CREATE TABLE IF NOT EXISTS company_accounts ( username TEXT PRIMARY KEY, password TEXT, company_id TEXT REFERENCES companies(id) ON DELETE CASCADE );
	CREATE TABLE IF NOT EXISTS panels (
		no_pp TEXT PRIMARY KEY, 
		no_panel TEXT, 
		no_wbs TEXT, 
		project TEXT, 
		percent_progress REAL,
		start_date TIMESTAMPTZ, 
		target_delivery TIMESTAMPTZ, 
		status_busbar_pcc TEXT, 
		status_busbar_mcc TEXT,
		status_component TEXT, 
		status_palet TEXT, 
		status_corepart TEXT, 
		ao_busbar_pcc TIMESTAMPTZ, 
		ao_busbar_mcc TIMESTAMPTZ,
		created_by TEXT, 
		vendor_id TEXT, 
		is_closed BOOLEAN DEFAULT false, 
		closed_date TIMESTAMPTZ
	);
	CREATE TABLE IF NOT EXISTS busbars ( id SERIAL PRIMARY KEY, panel_no_pp TEXT NOT NULL REFERENCES panels(no_pp) ON DELETE CASCADE ON UPDATE CASCADE, vendor TEXT NOT NULL, remarks TEXT, UNIQUE(panel_no_pp, vendor) );
	CREATE TABLE IF NOT EXISTS components ( id SERIAL PRIMARY KEY, panel_no_pp TEXT NOT NULL REFERENCES panels(no_pp) ON DELETE CASCADE ON UPDATE CASCADE, vendor TEXT NOT NULL, UNIQUE(panel_no_pp, vendor) );
	CREATE TABLE IF NOT EXISTS palet ( id SERIAL PRIMARY KEY, panel_no_pp TEXT NOT NULL REFERENCES panels(no_pp) ON DELETE CASCADE ON UPDATE CASCADE, vendor TEXT NOT NULL, UNIQUE(panel_no_pp, vendor) );
	CREATE TABLE IF NOT EXISTS corepart ( id SERIAL PRIMARY KEY, panel_no_pp TEXT NOT NULL REFERENCES panels(no_pp) ON DELETE CASCADE ON UPDATE CASCADE, vendor TEXT NOT NULL, UNIQUE(panel_no_pp, vendor) );
	`
	if _, err := db.Exec(createTablesSQL); err != nil {
		log.Fatalf("Gagal membuat tabel awal: %v", err)
	}

	createTablesSQLChats:= `
	CREATE TABLE IF NOT EXISTS chats (
		id SERIAL PRIMARY KEY,
		panel_no_pp VARCHAR(255) UNIQUE NOT NULL REFERENCES panels(no_pp) ON DELETE CASCADE,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`
	if _, err := db.Exec(createTablesSQLChats); err != nil {
		log.Fatalf("Gagal membuat tabel chats: %v", err)
	}

	createTablesSQLIssues:= `
	CREATE TABLE IF NOT EXISTS issues (
		id SERIAL PRIMARY KEY,
		chat_id INT NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
		title VARCHAR(255) NOT NULL,
		description TEXT,
		status VARCHAR(50) NOT NULL DEFAULT 'unsolved',
		logs JSONB,
		created_by VARCHAR(255),
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`
	if _, err := db.Exec(createTablesSQLIssues); err != nil {
		log.Fatalf("Gagal membuat tabel issues: %v", err)
	}

	createIssueTitlesTableSQL := `
	CREATE TABLE IF NOT EXISTS issue_titles (
		id SERIAL PRIMARY KEY,
		title TEXT UNIQUE NOT NULL
	);`
	if _, err := db.Exec(createIssueTitlesTableSQL); err != nil {
		log.Fatalf("Gagal membuat tabel issue_titles: %v", err)
	}

	// Tambahkan beberapa data awal jika tabel baru dibuat
	var count_titles int
	if err := db.QueryRow("SELECT COUNT(*) FROM issue_titles").Scan(&count_titles); err == nil && count_titles == 0 {
		log.Println("Tabel issue_titles kosong, menambahkan data awal...")
		initialTitles := []string{"Kerusakan Komponen", "Masalah Instalasi", "Error Software", "Lainnya"}
		for _, title := range initialTitles {
			db.Exec("INSERT INTO issue_titles (title) VALUES ($1) ON CONFLICT (title) DO NOTHING", title)
		}
	}

	createIssueCommentsTableSQL := `
	CREATE TABLE IF NOT EXISTS issue_comments (
		id TEXT PRIMARY KEY,
		issue_id INT NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
		sender_id TEXT NOT NULL REFERENCES company_accounts(username) ON DELETE CASCADE,
		text TEXT,
		timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		reply_to_comment_id TEXT REFERENCES issue_comments(id) ON DELETE SET NULL,
		reply_to_user_id TEXT REFERENCES company_accounts(username) ON DELETE SET NULL,
		is_edited BOOLEAN DEFAULT false,
		image_urls JSONB
	);
	`
	if _, err := db.Exec(createIssueCommentsTableSQL); err != nil {
		log.Fatalf("Gagal membuat tabel issue_comments: %v", err)
	}

	// Buat folder 'uploads' jika belum ada untuk menyimpan gambar
	if _, err := os.Stat("uploads"); os.IsNotExist(err) {
		os.Mkdir("uploads", 0755)
		log.Println("Folder 'uploads' berhasil dibuat.")
	}
	updateCommentConstraintSQL := `
	DO $$
	DECLARE
		constraint_name TEXT;
	BEGIN
		-- Cari nama constraint foreign key yang ada saat ini
		SELECT conname INTO constraint_name
		FROM pg_constraint
		WHERE conrelid = 'issue_comments'::regclass
		AND confrelid = 'issue_comments'::regclass
		AND contype = 'f';

		-- Jika constraint ditemukan, hapus dan buat yang baru dengan ON DELETE CASCADE
		IF constraint_name IS NOT NULL THEN
			EXECUTE 'ALTER TABLE issue_comments DROP CONSTRAINT ' || quote_ident(constraint_name);
			ALTER TABLE issue_comments
				ADD CONSTRAINT issue_comments_reply_to_comment_id_fkey
				FOREIGN KEY (reply_to_comment_id)
				REFERENCES issue_comments(id)
				ON DELETE CASCADE; -- <--- INI BAGIAN PENTINGNYA
		END IF;
	END;
	$$;
	`
	if _, err := db.Exec(updateCommentConstraintSQL); err != nil {
		log.Fatalf("Gagal menjalankan migrasi constraint untuk issue_comments: %v", err)
	}
	alterCommentsTableSQL := `
	DO $$
	BEGIN
		IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'issue_comments' AND column_name = 'is_system_comment') THEN
			ALTER TABLE issue_comments ADD COLUMN is_system_comment BOOLEAN DEFAULT false;
		END IF;
	END;
	$$;
	`
	if _, err := db.Exec(alterCommentsTableSQL); err != nil {
		log.Fatalf("Gagal menjalankan migrasi untuk kolom is_system_comment: %v", err)
	}
	
	alterIssuesTableSQL := `
	DO $$
	BEGIN
		IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'issues' AND column_name = 'issue_type') THEN
			ALTER TABLE issues DROP COLUMN issue_type;
		END IF;
	END;
	$$;
	`
	if _, err := db.Exec(alterIssuesTableSQL); err != nil {
		log.Fatalf("Gagal menjalankan migrasi untuk menghapus kolom issue_type: %v", err)
	}

	createTablesSQLPhotos:= `
	CREATE TABLE IF NOT EXISTS photos (
		id SERIAL PRIMARY KEY,
		issue_id INT NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
		photo_data TEXT NOT NULL
	);`
	if _, err := db.Exec(createTablesSQLPhotos); err != nil {
		log.Fatalf("Gagal membuat tabel photos: %v", err)
	}

	updateIssuesTrigger:= `
	CREATE OR REPLACE FUNCTION update_updated_at_column()
	RETURNS TRIGGER AS $$
	BEGIN
		NEW.updated_at = NOW();
		RETURN NEW;
	END;
	$$ language 'plpgsql';

	DROP TRIGGER IF EXISTS update_issues_updated_at ON issues;
	CREATE TRIGGER update_issues_updated_at
	BEFORE UPDATE ON issues
	FOR EACH ROW
	EXECUTE FUNCTION update_updated_at_column();
	`
	if _, err := db.Exec(updateIssuesTrigger); err != nil {
		log.Fatalf("Gagal membuat trigger updated_at di tabel issues: %v", err)
	}

	alterTableSQL := `
	DO $$
	BEGIN
		IF EXISTS (
			SELECT 1 FROM pg_constraint
			WHERE conname = 'panels_no_panel_key' AND conrelid = 'panels'::regclass
		) THEN
			ALTER TABLE panels DROP CONSTRAINT panels_no_panel_key;
		END IF;
	END;
	$$;
	`
	if _, err := db.Exec(alterTableSQL); err != nil {
		log.Fatalf("Gagal mengubah constraint tabel panels: %v", err)
	}

	alterTableAddPanelTypeSQL := `
	DO $$
	BEGIN
		IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'panels' AND column_name = 'panel_type') THEN
			ALTER TABLE panels ADD COLUMN panel_type TEXT;
		END IF;
	END;
	$$;
	`
	if _, err := db.Exec(alterTableAddPanelTypeSQL); err != nil {
		log.Fatalf("Gagal menjalankan migrasi untuk kolom panel_type: %v", err)
	}

	alterTableAddPanelRemarksSQL := `
	DO $$
	BEGIN
		IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'panels' AND column_name = 'remarks') THEN
			ALTER TABLE panels ADD COLUMN remarks TEXT;
		END IF;
	END;
	$$;
	`
	if _, err := db.Exec(alterTableAddPanelRemarksSQL); err != nil {
		log.Fatalf("Gagal menjalankan migrasi untuk kolom panel.remarks: %v", err)
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM companies").Scan(&count); err != nil {
		log.Fatalf("Gagal cek data dummy: %v", err)
	}
	if count == 0 {
		log.Println("Database kosong, membuat data dummy...")
		insertDummyData(db)
	}

	alterTableAddBusbarCloseDatesSQL := `
	DO $$
	BEGIN
		IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'panels' AND column_name = 'close_date_busbar_pcc') THEN
			ALTER TABLE panels ADD COLUMN close_date_busbar_pcc TIMESTAMPTZ;
		END IF;
		IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'panels' AND column_name = 'close_date_busbar_mcc') THEN
			ALTER TABLE panels ADD COLUMN close_date_busbar_mcc TIMESTAMPTZ;
		END IF;
	END;
	$$;
	`
	if _, err := db.Exec(alterTableAddBusbarCloseDatesSQL); err != nil {
		log.Fatalf("Gagal menjalankan migrasi untuk kolom close_date_busbar: %v", err)
	}

	log.Println("Memastikan user sistem untuk Gemini AI ada...")
	createAiUserSQL := `
		INSERT INTO companies (id, name, role)
		VALUES ('gemini_ai_system', 'Gemini AI', 'viewer')
		ON CONFLICT (id) DO NOTHING;

		INSERT INTO company_accounts (username, password, company_id)
		VALUES ('gemini_ai', NULL, 'gemini_ai_system')
		ON CONFLICT (username) DO NOTHING;
	`
	if _, err := db.Exec(createAiUserSQL); err != nil {
		log.Fatalf("Gagal membuat user sistem untuk Gemini AI: %v", err)
	}

	var count_companies int
	if err := db.QueryRow("SELECT COUNT(*) FROM companies").Scan(&count); err != nil {
		log.Fatalf("Gagal cek data dummy: %v", err)
	}
	if count_companies == 0 {
		log.Println("Database kosong, membuat data dummy...")
		insertDummyData(db)
	}

}

func insertDummyData(db *sql.DB) {
	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	companies := []Company{
		{ID: "admin", Name: "Administrator", Role: AppRoleAdmin}, {ID: "viewer", Name: "Viewer", Role: AppRoleViewer},
		{ID: "warehouse", Name: "Warehouse", Role: AppRoleWarehouse}, {ID: "abacus", Name: "ABACUS", Role: AppRoleK3},
		{ID: "gaa", Name: "GAA", Role: AppRoleK3}, {ID: "gpe", Name: "GPE", Role: AppRoleK5},
		{ID: "dsm", Name: "DSM", Role: AppRoleK5}, {ID: "presisi", Name: "Presisi", Role: AppRoleK5},
	}
	stmt, err := tx.Prepare("INSERT INTO companies (id, name, role) VALUES ($1, $2, $3) ON CONFLICT (id) DO NOTHING")
	if err != nil {
		tx.Rollback()
		log.Fatal(err)
	}
	defer stmt.Close()
	for _, c := range companies {
		if _, err := stmt.Exec(c.ID, c.Name, c.Role); err != nil {
			tx.Rollback()
			log.Fatal(err)
		}
	}
	accounts := []CompanyAccount{
		{Username: "admin", Password: "123", CompanyID: "admin"}, {Username: "viewer", Password: "123", CompanyID: "viewer"},
		{Username: "whs_user1", Password: "123", CompanyID: "warehouse"}, {Username: "whs_user2", Password: "123", CompanyID: "warehouse"},
		{Username: "abacus_user1", Password: "123", CompanyID: "abacus"}, {Username: "abacus_user2", Password: "123", CompanyID: "abacus"},
		{Username: "gaa_user1", Password: "123", CompanyID: "gaa"}, {Username: "gaa_user2", Password: "123", CompanyID: "gaa"},
		{Username: "gpe_user1", Password: "123", CompanyID: "gpe"}, {Username: "gpe_user2", Password: "123", CompanyID: "gpe"},
		{Username: "dsm_user1", Password: "123", CompanyID: "dsm"}, {Username: "dsm_user2", Password: "123", CompanyID: "dsm"},
	}
	stmt2, err := tx.Prepare("INSERT INTO company_accounts (username, password, company_id) VALUES ($1, $2, $3) ON CONFLICT (username) DO NOTHING")
	if err != nil {
		tx.Rollback()
		log.Fatal(err)
	}
	defer stmt2.Close()
	for _, a := range accounts {
		if _, err := stmt2.Exec(a.Username, a.Password, a.CompanyID); err != nil {
			tx.Rollback()
			log.Fatal(err)
		}
	}
	if err := tx.Commit(); err != nil {
		log.Fatal(err)
	}
	log.Println("Data dummy berhasil dibuat.")
}
func fetchAllAs(db DBTX, tableName string, factory func() interface{}) (interface{}, error) {
	rows, err := db.Query(fmt.Sprintf("SELECT * FROM %s", tableName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []interface{}
	for rows.Next() {
		model := factory()
		switch m := model.(type) {
		case *Company:
			if err := rows.Scan(&m.ID, &m.Name, &m.Role); err != nil {
				return nil, err
			}
		case *CompanyAccount:
			if err := rows.Scan(&m.Username, &m.Password, &m.CompanyID); err != nil {
				return nil, err
			}
		case *Panel:
			if err := rows.Scan(&m.NoPp, &m.NoPanel, &m.NoWbs, &m.Project, &m.PercentProgress, &m.StartDate, &m.TargetDelivery, &m.StatusBusbarPcc, &m.StatusBusbarMcc, &m.StatusComponent, &m.StatusPalet, &m.StatusCorepart, &m.AoBusbarPcc, &m.AoBusbarMcc, &m.CreatedBy, &m.VendorID, &m.IsClosed, &m.ClosedDate); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("unsupported type for fetchAllAs: %T", m)
		}
		results = append(results, model)
	}
	return results, nil
}
func fetchInAs(db DBTX, tableName, columnName string, ids []string, factory func() interface{}) (interface{}, error) {
	if len(ids) == 0 {
		return []interface{}{}, nil
	}

	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf("SELECT * FROM %s WHERE %s IN (%s)", tableName, columnName, strings.Join(placeholders, ","))
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []interface{}
	for rows.Next() {
		model := factory()
		switch m := model.(type) {
		case *Panel:
			if err := rows.Scan(&m.NoPp, &m.NoPanel, &m.NoWbs, &m.Project, &m.PercentProgress, &m.StartDate, &m.TargetDelivery, &m.StatusBusbarPcc, &m.StatusBusbarMcc, &m.StatusComponent, &m.StatusPalet, &m.StatusCorepart, &m.AoBusbarPcc, &m.AoBusbarMcc, &m.CreatedBy, &m.VendorID, &m.IsClosed, &m.ClosedDate); err != nil {
				return nil, err
			}
		case *Busbar:
			if err := rows.Scan(&m.ID, &m.PanelNoPp, &m.Vendor, &m.Remarks); err != nil {
				return nil, err
			}
		case *Component:
			if err := rows.Scan(&m.ID, &m.PanelNoPp, &m.Vendor); err != nil {
				return nil, err
			}
		case *Palet:
			if err := rows.Scan(&m.ID, &m.PanelNoPp, &m.Vendor); err != nil {
				return nil, err
			}
		case *Corepart:
			if err := rows.Scan(&m.ID, &m.PanelNoPp, &m.Vendor); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("unsupported type for fetchInAs: %T", m)
		}
		results = append(results, model)
	}

	return results, nil
}
func insertMap(tx DBTX, tableName string, data map[string]interface{}) error {
	var cols []string
	var vals []interface{}
	var placeholders []string
	i := 1
	for col, val := range data {
		if val == nil {
			continue
		}
		cols = append(cols, col)
		vals = append(vals, val)
		placeholders = append(placeholders, fmt.Sprintf("$%d", i))
		i++
	}

	if len(cols) == 0 {
		return nil
	}

	var conflictColumn string
	var updateSetters []string

	switch tableName {
	case "companies":
		conflictColumn = "(id)"
	case "company_accounts":
		conflictColumn = "(username)"
	case "panels":
		conflictColumn = "(no_pp)"
	case "busbars", "components", "palet", "corepart":
		conflictColumn = "(panel_no_pp, vendor)"
	default:
		return fmt.Errorf("unknown table for insertMap: %s", tableName)
	}

	for _, col := range cols {
		if !strings.Contains(conflictColumn, col) {
			updateSetters = append(updateSetters, fmt.Sprintf("%s = EXCLUDED.%s", col, col))
		}
	}

	if len(updateSetters) == 0 && strings.Contains(conflictColumn, ",") {
		query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) ON CONFLICT %s DO NOTHING",
			tableName, strings.Join(cols, ", "), strings.Join(placeholders, ", "), conflictColumn)
		_, err := tx.Exec(query, vals...)
		return err
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) ON CONFLICT %s DO UPDATE SET %s",
		tableName, strings.Join(cols, ", "), strings.Join(placeholders, ", "), conflictColumn, strings.Join(updateSetters, ", "))

	_, err := tx.Exec(query, vals...)
	return err
}
func generateJsonTemplate(dataType string) map[string]interface{} {
	jsonData := make(map[string]interface{})
	if dataType == "companies_and_accounts" {
		jsonData["companies"] = []map[string]string{{"id": "vendor_k3_contoh", "name": "Nama Vendor K3 Contoh", "role": "k3"}}
		jsonData["company_accounts"] = []map[string]string{{"username": "staff_k3_contoh", "password": "123", "company_id": "vendor_k3_contoh"}}
	} else {
		now := time.Now().Format(time.RFC3339)
		jsonData["panels"] = []map[string]interface{}{{"no_pp": "PP-CONTOH-01", "no_panel": "PANEL-CONTOH-A", "percent_progress": 80.5, "start_date": now}}
		jsonData["busbars"] = []map[string]string{{"panel_no_pp": "PP-CONTOH-01", "vendor": "vendor_k5_contoh", "remarks": "Catatan busbar"}}
		jsonData["components"] = []map[string]string{{"panel_no_pp": "PP-CONTOH-01", "vendor": "warehouse"}}
		jsonData["palet"] = []map[string]string{{"panel_no_pp": "PP-CONTOH-01", "vendor": "vendor_k3_contoh"}}
		jsonData["corepart"] = []map[string]string{{"panel_no_pp": "PP-CONTOH-01", "vendor": "vendor_k3_contoh"}}
	}
	return jsonData
}
func generateExcelTemplate(dataType string) *excelize.File {
	f := excelize.NewFile()
	if dataType == "companies_and_accounts" {
		f.NewSheet("companies")
		f.SetCellValue("companies", "A1", "id")
		f.SetCellValue("companies", "B1", "name")
		f.SetCellValue("companies", "C1", "role")
		f.SetCellValue("companies", "A2", "vendor_k3_contoh")
		f.SetCellValue("companies", "B2", "Nama Vendor K3 Contoh")
		f.SetCellValue("companies", "C2", "k3")
		f.NewSheet("company_accounts")
		f.SetCellValue("company_accounts", "A1", "username")
		f.SetCellValue("company_accounts", "B1", "password")
		f.SetCellValue("company_accounts", "C1", "company_id")
		f.SetCellValue("company_accounts", "A2", "staff_k3_contoh")
		f.SetCellValue("company_accounts", "B2", "password123")
		f.SetCellValue("company_accounts", "C2", "vendor_k3_contoh")
	} else {
		panelHeaders := []string{"no_pp", "no_panel", "no_wbs", "project", "percent_progress", "start_date", "target_delivery", "status_busbar_pcc", "status_busbar_mcc", "status_component", "status_palet", "status_corepart", "ao_busbar_pcc", "ao_busbar_mcc", "created_by", "vendor_id", "is_closed", "closed_date"}
		f.NewSheet("panels")
		for i, h := range panelHeaders {
			f.SetCellValue("panels", fmt.Sprintf("%c1", 'A'+i), h)
		}

		busbarHeaders := []string{"panel_no_pp", "vendor", "remarks"}
		f.NewSheet("busbars")
		for i, h := range busbarHeaders {
			f.SetCellValue("busbars", fmt.Sprintf("%c1", 'A'+i), h)
		}

		f.NewSheet("components")
		f.SetCellValue("components", "A1", "panel_no_pp")
		f.SetCellValue("components", "B1", "vendor")
		f.NewSheet("palet")
		f.SetCellValue("palet", "A1", "panel_no_pp")
		f.SetCellValue("palet", "B1", "vendor")
		f.NewSheet("corepart")
		f.SetCellValue("corepart", "A1", "panel_no_pp")
		f.SetCellValue("corepart", "B1", "vendor")
	}
	f.DeleteSheet("Sheet1")
	return f
}
func getValueCaseInsensitive(row map[string]interface{}, key string) string {
	lowerKey := strings.ToLower(strings.ReplaceAll(key, " ", ""))
	for k, v := range row {
		if strings.ToLower(strings.ReplaceAll(k, " ", "")) == lowerKey {
			if v != nil {
				return fmt.Sprintf("%v", v)
			}
		}
	}
	return ""
}

func parseDate(dateStr string) *string {
	if dateStr == "" {
		return nil
	}
	// Daftar format yang akan dicoba untuk di-parsing.
	layouts := []string{
		time.RFC3339Nano,              // [PENAMBAHAN] Format dengan milidetik: 2025-07-17T00:00:00.000Z
		time.RFC3339,                   // Format standar: 2025-07-17T00:00:00Z
		"2006-01-02T15:04:05Z07:00",     // Format standar lain
		"02-Jan-2006",                  // Format: 17-Jul-2025
		"02-Jan-06",                    // Format: 17-Jul-25
		"1/2/2006",                     // Format: 7/17/2025
	}
	for _, layout := range layouts {
		t, err := time.Parse(layout, dateStr)
		if err == nil {
			// Jika berhasil, ubah ke format standar ISO 8601 untuk disimpan di DB
			s := t.Format(time.RFC3339)
			return &s
		}
	}
	// Jika semua format gagal, kembalikan nil
	return nil
}

// getMessagesByChatIDHandler mengambil semua pesan untuk sebuah chat ID.
func (a *App) getMessagesByChatIDHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chatID, err := strconv.Atoi(vars["chat_id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid Chat ID")
		return
	}

	rows, err := a.DB.Query(`
		SELECT id, chat_id, sender_username, text, image_data, replied_issue_id, created_at 
		FROM chat_messages 
		WHERE chat_id = $1 
		ORDER BY created_at ASC`, chatID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to retrieve messages: "+err.Error())
		return
	}
	defer rows.Close()

	var messages []ChatMessage
	for rows.Next() {
		var msg ChatMessage
		if err := rows.Scan(&msg.ID, &msg.ChatID, &msg.SenderUsername, &msg.Text, &msg.ImageData, &msg.RepliedIssueID, &msg.CreatedAt); err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to scan message: "+err.Error())
			return
		}
		messages = append(messages, msg)
	}

	respondWithJSON(w, http.StatusOK, messages)
}

// createMessageHandler mengirim sebuah pesan baru ke dalam chat.
func (a *App) createMessageHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chatID, err := strconv.Atoi(vars["chat_id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid Chat ID")
		return
	}

	var payload struct {
		SenderUsername string  `json:"sender_username"`
		Text           *string `json:"text"`
		ImageData      *string `json:"image_data"` // Base64 string
		RepliedIssueID *int    `json:"replied_issue_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid payload: "+err.Error())
		return
	}

	if (payload.Text == nil || *payload.Text == "") && (payload.ImageData == nil || *payload.ImageData == "") {
		respondWithError(w, http.StatusBadRequest, "Message cannot be empty (must have text or image)")
		return
	}

	var createdMessage ChatMessage
	query := `
		INSERT INTO chat_messages (chat_id, sender_username, text, image_data, replied_issue_id) 
		VALUES ($1, $2, $3, $4, $5) 
		RETURNING id, chat_id, sender_username, text, image_data, replied_issue_id, created_at`

	err = a.DB.QueryRow(query, chatID, payload.SenderUsername, payload.Text, payload.ImageData, payload.RepliedIssueID).Scan(
		&createdMessage.ID, &createdMessage.ChatID, &createdMessage.SenderUsername, &createdMessage.Text, &createdMessage.ImageData, &createdMessage.RepliedIssueID, &createdMessage.CreatedAt,
	)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create message: "+err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, createdMessage)
}

func (a *App) getCommentsByIssueHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	issueID, err := strconv.Atoi(vars["issue_id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid Issue ID")
		return
	}

	query := `
		SELECT
			ic.id,
			ic.issue_id,
			ic.text,
			ic.timestamp,
			ic.reply_to_comment_id,
			ic.is_edited,
			COALESCE(ic.image_urls, '[]'::jsonb),
			sender.username as sender_id,
			sender.username as sender_name, -- Bisa diganti dengan nama asli jika ada
			reply_user.username as reply_to_user_id,
			reply_user.username as reply_to_user_name
		FROM issue_comments ic
		JOIN company_accounts sender ON ic.sender_id = sender.username
		LEFT JOIN company_accounts reply_user ON ic.reply_to_user_id = reply_user.username
		WHERE ic.issue_id = $1
		ORDER BY ic.timestamp ASC
	`

	rows, err := a.DB.Query(query, issueID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to query comments: "+err.Error())
		return
	}
	defer rows.Close()

	var comments []IssueComment
	for rows.Next() {
		var c IssueComment
		var senderID, senderName, replyToUserID, replyToUserName sql.NullString
		var imageUrlsJSON []byte

		err := rows.Scan(
			&c.ID, &c.IssueID, &c.Text, &c.Timestamp, &c.ReplyToCommentID, &c.IsEdited, &imageUrlsJSON,
			&senderID, &senderName, &replyToUserID, &replyToUserName,
		)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to scan comment: "+err.Error())
			return
		}

		c.Sender = User{ID: senderID.String, Name: senderName.String}
		if replyToUserID.Valid {
			c.ReplyTo = &User{ID: replyToUserID.String, Name: replyToUserName.String}
		}

		if err := json.Unmarshal(imageUrlsJSON, &c.ImageUrls); err != nil {
			c.ImageUrls = []string{}
		}

		comments = append(comments, c)
	}

	respondWithJSON(w, http.StatusOK, comments)
}

func (a *App) createCommentHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	issueID, err := strconv.Atoi(vars["issue_id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid Issue ID")
		return
	}

	var payload CreateCommentPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid payload: "+err.Error())
		return
	}

	var imageUrls []string
	for _, base64Image := range payload.Images {
		// Decode base64
		parts := strings.Split(base64Image, ",")
		if len(parts) != 2 {
			log.Println("Invalid base64 image format")
			continue
		}
		imgData, err := base64.StdEncoding.DecodeString(parts[1])
		if err != nil {
			log.Printf("Failed to decode base64: %v", err)
			continue
		}

		// Simpan ke file
		filename := fmt.Sprintf("%s.jpg", uuid.New().String())
		filepath := fmt.Sprintf("uploads/%s", filename)
		err = os.WriteFile(filepath, imgData, 0644)
		if err != nil {
			log.Printf("Failed to save image file: %v", err)
			continue
		}
		// Simpan URL-nya
		imageUrls = append(imageUrls, "/"+filepath)
	}

	imageUrlsJSON, _ := json.Marshal(imageUrls)
	newCommentID := uuid.New().String()

	query := `
		INSERT INTO issue_comments (id, issue_id, sender_id, text, reply_to_comment_id, reply_to_user_id, image_urls)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err = a.DB.Exec(query, newCommentID, issueID, payload.SenderID, payload.Text, payload.ReplyToCommentID, payload.ReplyToUserID, imageUrlsJSON)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create comment: "+err.Error())
		return
	}

	respondWithJSON(w, http.StatusCreated, map[string]string{"id": newCommentID})
}
// Ganti fungsi lama dengan versi final ini di main.go
func (a *App) updateCommentHandler(w http.ResponseWriter, r *http.Request) {
	commentID := mux.Vars(r)["id"]

	var payload struct {
		Text   string   `json:"text"`
		Images []string `json:"images"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid payload")
		return
	}

	tx, err := a.DB.Begin()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Gagal memulai transaksi")
		return
	}
	defer tx.Rollback()

	var issueID int
	var isSystemComment sql.NullBool
	err = tx.QueryRow("SELECT issue_id, is_system_comment FROM issue_comments WHERE id = $1", commentID).Scan(&issueID, &isSystemComment)
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusNotFound, "Komentar tidak ditemukan")
		} else {
			respondWithError(w, http.StatusInternalServerError, "Gagal mengambil detail komentar: "+err.Error())
		}
		return
	}

	// Update komentar itu sendiri seperti biasa
	var finalImageUrls []string
	for _, img := range payload.Images {
		if strings.HasPrefix(img, "data:image") {
			parts := strings.Split(img, ",")
			if len(parts) < 2 { continue }
			imgData, _ := base64.StdEncoding.DecodeString(parts[1])
			filename := fmt.Sprintf("%s.jpg", uuid.New().String())
			filepath := fmt.Sprintf("uploads/%s", filename)
			os.WriteFile(filepath, imgData, 0644)
			finalImageUrls = append(finalImageUrls, "/"+filepath)
		} else {
			finalImageUrls = append(finalImageUrls, img)
		}
	}
	imageUrlsJSON, _ := json.Marshal(finalImageUrls)
	
	updateCommentQuery := `UPDATE issue_comments SET text = $1, is_edited = true, image_urls = $2, is_system_comment = false WHERE id = $3`
	_, err = tx.Exec(updateCommentQuery, payload.Text, imageUrlsJSON, commentID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Gagal update komentar")
		return
	}

	// Jika komentar yang diedit ADALAH komentar sistem, update juga isunya
	if isSystemComment.Valid && isSystemComment.Bool {
		
		// --- ▼▼▼ LOGIKA PARSING BARU YANG LEBIH FLEKSIBEL ▼▼▼ ---
		var newTitle, newDescription string
		
		// Gunakan SplitN untuk membagi string hanya pada kemunculan ": " pertama
		parts := strings.SplitN(payload.Text, ": ", 2)

		if len(parts) == 2 {
			// Jika ada pemisah ": ", bagian pertama adalah title, kedua adalah description
			newTitle = strings.Trim(parts[0], "* ") // Hapus markdown ** dan spasi
			newDescription = strings.TrimSpace(parts[1])
		} else {
			// Jika tidak ada pemisah, seluruh teks adalah title
			newTitle = strings.Trim(payload.Text, "* ")
			newDescription = "" // Kosongkan deskripsi
		}
		// --- ▲▲▲ AKHIR LOGIKA PARSING BARU ▲▲▲ ---

		// Lanjutkan hanya jika title tidak kosong setelah parsing
		if newTitle != "" {
			updateIssueQuery := `UPDATE issues SET title = $1, description = $2 WHERE id = $3`
			_, err = tx.Exec(updateIssueQuery, newTitle, newDescription, issueID)
			if err != nil {
				respondWithError(w, http.StatusInternalServerError, "Gagal sinkronisasi update ke isu: "+err.Error())
				return
			}
		}
	}

	if err := tx.Commit(); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Gagal commit transaksi")
		return
	}
	
	respondWithJSON(w, http.StatusOK, map[string]string{"status": "success"})
}
func (a *App) deleteCommentHandler(w http.ResponseWriter, r *http.Request) {
	commentID := mux.Vars(r)["id"]
	_, err := a.DB.Exec("DELETE FROM issue_comments WHERE id = $1", commentID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to delete comment")
		return
	}
	respondWithJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
func (a *App) getAllIssueTitlesHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := a.DB.Query("SELECT id, title FROM issue_titles ORDER BY title ASC")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	var titles []IssueTitle
	for rows.Next() {
		var it IssueTitle
		if err := rows.Scan(&it.ID, &it.Title); err != nil {
			respondWithError(w, http.StatusInternalServerError, "Gagal scan title: "+err.Error())
			return
		}
		titles = append(titles, it)
	}
	respondWithJSON(w, http.StatusOK, titles)
}

func (a *App) createIssueTitleHandler(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid payload")
		return
	}
	if payload.Title == "" {
		respondWithError(w, http.StatusBadRequest, "Title tidak boleh kosong")
		return
	}

	var newTitle IssueTitle
	query := "INSERT INTO issue_titles (title) VALUES ($1) RETURNING id, title"
	err := a.DB.QueryRow(query, payload.Title).Scan(&newTitle.ID, &newTitle.Title)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			respondWithError(w, http.StatusConflict, "Tipe masalah dengan nama tersebut sudah ada.")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Gagal membuat tipe masalah: "+err.Error())
		return
	}
	respondWithJSON(w, http.StatusCreated, newTitle)
}

func (a *App) updateIssueTitleHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID")
		return
	}
	var payload struct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid payload")
		return
	}
	if payload.Title == "" {
		respondWithError(w, http.StatusBadRequest, "Title tidak boleh kosong")
		return
	}

	tx, err := a.DB.Begin()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Gagal memulai transaksi")
		return
	}
	defer tx.Rollback()

	var oldTitle string
	err = tx.QueryRow("SELECT title FROM issue_titles WHERE id = $1", id).Scan(&oldTitle)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Tipe masalah tidak ditemukan")
		return
	}

	_, err = tx.Exec("UPDATE issue_titles SET title = $1 WHERE id = $2", payload.Title, id)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			respondWithError(w, http.StatusConflict, "Tipe masalah dengan nama tersebut sudah ada.")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Gagal update master title: "+err.Error())
		return
	}

	_, err = tx.Exec("UPDATE issues SET title = $1 WHERE title = $2", payload.Title, oldTitle)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Gagal update issue yang ada: "+err.Error())
		return
	}

	if err := tx.Commit(); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Gagal commit transaksi")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

func (a *App) deleteIssueTitleHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID")
		return
	}

	var oldTitle string
	err = a.DB.QueryRow("SELECT title FROM issue_titles WHERE id = $1", id).Scan(&oldTitle)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Tipe masalah tidak ditemukan")
		return
	}

	var count int
	err = a.DB.QueryRow("SELECT COUNT(*) FROM issues WHERE title = $1", oldTitle).Scan(&count)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Gagal memeriksa penggunaan title: "+err.Error())
		return
	}

	if count > 0 {
		respondWithError(w, http.StatusConflict, fmt.Sprintf("Tidak dapat menghapus. Tipe masalah ini digunakan oleh %d isu.", count))
		return
	}

	_, err = a.DB.Exec("DELETE FROM issue_titles WHERE id = $1", id)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Gagal menghapus: "+err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
func (a *App) askGeminiHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	issueID, err := strconv.Atoi(vars["issue_id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid Issue ID")
		return
	}

	var payload struct {
		Question       string `json:"question"`
		SenderID       string `json:"sender_id"`
		ReplyToCommentID string `json:"reply_to_comment_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid payload: "+err.Error())
		return
	}

	// 1. Ambil Konteks Isu
	var issueTitle, issueDesc string
	err = a.DB.QueryRow("SELECT title, description FROM issues WHERE id = $1", issueID).Scan(&issueTitle, &issueDesc)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Issue not found")
		return
	}

	// 2. Ambil Konteks Komentar
	rows, err := a.DB.Query(`
        SELECT ca.username, ic.text 
        FROM issue_comments ic
        JOIN company_accounts ca ON ic.sender_id = ca.username
        WHERE ic.issue_id = $1 ORDER BY ic.timestamp ASC
    `, issueID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Gagal mengambil histori komentar")
		return
	}
	defer rows.Close()

	var commentHistory strings.Builder
	for rows.Next() {
		var username, text string
		if err := rows.Scan(&username, &text); err != nil {
			continue
		}
		commentHistory.WriteString(fmt.Sprintf("%s: %s\n", username, text))
	}

	// 3. Setup Gemini Client
	ctx := context.Background()
	
	apiKey := "AIzaSyDiMY2xY0N_eOw5vUzk-J3sLVDb81TEfS8"
	if apiKey == ""  {
		respondWithError(w, http.StatusInternalServerError, "GEMINI_API_KEY is not set on the server")
		return
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create Gemini client: "+err.Error())
		return
	}
	defer client.Close()

	// 4. Setup Model dan Prompt
	model := client.GenerativeModel("gemini-2.5-flash-lite")
	model.Tools = tools // Menggunakan tools yang sudah didefinisikan
	cs := model.StartChat()

	fullPrompt := fmt.Sprintf(
		"Anda adalah asisten AI. Berdasarkan konteks isu dan histori komentar berikut, jawab pertanyaan user.\n\n"+
			"--- Konteks Isu ---\n"+
			"Judul: %s\n"+
			"Deskripsi: %s\n\n"+
			"--- Histori Komentar Sejauh Ini ---\n"+
			"%s\n"+
			"-----------------------------------\n\n"+
			"Pertanyaan dari user '%s': %s",
		issueTitle, issueDesc, commentHistory.String(), payload.SenderID, payload.Question,
	)

	// 5. Kirim Prompt dan Eksekusi Function Call (BAGIAN PERBAIKAN)
	log.Printf("Mengirim prompt ke Gemini: %s", fullPrompt)
	resp, err := cs.SendMessage(ctx, genai.Text(fullPrompt))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to generate content: "+err.Error())
		return
	}

	if resp.Candidates != nil && len(resp.Candidates) > 0 && resp.Candidates[0].Content != nil {
		part := resp.Candidates[0].Content.Parts[0]
		if fc, ok := part.(genai.FunctionCall); ok {
			log.Printf("Gemini meminta pemanggilan fungsi: %s dengan argumen: %v", fc.Name, fc.Args)

			// ▼▼▼ PERBAIKAN UTAMA ADA DI SINI ▼▼▼
			// Dapatkan 'panelNoPp' dari 'issueID' sebelum memanggil fungsi eksekusi.
			var panelNoPp string
			err := a.DB.QueryRow(`
                SELECT p.no_pp FROM panels p 
                JOIN chats c ON p.no_pp = c.panel_no_pp 
                JOIN issues i ON c.id = i.chat_id 
                WHERE i.id = $1`, issueID).Scan(&panelNoPp)
			if err != nil {
				// Jika panel tidak ditemukan, kirim pesan error yang jelas.
				respondWithError(w, http.StatusInternalServerError, "Gagal menemukan panel terkait isu ini: "+err.Error())
				return
			}
			// ▲▲▲ AKHIR PERBAIKAN UTAMA ▲▲▲

			// Panggil fungsi eksekusi dengan argumen yang benar (string, bukan int)
			functionResult, err := a.executeDatabaseFunction(fc, panelNoPp)
			if err != nil {
				functionResult = fmt.Sprintf("Error saat menjalankan fungsi: %v", err)
			}

			log.Printf("Hasil eksekusi fungsi: %s", functionResult)

			resp, err = cs.SendMessage(ctx, genai.FunctionResponse{Name: fc.Name, Response: map[string]any{"result": functionResult}})
			if err != nil {
				respondWithError(w, http.StatusInternalServerError, "Failed to generate confirmation message: "+err.Error())
				return
			}
		}
	}

	// 6. Proses dan Kirim Jawaban Final
	finalResponseText := extractTextFromResponse(resp)
	if finalResponseText == "" {
		finalResponseText = "Maaf, terjadi kesalahan saat memproses permintaan Anda."
	}
	log.Printf("Jawaban final dari Gemini: %s", finalResponseText)

	a.postAiComment(issueID, payload.SenderID, finalResponseText, payload.ReplyToCommentID)
	respondWithJSON(w, http.StatusCreated, map[string]string{"status": "success"})
}
func (a *App) postAiComment(issueID int, senderID string, text string, replyToCommentID string) { // <-- UBAH DI SINI
	newCommentID := uuid.New().String()
	geminiUserID := "gemini_ai"

	// Query INSERT sekarang harus menyertakan reply_to_comment_id
	query := `
		INSERT INTO issue_comments (id, issue_id, sender_id, text, reply_to_user_id, reply_to_comment_id) 
		VALUES ($1, $2, $3, $4, $5, $6)` 
		
	_, _ = a.DB.Exec(query, newCommentID, issueID, geminiUserID, text, senderID, replyToCommentID) // <-- UBAH DI SINI
}
// FUNGSI HELPER BARU untuk mengekstrak teks dari response Gemini
func extractTextFromResponse(resp *genai.GenerateContentResponse) string {
	if resp != nil && len(resp.Candidates) > 0 {
		if content := resp.Candidates[0].Content; content != nil && len(content.Parts) > 0 {
			if txt, ok := content.Parts[0].(genai.Text); ok {
				return string(txt)
			}
		}
	}
	return ""
}
var tools = []*genai.Tool{
	{
		FunctionDeclarations: []*genai.FunctionDeclaration{
			{
				Name:        "get_issue_explanation",
				Description: "Memberikan penjelasan dan ringkasan tentang isu yang sedang dibahas berdasarkan judul dan deskripsinya.",
			},
			{
				Name:        "find_related_issues",
				Description: "Mencari dan memberikan daftar isu-isu lain yang relevan di dalam panel yang sama.",
			},
			{
				Name:        "update_issue_status",
				Description: "Mengubah status dari sebuah isu. Status 'done' atau 'selesai' akan dianggap sebagai 'solved'.",
				Parameters: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"new_status": {
							Type:        genai.TypeString,
							Description: "Status baru untuk isu ini. Pilihan: 'solved' atau 'unsolved'.",
							Enum:        []string{"solved", "unsolved"},
						},
					},
					Required: []string{"new_status"},
				},
			},
			{
				Name:        "assign_vendor_to_panel",
				Description: "Menugaskan (assign) sebuah vendor/tim ke sebuah kategori pekerjaan di panel ini.",
				Parameters: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"vendor_name": {
							Type:        genai.TypeString,
							Description: "Nama vendor atau tim yang akan ditugaskan, contoh: 'GPE', 'DSM', 'Warehouse'.",
						},
						"category": {
							Type:        genai.TypeString,
							Description: "Kategori pekerjaan yang akan ditugaskan. Pilihan: 'busbar', 'component', 'palet', 'corepart'.",
							Enum:        []string{"busbar", "component", "palet", "corepart"},
						},
					},
					Required: []string{"vendor_name", "category"},
				},
			},{
				Name:        "update_busbar_status",
				Description: "Mengubah status untuk komponen Busbar PCC atau Busbar MCC.",
				Parameters: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"busbar_type": {
							Type:        genai.TypeString,
							Description: "Tipe busbar yang akan diubah.",
							Enum:        []string{"pcc", "mcc"},
						},
						"new_status": {
							Type:        genai.TypeString,
							Description: "Status baru untuk busbar.",
							Enum:        []string{"Open", "Punching/Bending","Plating/Epoxy","100% Siap Kirim", "Close"},
						},
					},
					Required: []string{"busbar_type", "new_status"},
				},
			},

			// 2. Fungsi khusus untuk Component
			{
				Name:        "update_component_status",
				Description: "Mengubah status untuk komponen utama (picking component).",
				Parameters: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"new_status": {
							Type:        genai.TypeString,
							Description: "Status baru untuk komponen.",
							Enum:        []string{"Open", "On Progress", "Done"},
						},
					},
					Required: []string{"new_status"},
				},
			},

			// 3. Fungsi khusus untuk Palet
			{
				Name:        "update_palet_status",
				Description: "Mengubah status untuk komponen Palet.",
				Parameters: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"new_status": {
							Type:        genai.TypeString,
							Description: "Status baru untuk palet.",
							Enum:        []string{"Open", "Close"},
						},
					},
					Required: []string{"new_status"},
				},
			},

			// 4. Fungsi khusus untuk Corepart
			{
				Name:        "update_corepart_status",
				Description: "Mengubah status untuk komponen Corepart.",
				Parameters: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"new_status": {
							Type:        genai.TypeString,
							Description: "Status baru untuk corepart.",
							Enum:        []string{"Open", "Close"},
						},
					},
					Required: []string{"new_status"},
				},
			},
		},
	},
}
type pdd struct {
	NoPp               sql.NullString  `json:"no_pp"`
	NoPanel            sql.NullString  `json:"no_panel"`
	Project            sql.NullString  `json:"project"`
	NoWbs              sql.NullString  `json:"no_wbs"`
	PercentProgress    sql.NullFloat64 `json:"percent_progress"`
	StatusBusbarPcc    sql.NullString  `json:"status_busbar_pcc"`
	StatusBusbarMcc    sql.NullString  `json:"status_busbar_mcc"`
	StatusComponent    sql.NullString  `json:"status_component"`
	StatusPalet        sql.NullString  `json:"status_palet"`
	StatusCorepart     sql.NullString  `json:"status_corepart"`
	PanelVendorName    sql.NullString  `json:"panel_vendor_name"`
	BusbarVendorNames  sql.NullString  `json:"busbar_vendor_names"`
}

func (a *App) askGeminiAboutPanelHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	panelNoPp := vars["no_pp"]

	var payload struct {
		Question string  `json:"question"`
		SenderID string  `json:"sender_id"`
		ImageB64 *string `json:"image_b64,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid payload")
		return
	}

	// 1. Dapatkan role user
	var senderRole string
	err := a.DB.QueryRow(`
		SELECT c.role FROM companies c
		JOIN company_accounts ca ON c.id = ca.company_id
		WHERE ca.username = $1`, payload.SenderID).Scan(&senderRole)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Gagal mendapatkan role user: "+err.Error())
		return
	}

	// 2. Kumpulkan konteks data panel
	var panel pdd
	err = a.DB.QueryRow(`
        SELECT p.no_pp, p.no_panel, p.project, p.no_wbs, p.percent_progress, p.status_busbar_pcc, p.status_busbar_mcc, p.status_component, p.status_palet, p.status_corepart, pu.name as panel_vendor_name, (SELECT STRING_AGG(c.name, ', ') FROM companies c JOIN busbars b ON c.id = b.vendor WHERE b.panel_no_pp = p.no_pp) as busbar_vendor_names
        FROM panels p
        LEFT JOIN companies pu ON p.vendor_id = pu.id
        WHERE p.no_pp = $1`, panelNoPp).Scan(&panel.NoPp, &panel.NoPanel, &panel.Project, &panel.NoWbs, &panel.PercentProgress, &panel.StatusBusbarPcc, &panel.StatusBusbarMcc, &panel.StatusComponent, &panel.StatusPalet, &panel.StatusCorepart, &panel.PanelVendorName, &panel.BusbarVendorNames)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Panel tidak ditemukan: "+err.Error())
		return
	}
	panelDetailsBytes, _ := json.MarshalIndent(panel, "", "  ")
    panelDetails := string(panelDetailsBytes)


	// 3. Kumpulkan konteks histori isu & komentar
	rows, err := a.DB.Query(`
		SELECT i.title, i.description, i.status, ic.sender_id, ic.text
		FROM issues i
		JOIN chats ch ON i.chat_id = ch.id
		LEFT JOIN issue_comments ic ON i.id = ic.issue_id
		WHERE ch.panel_no_pp = $1
		ORDER BY i.created_at, ic.timestamp`, panelNoPp)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Gagal mengambil histori isu")
		return
	}
	defer rows.Close()

	var issuesHistory strings.Builder
	currentIssueTitle := ""
	for rows.Next() {
		var issueTitle, issueDesc, issueStatus, commentSender, commentText sql.NullString
		if err := rows.Scan(&issueTitle, &issueDesc, &issueStatus, &commentSender, &commentText); err != nil { continue }
		if issueTitle.String != currentIssueTitle {
			issuesHistory.WriteString(fmt.Sprintf("\n--- ISU BARU ---\nJudul: %s\nDeskripsi: %s\nStatus: %s\n", issueTitle.String, issueDesc.String, issueStatus.String))
			currentIssueTitle = issueTitle.String
		}
		if commentSender.Valid {
			issuesHistory.WriteString(fmt.Sprintf(" - Komentar dari %s: %s\n", commentSender.String, commentText.String))
		}
	}

	// 4. Setup Gemini Client & Prompt yang disempurnakan
	ctx := context.Background()
	apiKey := "AIzaSyDiMY2xY0N_eOw5vUzk-J3sLVDb81TEfS8"
	if apiKey == "" {
		respondWithError(w, http.StatusInternalServerError, "GEMINI_API_KEY is not set on the server")
		return
	}
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil { respondWithError(w, http.StatusInternalServerError, "Failed to create Gemini client: "+err.Error()); return }
	defer client.Close()

	model := client.GenerativeModel("gemini-1.5-flash")
	model.Tools = getToolsForRole(senderRole)
	cs := model.StartChat()
	
	var promptParts []genai.Part
	fullPromptText := fmt.Sprintf(
		"**Persona & Aturan:**\n"+
			"1.  **Kamu adalah asisten AI yang ramah, proaktif, dan suportif bernama Gemini.** Gunakan bahasa Indonesia yang santai dan bersahabat, bukan bahasa yang kaku atau robotik. Sapa user jika perlu.\n"+
			"2.  **Gunakan format tebal (bold) dengan markdown `**teks**`** saat menyebutkan nama isu atau hal penting lainnya agar mudah dibaca.\n"+
			"3.  Selalu berikan jawaban yang jelas dan langsung ke intinya. Jika kamu melakukan sebuah aksi (seperti mengubah status), berikan konfirmasi yang jelas dan positif, contoh: 'Siap! Status untuk isu **Add SR** sudah aku ubah jadi solved ya.'\n"+
			"4.  Kamu memiliki akses luas untuk melihat dan mengubah semua data di database, KECUALI data akun pengguna.\n"+
			"5.  Lakukan aksi HANYA jika diizinkan oleh role user yang bertanya. Role user saat ini adalah: **%s**.\n"+
			"6. **PENTING: Aturan Membuat Sugesti Aksi**\n" +
			"   - Jika ada isu yang belum selesai ('unsolved'), kamu HARUS memberikan rekomendasi dalam format `[SUGGESTION: Teks Aksi]`.\n" +
			"   - **Teks Aksi HARUS singkat, jelas, dan berupa perintah langsung** yang bisa dieksekusi.\n" +
			"   - **JANGAN PERNAH** menyertakan penjelasan, kondisi, atau kalimat percakapan di dalam kurung siku `[]`.\n" +
			"   - **Contoh BAIK:** `[SUGGESTION: Ubah status 'Missing Metal Part' menjadi solved]`\n" +
			"   - **Contoh BAIK:** `[SUGGESTION: Apa saja detail isu 'NC Metal/Busbar'?]`\n" +
			"   - **Contoh BURUK:** `[SUGGESTION: Mungkin kamu bisa mengubah status 'Missing Metal Part' menjadi solved kalau sudah selesai]`\n" +
			"   - **Contoh BURUK:** `[SUGGESTION: Ubah statusnya menjadi solved setelah masalahnya teratasi]`\n\n" +
			"**Konteks Data Proyek Saat Ini:**\n"+
			"Berikut adalah data detail dari panel yang sedang kita diskusikan:\n"+
			"```json\n%s\n```\n\n"+
			"Dan ini adalah riwayat semua isu dan komentar yang pernah ada di panel ini:\n"+
			"```\n%s\n```\n\n"+
			"**Permintaan User:**\n"+
			"User '%s' bertanya: \"%s\"",
		senderRole, panelDetails, issuesHistory.String(), payload.SenderID, payload.Question,
	)
	promptParts = append(promptParts, genai.Text(fullPromptText))
	
	if payload.ImageB64 != nil && *payload.ImageB64 != "" {
		parts := strings.Split(*payload.ImageB64, ",")
		if len(parts) == 2 {
			imageBytes, err := base64.StdEncoding.DecodeString(parts[1])
			if err == nil {
				promptParts = append(promptParts, genai.ImageData("jpeg", imageBytes))
				log.Println("Gambar berhasil ditambahkan ke prompt Gemini.")
			}
		}
	}

	// 5. Kirim ke Gemini dan LAKUKAN FUNCTION CALL JIKA PERLU
	log.Printf("Mengirim prompt ke Gemini untuk user role: %s", senderRole)
	resp, err := cs.SendMessage(ctx, promptParts...)
	if err != nil { respondWithError(w, http.StatusInternalServerError, "Failed to generate content: "+err.Error()); return }

	actionTaken := false
	if resp.Candidates != nil && len(resp.Candidates) > 0 && resp.Candidates[0].Content != nil {
		part := resp.Candidates[0].Content.Parts[0]
		// ▼▼▼ BAGIAN KUNCI: Cek apakah AI ingin memanggil fungsi ▼▼▼
		if fc, ok := part.(genai.FunctionCall); ok {
			actionTaken = true // Tandai bahwa aksi akan dilakukan
			log.Printf("Gemini meminta pemanggilan fungsi: %s dengan argumen: %v", fc.Name, fc.Args)

			// Jalankan fungsi di database
			functionResult, err := a.executeDatabaseFunction(fc, panelNoPp)
			if err != nil { 
				functionResult = fmt.Sprintf("Error saat menjalankan fungsi: %v", err) 
			}
			log.Printf("Hasil eksekusi fungsi: %s", functionResult)
			
			// Kirim hasil eksekusi kembali ke Gemini agar ia bisa merangkai kalimat konfirmasi
			resp, err = cs.SendMessage(ctx, genai.FunctionResponse{Name: fc.Name, Response: map[string]any{"result": functionResult}})
			if err != nil { respondWithError(w, http.StatusInternalServerError, "Failed to generate confirmation message: "+err.Error()); return }
		}
		// ▲▲▲ AKHIR BAGIAN KUNCI ▲▲▲
	}
	
	finalResponseText := extractTextFromResponse(resp)
	if finalResponseText == "" { finalResponseText = "Waduh, maaf, sepertinya ada sedikit kendala. Boleh coba tanya lagi?" }

	var suggestions []string
	re := regexp.MustCompile(`\[SUGGESTION:\s*(.*?)\]`)
	matches := re.FindAllStringSubmatch(finalResponseText, -1)
	for _, match := range matches {
		if len(match) > 1 {
			suggestions = append(suggestions, match[1])
		}
	}
	finalResponseText = re.ReplaceAllString(finalResponseText, "")
	
	// 6. Kirim Response ke Frontend
	respondWithJSON(w, http.StatusCreated, map[string]interface{}{
		"id":   uuid.New().String(),
		"text": strings.TrimSpace(finalResponseText),
		"action_taken": actionTaken, // Kirim status apakah aksi telah dilakukan
		"suggested_actions": suggestions,
	})
}

// Fungsi ini mendefinisikan SEMUA kemampuan yang bisa dimiliki AI
func getToolsForRole(role string) []*genai.Tool {
	// 1. Definisikan semua kemungkinan "alat" (kemampuan)
	viewTools := []*genai.FunctionDeclaration{
		{
			Name: "get_panel_summary",
			Description: "Memberikan ringkasan status dan progres terkini dari panel yang sedang dibahas.",
		},
	}

	issueEditTools := []*genai.FunctionDeclaration{
		{
			Name: "update_issue_status",
			Description: "Mengubah status dari sebuah isu spesifik di dalam panel. Gunakan judul isu untuk mengidentifikasinya.",
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"issue_title": {Type: genai.TypeString, Description: "Judul isu yang statusnya ingin diubah. Harus sama persis."},
					"new_status":  {Type: genai.TypeString, Enum: []string{"solved", "unsolved"}},
				},
				Required: []string{"issue_title", "new_status"},
			},
		},
	}

	adminTools := []*genai.FunctionDeclaration{
		{
			Name: "update_panel_progress",
			Description: "ADMIN ONLY: Mengubah persentase progres dari sebuah panel.",
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{"new_progress": {Type: genai.TypeNumber, Description: "Nilai progres baru antara 0-100."}},
				Required: []string{"new_progress"},
			},
		},
		{
			Name: "update_panel_closed_status",
			Description: "ADMIN ONLY: Menandai panel sebagai 'sudah dikirim' (closed) atau 'belum dikirim' (open).",
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"is_closed": {Type: genai.TypeBoolean, Description: "Status baru, true untuk closed, false untuk open."},
					"closed_date": {Type: genai.TypeString, Description: "Tanggal penutupan dalam format YYYY-MM-DD. Hanya relevan jika is_closed true."},
				},
				Required: []string{"is_closed"},
			},
		},
		{
			Name: "assign_k3_vendor_to_panel",
			Description: "ADMIN ONLY: Menugaskan vendor K3 (Panel/Palet/Corepart) ke panel ini.",
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{"vendor_name": {Type: genai.TypeString, Description: "Nama vendor K3 yang akan ditugaskan."}},
				Required: []string{"vendor_name"},
			},
		},
	}

	k3Tools := []*genai.FunctionDeclaration{
		{
			Name: "update_palet_status",
			Description: "K3 & ADMIN ONLY: Mengubah status untuk komponen Palet.",
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{"new_status": {Type: genai.TypeString, Enum: []string{"Open", "Close"}}},
				Required: []string{"new_status"},
			},
		},
		{
			Name: "update_corepart_status",
			Description: "K3 & ADMIN ONLY: Mengubah status untuk komponen Corepart.",
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{"new_status": {Type: genai.TypeString, Enum: []string{"Open", "Close"}}},
				Required: []string{"new_status"},
			},
		},
	}

	k5Tools := []*genai.FunctionDeclaration{
		{
			Name: "update_busbar_status",
			Description: "K5 & ADMIN ONLY: Mengubah status untuk komponen Busbar PCC atau Busbar MCC.",
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"busbar_type": {Type: genai.TypeString, Enum: []string{"pcc", "mcc"}},
					"new_status":  {Type: genai.TypeString, Enum: []string{"Open", "Punching/Bending", "Plating/Epoxy", "100% Siap Kirim", "Close"}},
				},
				Required: []string{"busbar_type", "new_status"},
			},
		},
		{
			Name: "update_panel_ao_date",
			Description: "K5 & ADMIN ONLY: Mengubah atau mengatur tanggal Acknowledgment Order (AO) untuk Busbar.",
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"new_date": {Type: genai.TypeString, Description: "Tanggal baru dalam format YYYY-MM-DD."},
				},
				Required: []string{"new_date"},
			},
		},
	}

	whsTools := []*genai.FunctionDeclaration{
		{
			Name: "update_component_status",
			Description: "WAREHOUSE & ADMIN ONLY: Mengubah status untuk komponen utama (picking component).",
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{"new_status": {Type: genai.TypeString, Enum: []string{"Open", "On Progress", "Done"}}},
				Required: []string{"new_status"},
			},
		},
	}

	// 2. Berikan "alat" yang sesuai berdasarkan role user
	var availableTools []*genai.FunctionDeclaration
	availableTools = append(availableTools, viewTools...)
	availableTools = append(availableTools, issueEditTools...) 

	switch role {
	case AppRoleAdmin:
		availableTools = append(availableTools, adminTools...)
		availableTools = append(availableTools, k3Tools...)
		availableTools = append(availableTools, k5Tools...)
		availableTools = append(availableTools, whsTools...)
	case AppRoleK3:
		availableTools = append(availableTools, k3Tools...)
	case AppRoleK5:
		availableTools = append(availableTools, k5Tools...)
	case AppRoleWarehouse:
		availableTools = append(availableTools, whsTools...)
	}

	return []*genai.Tool{{FunctionDeclarations: availableTools}}
}


func (a *App) executeDatabaseFunction(fc genai.FunctionCall, panelNoPp string) (string, error) {
	// Helper function serbaguna untuk menjalankan query UPDATE panel
	executeUpdate := func(column string, value interface{}) error {
		query := fmt.Sprintf("UPDATE panels SET %s = $1 WHERE no_pp = $2", column)
		_, err := a.DB.Exec(query, value, panelNoPp)
		return err
	}

	switch fc.Name {
	case "get_panel_summary":
		var panel pdd
		err := a.DB.QueryRow(`SELECT p.percent_progress, p.status_busbar_pcc, p.status_busbar_mcc, p.status_component, p.status_palet, p.status_corepart FROM panels p WHERE p.no_pp = $1`, panelNoPp).Scan(&panel.PercentProgress, &panel.StatusBusbarPcc, &panel.StatusBusbarMcc, &panel.StatusComponent, &panel.StatusPalet, &panel.StatusCorepart)
		if err != nil { return "", fmt.Errorf("gagal mendapatkan detail panel: %w", err) }
		summary := fmt.Sprintf("Progres panel saat ini %.0f%%. Status Busbar PCC: %s, Busbar MCC: %s, Komponen: %s, Palet: %s, Corepart: %s.", panel.PercentProgress.Float64, panel.StatusBusbarPcc.String, panel.StatusBusbarMcc.String, panel.StatusComponent.String, panel.StatusPalet.String, panel.StatusCorepart.String)
		return summary, nil

	case "update_issue_status":
		title, ok1 := fc.Args["issue_title"].(string)
		newStatus, ok2 := fc.Args["new_status"].(string)
		if !ok1 || !ok2 { return "", fmt.Errorf("argumen issue_title atau new_status tidak valid") }

		var issueID int
		var currentLogs Logs
		var currentStatus string
		err := a.DB.QueryRow(`
			SELECT i.id, i.logs, i.status FROM issues i
			JOIN chats ch ON i.chat_id = ch.id
			WHERE ch.panel_no_pp = $1 AND i.title ILIKE $2`, panelNoPp, "%"+title+"%").Scan(&issueID, &currentLogs, &currentStatus)
		if err != nil {
			if err == sql.ErrNoRows { return "", fmt.Errorf("Isu dengan judul yang mengandung '%s' tidak ditemukan di panel ini.", title) }
			return "", fmt.Errorf("Gagal mencari isu: %w", err)
		}

		logAction := "mengubah issue"
		if newStatus != currentStatus {
			if newStatus == "solved" { logAction = "menandai solved" } else if newStatus == "unsolved" { logAction = "membuka kembali issue" }
		}
		newLogEntry := LogEntry{Action: logAction, User: "gemini_ai", Timestamp: time.Now()}
		updatedLogs := append(currentLogs, newLogEntry)
		
		_, err = a.DB.Exec("UPDATE issues SET status = $1, logs = $2 WHERE id = $3", newStatus, updatedLogs, issueID)
		if err != nil { return "", fmt.Errorf("Gagal mengupdate status isu di database: %w", err) }

		return fmt.Sprintf("Status untuk isu '%s' berhasil diubah menjadi '%s'.", title, newStatus), nil

	case "update_panel_progress":
		progress, ok := fc.Args["new_progress"].(float64)
		if !ok { return "", fmt.Errorf("argumen new_progress tidak valid") }
		if progress < 0 || progress > 100 { return "", fmt.Errorf("nilai progres harus antara 0 dan 100") }
		if err := executeUpdate("percent_progress", progress); err != nil { return "", err }
		return fmt.Sprintf("Progres panel berhasil diubah menjadi %.0f%%.", progress), nil

	case "update_busbar_status":
		busbarType, _ := fc.Args["busbar_type"].(string)
		newStatus, _ := fc.Args["new_status"].(string)
		var dbColumn string
		if busbarType == "pcc" { dbColumn = "status_busbar_pcc" } else { dbColumn = "status_busbar_mcc" }
		if err := executeUpdate(dbColumn, newStatus); err != nil { return "", err }
		return fmt.Sprintf("Status Busbar %s berhasil diubah menjadi '%s'.", strings.ToUpper(busbarType), newStatus), nil

	case "update_component_status":
		newStatus, _ := fc.Args["new_status"].(string)
		if err := executeUpdate("status_component", newStatus); err != nil { return "", err }
		return fmt.Sprintf("Status Komponen berhasil diubah menjadi '%s'.", newStatus), nil

	case "update_palet_status":
		newStatus, _ := fc.Args["new_status"].(string)
		if err := executeUpdate("status_palet", newStatus); err != nil { return "", err }
		return fmt.Sprintf("Status Palet berhasil diubah menjadi '%s'.", newStatus), nil

	case "update_corepart_status":
		newStatus, _ := fc.Args["new_status"].(string)
		if err := executeUpdate("status_corepart", newStatus); err != nil { return "", err }
		return fmt.Sprintf("Status Corepart berhasil diubah menjadi '%s'.", newStatus), nil

	case "update_panel_ao_date":
		busbarType, ok1 := fc.Args["busbar_type"].(string)
		newDateStr, ok2 := fc.Args["new_date"].(string)
		if !ok1 || !ok2 { return "", fmt.Errorf("argumen busbar_type atau new_date tidak valid") }

		_, err := time.Parse("2006-01-02", newDateStr)
		if err != nil { return "", fmt.Errorf("format tanggal tidak valid, harus YYYY-MM-DD") }

		var dbColumn string
		if busbarType == "pcc" { dbColumn = "ao_busbar_pcc" } else { dbColumn = "ao_busbar_mcc" }
		
		if err := executeUpdate(dbColumn, newDateStr); err != nil { return "", err }
		return fmt.Sprintf("Oke, tanggal AO untuk Busbar %s sudah diupdate menjadi %s.", strings.ToUpper(busbarType), newDateStr), nil

	case "update_panel_closed_status":
		isClosed, ok := fc.Args["is_closed"].(bool)
		if !ok { return "", fmt.Errorf("argumen is_closed tidak valid") }
		
		if err := executeUpdate("is_closed", isClosed); err != nil { return "", err }
		
		if isClosed {
			dateStr, ok := fc.Args["closed_date"].(string)
			if ok {
				if err := executeUpdate("closed_date", dateStr); err != nil { return "", err }
				return fmt.Sprintf("Sip! Panel ini sudah ditandai sebagai Selesai (Closed) pada tanggal %s.", dateStr), nil
			}
			return "Sip! Panel ini sudah ditandai sebagai Selesai (Closed).", nil
		} else {
			if err := executeUpdate("closed_date", nil); err != nil { return "", err }
			return "Oke, panel ini sudah dibuka kembali statusnya.", nil
		}
	
	case "assign_k3_vendor_to_panel":
		vendorName, ok := fc.Args["vendor_name"].(string)
		if !ok { return "", fmt.Errorf("argumen vendor_name tidak valid") }
		
		var vendorID string
		err := a.DB.QueryRow("SELECT id FROM companies WHERE name ILIKE $1 AND role = 'k3'", vendorName).Scan(&vendorID)
		if err != nil { return "", fmt.Errorf("vendor K3 dengan nama '%s' tidak ditemukan.", vendorName) }
		
		if err := executeUpdate("vendor_id", vendorID); err != nil { return "", err }
		return fmt.Sprintf("Vendor Panel (K3) berhasil diubah menjadi %s.", vendorName), nil

	default:
		return "", fmt.Errorf("fungsi tidak dikenal: %s", fc.Name)
	}
}