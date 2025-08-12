package main

import (
	"database/sql"
	"database/sql/driver"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/lib/pq"
	"github.com/xuri/excelize/v2"
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
	a.Router = mux.NewRouter()
	a.initializeRoutes()
}

func (a *App) Run(addr string) {
	log.Printf("Server berjalan di %s", addr)
	log.Fatal(http.ListenAndServe(addr, jsonContentTypeMiddleware(a.Router)))
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
	a.Router.HandleFunc("/panels/bulk-delete", a.deletePanelsHandler).Methods("DELETE")
	a.Router.HandleFunc("/panels/{no_pp}", a.deletePanelHandler).Methods("DELETE")
	a.Router.HandleFunc("/panels/all", a.getAllPanelsHandler).Methods("GET")
	a.Router.HandleFunc("/panels/keys", a.getPanelKeysHandler).Methods("GET")
	a.Router.HandleFunc("/panel/{no_pp}", a.getPanelByNoPpHandler).Methods("GET")
	a.Router.HandleFunc("/panel/exists/no-pp/{no_pp}", a.isNoPpTakenHandler).Methods("GET")
	a.Router.HandleFunc("/panels/{old_no_pp}/change-pp", a.changePanelNoPpHandler).Methods("PUT")

	// Rute isPanelNumberUniqueHandler tidak lagi diperlukan karena no_panel tidak unik
	// a.Router.HandleFunc("/panel/exists/no-panel/{no_panel}", a.isPanelNumberUniqueHandler).Methods("GET")

	// Sub-Panel Parts Management
	a.Router.HandleFunc("/busbar", a.upsertGenericHandler("busbars", &Busbar{})).Methods("POST")
	a.Router.HandleFunc("/busbar", a.deleteBusbarHandler).Methods("DELETE") 

	a.Router.HandleFunc("/component", a.upsertGenericHandler("components", &Component{})).Methods("POST")
	a.Router.HandleFunc("/palet", a.upsertGenericHandler("palet", &Palet{})).Methods("POST")
	a.Router.HandleFunc("/corepart", a.upsertGenericHandler("corepart", &Corepart{})).Methods("POST")
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
			respondWithError(w, http.StatusNotFound, "Company not found")
			return
		}
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
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
			panelIdsSubQuery = fmt.Sprintf(`SELECT no_pp FROM panels WHERE vendor_id = $%d UNION SELECT panel_no_pp FROM palet WHERE vendor = $%d UNION SELECT panel_no_pp FROM corepart WHERE vendor = $%d`, argCounter, argCounter, argCounter)
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

	rawIDs := r.URL.Query().Get("raw_ids") == "true"

	finalQuery := fmt.Sprintf(`
		SELECT
			p.no_pp, p.no_panel, p.no_wbs, p.project, p.percent_progress, p.start_date, p.target_delivery,
			p.status_busbar_pcc, p.status_busbar_mcc, p.status_component, p.status_palet,
			p.status_corepart, p.ao_busbar_pcc, p.ao_busbar_mcc, p.created_by, p.vendor_id,
			p.is_closed, p.closed_date, p.panel_type,
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
			&panel.IsClosed, &panel.ClosedDate, &panel.PanelType,
			&pdd.PanelVendorName, &pdd.BusbarVendorNames, &busbarVendorIds, &pdd.BusbarRemarks,
			&pdd.ComponentVendorNames, &componentVendorIds, &pdd.PaletVendorNames, &paletVendorIds,
			&pdd.CorepartVendorNames, &corepartVendorIds,
		)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Error scanning row: "+err.Error())
			return
		}
		
		if !rawIDs && strings.HasPrefix(panel.NoPp, "TEMP_PP_") {
			panel.NoPp = ""
		}

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
	if exists {
		respondWithJSON(w, http.StatusOK, map[string]bool{"exists": true})
		return
	}
	w.WriteHeader(http.StatusNotFound)
}

func (a *App) changePanelNoPpHandler(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    oldNoPp, ok := vars["old_no_pp"]
    if !ok {
        respondWithError(w, http.StatusBadRequest, "No. PP lama tidak ditemukan di URL")
        return
    }

    var payload struct {
        NewNoPp string `json:"new_no_pp"`
    }
    if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
        respondWithError(w, http.StatusBadRequest, "Payload tidak valid: "+err.Error())
        return
    }

    newNoPp := payload.NewNoPp
    if newNoPp == "" {
        respondWithError(w, http.StatusBadRequest, "No. PP baru tidak boleh kosong")
        return
    }

    // Mulai Transaksi Database
    tx, err := a.DB.Begin()
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, "Gagal memulai transaksi: "+err.Error())
        return
    }
    // Pastikan transaksi di-rollback jika terjadi error
    defer tx.Rollback()

    // 1. Cek apakah No. PP baru sudah digunakan oleh panel lain
    var exists bool
    err = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM panels WHERE no_pp = $1)", newNoPp).Scan(&exists)
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, "Gagal memvalidasi No. PP baru: "+err.Error())
        return
    }
    if exists {
        respondWithError(w, http.StatusConflict, fmt.Sprintf("No. PP '%s' sudah digunakan. Pilih nomor lain.", newNoPp))
        return
    }

    // 2. Update semua tabel anak yang memiliki relasi ke panels
    // Abaikan error jika tidak ada baris yang ter-update
    tablesToUpdate := []string{"busbars", "components", "palet", "corepart"}
    for _, table := range tablesToUpdate {
        query := fmt.Sprintf("UPDATE %s SET panel_no_pp = $1 WHERE panel_no_pp = $2", table)
        if _, err := tx.Exec(query, newNoPp, oldNoPp); err != nil {
            respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Gagal update tabel %s: %v", table, err))
            return
        }
    }

    // 3. Setelah semua anak di-update, update tabel induk (panels)
    _, err = tx.Exec("UPDATE panels SET no_pp = $1 WHERE no_pp = $2", newNoPp, oldNoPp)
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, "Gagal update tabel panels: "+err.Error())
        return
    }

    // 4. Jika semua berhasil, commit transaksi
    if err := tx.Commit(); err != nil {
        respondWithError(w, http.StatusInternalServerError, "Gagal commit transaksi: "+err.Error())
        return
    }

    respondWithJSON(w, http.StatusOK, map[string]string{"status": "success", "message": "No. PP berhasil diubah"})
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

// --- [PERUBAHAN] Fungsi getFilteredDataForExport diubah untuk mencegah nilai 'null' pada JSON ---
func (a *App) getFilteredDataForExport(userRole, companyId string) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	var relevantPanelIds []string

	tx, err := a.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if userRole != AppRoleAdmin && userRole != AppRoleViewer {
		var panelIdRows *sql.Rows
		var query string
		switch userRole {
		case AppRoleK3:
			query = `
                        SELECT no_pp FROM panels WHERE vendor_id = $1
                        UNION SELECT panel_no_pp FROM palet WHERE vendor = $1
                        UNION SELECT panel_no_pp FROM corepart WHERE vendor = $1`
			panelIdRows, err = tx.Query(query, companyId)
		case AppRoleK5:
			query = "SELECT panel_no_pp FROM busbars WHERE vendor = $1"
			panelIdRows, err = tx.Query(query, companyId)
		case AppRoleWarehouse:
			query = "SELECT panel_no_pp FROM components WHERE vendor = $1"
			panelIdRows, err = tx.Query(query, companyId)
		}
		if err != nil {
			return nil, err
		}
		defer panelIdRows.Close()
		idSet := make(map[string]bool)
		for panelIdRows.Next() {
			var id string
			if err := panelIdRows.Scan(&id); err == nil {
				idSet[id] = true
			}
		}
		for id := range idSet {
			relevantPanelIds = append(relevantPanelIds, id)
		}
	}

	if userRole == AppRoleAdmin || userRole == AppRoleViewer {
		result["companies"], _ = fetchAllAs(tx, "companies", func() interface{} { return &Company{} })
		result["companyAccounts"], _ = fetchAllAs(tx, "company_accounts", func() interface{} { return &CompanyAccount{} })
		result["panels"], _ = fetchAllAs(tx, "panels", func() interface{} { return &Panel{} })
		result["busbars"], _ = fetchAllAs(tx, "busbars", func() interface{} { return &Busbar{} })
		result["components"], _ = fetchAllAs(tx, "components", func() interface{} { return &Component{} })
		result["palet"], _ = fetchAllAs(tx, "palet", func() interface{} { return &Palet{} })
		result["corepart"], _ = fetchAllAs(tx, "corepart", func() interface{} { return &Corepart{} })
	} else {
		var companies []Company
		row := tx.QueryRow("SELECT id, name, role FROM companies WHERE id = $1", companyId)
		var c Company
		if err := row.Scan(&c.ID, &c.Name, &c.Role); err == nil {
			companies = append(companies, c)
		}
		result["companies"] = companies

		var accounts []CompanyAccount
		accRows, _ := tx.Query("SELECT username, password, company_id FROM company_accounts WHERE company_id = $1", companyId)
		if accRows != nil {
			defer accRows.Close()
			for accRows.Next() {
				var acc CompanyAccount
				if err := accRows.Scan(&acc.Username, &acc.Password, &acc.CompanyID); err == nil {
					accounts = append(accounts, acc)
				}
			}
		}
		result["companyAccounts"] = accounts

		if len(relevantPanelIds) > 0 {
			result["panels"], _ = fetchInAs(tx, "panels", "no_pp", relevantPanelIds, func() interface{} { return &Panel{} })
			result["busbars"], _ = fetchInAs(tx, "busbars", "panel_no_pp", relevantPanelIds, func() interface{} { return &Busbar{} })
			result["components"], _ = fetchInAs(tx, "components", "panel_no_pp", relevantPanelIds, func() interface{} { return &Component{} })
			result["palet"], _ = fetchInAs(tx, "palet", "panel_no_pp", relevantPanelIds, func() interface{} { return &Palet{} })
			result["corepart"], _ = fetchInAs(tx, "corepart", "panel_no_pp", relevantPanelIds, func() interface{} { return &Corepart{} })
		} else {
			result["panels"], result["busbars"], result["components"], result["palet"], result["corepart"] =
				[]Panel{}, []Busbar{}, []Component{}, []Palet{}, []Corepart{}
		}
	}

	// [FIX] Memastikan tidak ada key yang memiliki nilai nil; ganti dengan slice kosong.
	// Ini membuat respons API konsisten dan mencegah error di frontend.
	keys := []string{"companies", "companyAccounts", "panels", "busbars", "components", "palet", "corepart"}
	for _, key := range keys {
		if result[key] == nil {
			result[key] = []interface{}{}
		}
	}

	return result, nil
}

// --- [AKHIR PERUBAHAN] ---

func (a *App) getFilteredDataForExportHandler(w http.ResponseWriter, r *http.Request) {
	userRole := r.URL.Query().Get("role")
	companyId := r.URL.Query().Get("company_id")
	data, err := a.getFilteredDataForExport(userRole, companyId)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondWithJSON(w, http.StatusOK, data)
}
func (a *App) generateCustomExportJsonHandler(w http.ResponseWriter, r *http.Request) {
	userRole := r.URL.Query().Get("role")
	companyId := r.URL.Query().Get("company_id")
	includePanels, _ := strconv.ParseBool(r.URL.Query().Get("panels"))
	includeUsers, _ := strconv.ParseBool(r.URL.Query().Get("users"))

	data, err := a.getFilteredDataForExport(userRole, companyId)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	allCompaniesResult, _ := fetchAllAs(a.DB, "companies", func() interface{} { return &Company{} })
	allCompanies := allCompaniesResult.([]interface{})

	companyMap := make(map[string]Company)
	for _, c := range allCompanies {
		company := c.(*Company)
		companyMap[company.ID] = *company
	}

	jsonData := make(map[string]interface{})

	if includePanels {
		panels := data["panels"].([]interface{})
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
		for _, p := range panels {
			panel := p.(*Panel)
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
func (a *App) generateFilteredDatabaseJsonHandler(w http.ResponseWriter, r *http.Request) {
	userRole := r.URL.Query().Get("role")
	companyId := r.URL.Query().Get("company_id")
	tablesParam := r.URL.Query().Get("tables")
	tablesToInclude := strings.Split(tablesParam, ",")

	data, err := a.getFilteredDataForExport(userRole, companyId)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	jsonData := make(map[string]interface{})
	for _, table := range tablesToInclude {
		switch strings.ToLower(strings.ReplaceAll(table, " ", "")) {
		case "companies":
			jsonData["companies"] = data["companies"]
		case "companyaccounts":
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
			panelVendorInput := getValueCaseInsensitive(row, "Panel")
			busbarVendorInput := getValueCaseInsensitive(row, "Busbar")

			if noPp == "" {
				errors = append(errors, fmt.Sprintf("Panel baris %d: Kolom 'PP Panel' wajib diisi.", rowNum))
				continue
			}

			var panelVendorId, busbarVendorId sql.NullString
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

			lastErrorIndex := len(errors) - 1
			if lastErrorIndex >= 0 && strings.Contains(errors[lastErrorIndex], fmt.Sprintf("baris %d", rowNum)) {
				continue
			}

			actualDeliveryDate := parseDate(getValueCaseInsensitive(row, "Actual Delivery ke SEC"))

			// [PERBAIKAN 2] Tambahkan logika untuk progress
			var progress float64 = 0.0
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
				"is_closed":         actualDeliveryDate != nil,
				"vendor_id":         panelVendorId,
				"created_by":        "import",
				"percent_progress":  progress, // Gunakan variabel progress di sini
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
      -- Kolom panel_type sengaja tidak ditambahkan di sini agar migrasi berjalan
    );
    CREATE TABLE IF NOT EXISTS busbars ( id SERIAL PRIMARY KEY, panel_no_pp TEXT NOT NULL REFERENCES panels(no_pp) ON DELETE CASCADE, vendor TEXT NOT NULL, remarks TEXT, UNIQUE(panel_no_pp, vendor) );
    CREATE TABLE IF NOT EXISTS components ( id SERIAL PRIMARY KEY, panel_no_pp TEXT NOT NULL REFERENCES panels(no_pp) ON DELETE CASCADE, vendor TEXT NOT NULL, UNIQUE(panel_no_pp, vendor) );
    CREATE TABLE IF NOT EXISTS palet ( id SERIAL PRIMARY KEY, panel_no_pp TEXT NOT NULL REFERENCES panels(no_pp) ON DELETE CASCADE, vendor TEXT NOT NULL, UNIQUE(panel_no_pp, vendor) );
    CREATE TABLE IF NOT EXISTS corepart ( id SERIAL PRIMARY KEY, panel_no_pp TEXT NOT NULL REFERENCES panels(no_pp) ON DELETE CASCADE, vendor TEXT NOT NULL, UNIQUE(panel_no_pp, vendor) );`

	if _, err := db.Exec(createTablesSQL); err != nil {
		log.Fatalf("Gagal membuat tabel awal: %v", err)
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

	// [FIX] SCRIPT MIGRASI UNTUK MENAMBAHKAN KOLOM panel_type
	// Script ini memastikan kolom 'panel_type' ada di tabel 'panels', baik untuk database baru maupun yang sudah ada.
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


	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM companies").Scan(&count); err != nil {
		log.Fatalf("Gagal cek data dummy: %v", err)
	}
	if count == 0 {
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
		time.RFC3339,                 // Format standar: 2025-08-04T00:00:00Z
		"2006-01-02T15:04:05Z07:00",   // Format standar lain
		"02-Jan-2006",                 // Format BARU: 04-Aug-2025 (tahun 4 digit)
		"02-Jan-06",                   // Format lama: 04-Aug-25 (tahun 2 digit)
		"1/2/2006",                    // Format: 8/4/2025
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