// filename: main.go
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
	Panel                Panel    `json:"panel"`
	PanelVendorName      *string  `json:"panel_vendor_name"`
	BusbarVendorNames    *string  `json:"busbar_vendor_names"`
	BusbarVendorIds      []string `json:"busbar_vendor_ids"`
	BusbarRemarks        *string  `json:"busbar_remarks"`
	ComponentVendorNames *string  `json:"component_vendor_names"`
	ComponentVendorIds   []string `json:"component_vendor_ids"`
	PaletVendorNames     *string  `json:"palet_vendor_names"`
	PaletVendorIds       []string `json:"palet_vendor_ids"`
	CorepartVendorNames  *string  `json:"corepart_vendor_names"`
	CorepartVendorIds    []string `json:"corepart_vendor_ids"`
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

	// [PERUBAHAN] Konfigurasi connection pool untuk menjaga koneksi tetap sehat
	// Mengatur jumlah maksimum koneksi yang diizinkan terbuka ke database.
	a.DB.SetMaxOpenConns(25)
	// Mengatur jumlah maksimum koneksi yang boleh dalam keadaan idle (tidak terpakai) di dalam pool.
	a.DB.SetMaxIdleConns(25)
	// Mengatur durasi maksimum sebuah koneksi boleh digunakan sebelum ditutup dan diganti dengan yang baru.
	// Ini adalah kunci untuk mencegah error "bad connection" pada platform cloud.
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
	a.Router.HandleFunc("/company/{id}", a.getCompanyByIdHandler).Methods("GET")
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
	a.Router.HandleFunc("/panel/{no_pp}", a.getPanelByNoPpHandler).Methods("GET")
	a.Router.HandleFunc("/panel/exists/no-pp/{no_pp}", a.isNoPpTakenHandler).Methods("GET")
	a.Router.HandleFunc("/panel/exists/no-panel/{no_panel}", a.isPanelNumberUniqueHandler).Methods("GET")

	// Sub-Panel Parts Management
	a.Router.HandleFunc("/busbar", a.upsertGenericHandler("busbars", &Busbar{})).Methods("POST")
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

func (a *App) upsertPanelHandler(w http.ResponseWriter, r *http.Request) {
	var p Panel
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid payload: "+err.Error())
		return
	}

	query := `
        INSERT INTO panels (no_pp, no_panel, no_wbs, project, percent_progress, start_date, target_delivery, status_busbar_pcc, status_busbar_mcc, status_component, status_palet, status_corepart, ao_busbar_pcc, ao_busbar_mcc, created_by, vendor_id, is_closed, closed_date)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
        ON CONFLICT (no_pp) DO UPDATE SET
        no_panel = EXCLUDED.no_panel, no_wbs = EXCLUDED.no_wbs, project = EXCLUDED.project, percent_progress = EXCLUDED.percent_progress, start_date = EXCLUDED.start_date, target_delivery = EXCLUDED.target_delivery, status_busbar_pcc = EXCLUDED.status_busbar_pcc, status_busbar_mcc = EXCLUDED.status_busbar_mcc, status_component = EXCLUDED.status_component, status_palet = EXCLUDED.status_palet, status_corepart = EXCLUDED.status_corepart, ao_busbar_pcc = EXCLUDED.ao_busbar_pcc, ao_busbar_mcc = EXCLUDED.ao_busbar_mcc, created_by = EXCLUDED.created_by, vendor_id = EXCLUDED.vendor_id, is_closed = EXCLUDED.is_closed, closed_date = EXCLUDED.closed_date`

	_, err := a.DB.Exec(query, p.NoPp, p.NoPanel, p.NoWbs, p.Project, p.PercentProgress, p.StartDate, p.TargetDelivery, p.StatusBusbarPcc, p.StatusBusbarMcc, p.StatusComponent, p.StatusPalet, p.StatusCorepart, p.AoBusbarPcc, p.AoBusbarMcc, p.CreatedBy, p.VendorID, p.IsClosed, p.ClosedDate)
	if err != nil {
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
	if exists {
		respondWithJSON(w, http.StatusOK, map[string]bool{"exists": true})
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
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
        SELECT name, role FROM companies
        WHERE role != 'admin'
        GROUP BY name, role ORDER BY name ASC`

	rows, err := a.DB.Query(query)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	var results []map[string]string
	for rows.Next() {
		var name, role string
		if err := rows.Scan(&name, &role); err != nil {
			respondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
		results = append(results, map[string]string{"name": name, "role": role})
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

	// Jika role tidak dikirim (dari bulk delete), anggap sebagai admin untuk ambil semua data
	if userRole == "" {
		userRole = AppRoleAdmin
	}

	if userRole != AppRoleAdmin && userRole != AppRoleViewer {
		// Logika filter untuk role lain tetap sama
		switch userRole {
		case AppRoleK3:
			panelIdsSubQuery = fmt.Sprintf(`
                SELECT no_pp FROM panels WHERE vendor_id = $%d
                UNION SELECT panel_no_pp FROM palet WHERE vendor = $%d
                UNION SELECT panel_no_pp FROM corepart WHERE vendor = $%d
                UNION SELECT no_pp FROM panels WHERE no_pp NOT IN (SELECT DISTINCT panel_no_pp FROM palet)
                UNION SELECT no_pp FROM panels WHERE no_pp NOT IN (SELECT DISTINCT panel_no_pp FROM corepart)
            `, argCounter, argCounter, argCounter)
			args = append(args, companyId)
		case AppRoleK5:
			panelIdsSubQuery = fmt.Sprintf(`
                SELECT panel_no_pp FROM busbars WHERE vendor = $%d
                UNION SELECT no_pp FROM panels WHERE no_pp NOT IN (SELECT DISTINCT panel_no_pp FROM busbars)
            `, argCounter)
			args = append(args, companyId)
		case AppRoleWarehouse:
			panelIdsSubQuery = fmt.Sprintf(`
                SELECT panel_no_pp FROM components WHERE vendor = $%d
                UNION SELECT no_pp FROM panels WHERE no_pp NOT IN (SELECT DISTINCT panel_no_pp FROM components)
            `, argCounter)
			args = append(args, companyId)
		default:
			respondWithJSON(w, http.StatusOK, []PanelDisplayData{})
			return
		}
	} else {
		panelIdsSubQuery = "SELECT no_pp FROM panels"
	}

	finalQuery := `
        SELECT p.*, pu.name as panel_vendor_name,
            (SELECT STRING_AGG(c.name, ', ') FROM companies c JOIN busbars b ON c.id = b.vendor WHERE b.panel_no_pp = p.no_pp) as busbar_vendor_names,
            (SELECT STRING_AGG(c.id, ',') FROM companies c JOIN busbars b ON c.id = b.vendor WHERE b.panel_no_pp = p.no_pp) as busbar_vendor_ids,
            (SELECT STRING_AGG(b.remarks, '; ') FROM busbars b WHERE b.panel_no_pp = p.no_pp) as busbar_remarks,
            (SELECT STRING_AGG(c.name, ', ') FROM companies c JOIN components co ON c.id = co.vendor WHERE co.panel_no_pp = p.no_pp) as component_vendor_names,
            (SELECT STRING_AGG(c.id, ',') FROM companies c JOIN components co ON c.id = co.vendor WHERE co.panel_no_pp = p.no_pp) as component_vendor_ids,
            (SELECT STRING_AGG(c.name, ', ') FROM companies c JOIN palet pa ON c.id = pa.vendor WHERE pa.panel_no_pp = p.no_pp) as palet_vendor_names,
            (SELECT STRING_AGG(c.id, ',') FROM companies c JOIN palet pa ON c.id = pa.vendor WHERE pa.panel_no_pp = p.no_pp) as palet_vendor_ids,
            (SELECT STRING_AGG(c.name, ', ') FROM companies c JOIN corepart cp ON c.id = cp.vendor WHERE cp.panel_no_pp = p.no_pp) as corepart_vendor_names,
            (SELECT STRING_AGG(c.id, ',') FROM companies c JOIN corepart cp ON c.id = cp.vendor WHERE cp.panel_no_pp = p.no_pp) as corepart_vendor_ids
        FROM panels p LEFT JOIN companies pu ON p.vendor_id = pu.id
        WHERE p.no_pp IN (` + panelIdsSubQuery + `) ORDER BY p.start_date DESC`

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
		err := rows.Scan(
			&pdd.Panel.NoPp, &pdd.Panel.NoPanel, &pdd.Panel.NoWbs, &pdd.Panel.Project, &pdd.Panel.PercentProgress, &pdd.Panel.StartDate, &pdd.Panel.TargetDelivery, &pdd.Panel.StatusBusbarPcc, &pdd.Panel.StatusBusbarMcc, &pdd.Panel.StatusComponent, &pdd.Panel.StatusPalet, &pdd.Panel.StatusCorepart, &pdd.Panel.AoBusbarPcc, &pdd.Panel.AoBusbarMcc, &pdd.Panel.CreatedBy, &pdd.Panel.VendorID, &pdd.Panel.IsClosed, &pdd.Panel.ClosedDate,
			&pdd.PanelVendorName, &pdd.BusbarVendorNames, &busbarVendorIds, &pdd.BusbarRemarks, &pdd.ComponentVendorNames, &componentVendorIds, &pdd.PaletVendorNames, &paletVendorIds, &pdd.CorepartVendorNames, &corepartVendorIds,
		)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Error scanning row: "+err.Error())
			return
		}
		pdd.BusbarVendorIds = splitIds(busbarVendorIds)
		pdd.ComponentVendorIds = splitIds(componentVendorIds)
		pdd.PaletVendorIds = splitIds(paletVendorIds)
		pdd.CorepartVendorIds = splitIds(corepartVendorIds)
		results = append(results, pdd)
	}
	respondWithJSON(w, http.StatusOK, results)
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

	// Menggunakan pq.Array untuk mengirim list string ke query PostgreSQL
	// ON DELETE CASCADE pada schema akan menangani penghapusan di tabel-tabel terkait (busbars, components, dll)
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
	tx, err := a.DB.Begin()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Gagal memulai transaksi")
		return
	}
	result, err := tx.Exec("DELETE FROM panels WHERE no_pp = $1", noPp)
	if err != nil {
		tx.Rollback()
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		tx.Rollback()
		respondWithError(w, http.StatusNotFound, "Panel tidak ditemukan")
		return
	}
	if err := tx.Commit(); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Gagal commit transaksi")
		return
	}
	respondWithJSON(w, http.StatusOK, map[string]string{"status": "success", "message": "Panel " + noPp + " berhasil dihapus"})
}
func (a *App) getAllPanelsHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := a.DB.Query("SELECT no_pp, no_panel, no_wbs, project, percent_progress, start_date, target_delivery, status_busbar_pcc, status_busbar_mcc, status_component, status_palet, status_corepart, ao_busbar_pcc, ao_busbar_mcc, created_by, vendor_id, is_closed, closed_date FROM panels")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()
	var panels []Panel
	for rows.Next() {
		var p Panel
		if err := rows.Scan(&p.NoPp, &p.NoPanel, &p.NoWbs, &p.Project, &p.PercentProgress, &p.StartDate, &p.TargetDelivery, &p.StatusBusbarPcc, &p.StatusBusbarMcc, &p.StatusComponent, &p.StatusPalet, &p.StatusCorepart, &p.AoBusbarPcc, &p.AoBusbarMcc, &p.CreatedBy, &p.VendorID, &p.IsClosed, &p.ClosedDate); err != nil {
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
	err := a.DB.QueryRow("SELECT no_pp, no_panel, no_wbs, project, percent_progress, start_date, target_delivery, status_busbar_pcc, status_busbar_mcc, status_component, status_palet, status_corepart, ao_busbar_pcc, ao_busbar_mcc, created_by, vendor_id, is_closed, closed_date FROM panels WHERE no_pp = $1", noPp).Scan(
		&p.NoPp, &p.NoPanel, &p.NoWbs, &p.Project, &p.PercentProgress, &p.StartDate, &p.TargetDelivery, &p.StatusBusbarPcc, &p.StatusBusbarMcc, &p.StatusComponent, &p.StatusPalet, &p.StatusCorepart, &p.AoBusbarPcc, &p.AoBusbarMcc, &p.CreatedBy, &p.VendorID, &p.IsClosed, &p.ClosedDate)
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
func (a *App) importFromCustomTemplateHandler(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Data             map[string][]map[string]interface{} `json:"data"`
		LoggedInUsername *string                             `json:"loggedInUsername"`
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

	findVendorId := func(vendorName string) (string, error) {
		if strings.TrimSpace(vendorName) == "" {
			return "", nil
		}
		var id string
		err := tx.QueryRow("SELECT id FROM companies WHERE LOWER(name) = LOWER($1)", strings.TrimSpace(vendorName)).Scan(&id)
		if err == sql.ErrNoRows {
			return "", nil
		}
		return id, err
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
				errors = append(errors, fmt.Sprintf("User baris %d: Kolom 'Username' kosong.", rowNum))
				continue
			}
			if companyName == "" {
				errors = append(errors, fmt.Sprintf("User baris %d: Kolom 'Company' kosong.", rowNum))
				continue
			}

			companyId, err := findVendorId(companyName)
			if err != nil {
				errors = append(errors, fmt.Sprintf("User baris %d: Error mencari company: %v", rowNum, err))
				continue
			}
			if companyId == "" {
				companyId = strings.ToLower(strings.ReplaceAll(companyName, " ", "_"))
				_, err := tx.Exec("INSERT INTO companies (id, name, role) VALUES ($1, $2, $3) ON CONFLICT(id) DO NOTHING", companyId, companyName, companyRole)
				if err != nil {
					errors = append(errors, fmt.Sprintf("User baris %d: Gagal membuat company baru '%s': %v", rowNum, companyName, err))
					continue
				}
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
		warehouseCompanyId, err := findVendorId("Warehouse")
		if err != nil || warehouseCompanyId == "" {
			errors = append(errors, "Kritis: Perusahaan 'Warehouse' tidak ada di DB. Component tidak bisa di-assign.")
		}

		for i, row := range panelSheetData {
			rowNum := i + 2
			noPp := getValueCaseInsensitive(row, "PP Panel")
			panelVendorName := getValueCaseInsensitive(row, "Panel")
			busbarVendorName := getValueCaseInsensitive(row, "Busbar")

			if noPp == "" {
				errors = append(errors, fmt.Sprintf("Panel baris %d: Kolom 'PP Panel' wajib diisi.", rowNum))
				continue
			}
			if panelVendorName == "" {
				errors = append(errors, fmt.Sprintf("Panel baris %d: Kolom 'Panel' (untuk vendor K3) wajib diisi.", rowNum))
				continue
			}

			panelK3VendorId, err := findVendorId(panelVendorName)
			if err != nil || panelK3VendorId == "" {
				errors = append(errors, fmt.Sprintf("Panel baris %d: Vendor Panel '%s' tidak ditemukan.", rowNum, panelVendorName))
				continue
			}

			panelMap := map[string]interface{}{
				"no_pp":             noPp,
				"no_panel":          getValueCaseInsensitive(row, "Panel No"),
				"no_wbs":            getValueCaseInsensitive(row, "WBS"),
				"project":           getValueCaseInsensitive(row, "PROJECT"),
				"start_date":        parseDate(getValueCaseInsensitive(row, "Plan Start")),
				"target_delivery":   parseDate(getValueCaseInsensitive(row, "Actual Delivery ke SEC")),
				"vendor_id":         panelK3VendorId,
				"created_by":        "import",
				"percent_progress":  0.0,
				"is_closed":         false,
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
				errors = append(errors, fmt.Sprintf("Panel baris %d: Gagal insert panel: %v", rowNum, err))
				continue
			}

			if err := insertMap(tx, "palet", map[string]interface{}{"panel_no_pp": noPp, "vendor": panelK3VendorId}); err != nil {
				errors = append(errors, fmt.Sprintf("Panel baris %d: Gagal link palet: %v", rowNum, err))
			}
			if err := insertMap(tx, "corepart", map[string]interface{}{"panel_no_pp": noPp, "vendor": panelK3VendorId}); err != nil {
				errors = append(errors, fmt.Sprintf("Panel baris %d: Gagal link corepart: %v", rowNum, err))
			}
			if warehouseCompanyId != "" {
				if err := insertMap(tx, "components", map[string]interface{}{"panel_no_pp": noPp, "vendor": warehouseCompanyId}); err != nil {
					errors = append(errors, fmt.Sprintf("Panel baris %d: Gagal link component: %v", rowNum, err))
				}
			}
			if busbarVendorName != "" {
				busbarVendorId, err := findVendorId(busbarVendorName)
				if err != nil || busbarVendorId == "" {
					errors = append(errors, fmt.Sprintf("Panel baris %d: Vendor Busbar '%s' tidak ditemukan.", rowNum, busbarVendorName))
				} else {
					if err := insertMap(tx, "busbars", map[string]interface{}{"panel_no_pp": noPp, "vendor": busbarVendorId}); err != nil {
						errors = append(errors, fmt.Sprintf("Panel baris %d: Gagal link busbar: %v", rowNum, err))
					}
				}
			}
			dataProcessed = true
		}
	}

	if len(errors) > 0 {
		errorString := strings.Join(errors, "\n- ")
		respondWithError(w, http.StatusBadRequest, "Impor dibatalkan karena error validasi:\n- "+errorString)
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
      no_pp TEXT PRIMARY KEY, no_panel TEXT UNIQUE, no_wbs TEXT, project TEXT, percent_progress REAL,
      start_date TIMESTAMPTZ, target_delivery TIMESTAMPTZ, status_busbar_pcc TEXT, status_busbar_mcc TEXT,
      status_component TEXT, status_palet TEXT, status_corepart TEXT, ao_busbar_pcc TIMESTAMPTZ, ao_busbar_mcc TIMESTAMPTZ,
      created_by TEXT, vendor_id TEXT, is_closed BOOLEAN DEFAULT false, closed_date TIMESTAMPTZ );
    CREATE TABLE IF NOT EXISTS busbars ( id SERIAL PRIMARY KEY, panel_no_pp TEXT NOT NULL REFERENCES panels(no_pp) ON DELETE CASCADE, vendor TEXT NOT NULL, remarks TEXT, UNIQUE(panel_no_pp, vendor) );
    CREATE TABLE IF NOT EXISTS components ( id SERIAL PRIMARY KEY, panel_no_pp TEXT NOT NULL REFERENCES panels(no_pp) ON DELETE CASCADE, vendor TEXT NOT NULL, UNIQUE(panel_no_pp, vendor) );
    CREATE TABLE IF NOT EXISTS palet ( id SERIAL PRIMARY KEY, panel_no_pp TEXT NOT NULL REFERENCES panels(no_pp) ON DELETE CASCADE, vendor TEXT NOT NULL, UNIQUE(panel_no_pp, vendor) );
    CREATE TABLE IF NOT EXISTS corepart ( id SERIAL PRIMARY KEY, panel_no_pp TEXT NOT NULL REFERENCES panels(no_pp) ON DELETE CASCADE, vendor TEXT NOT NULL, UNIQUE(panel_no_pp, vendor) );`
	if _, err := db.Exec(createTablesSQL); err != nil {
		log.Fatalf("Gagal membuat tabel: %v", err)
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
		time.RFC3339,                   // Format standar: 2025-08-04T00:00:00Z
		"2006-01-02T15:04:05Z07:00",     // Format standar lain
		"02-Jan-2006",                  // Format BARU: 04-Aug-2025 (tahun 4 digit)
		"02-Jan-06",                    // Format lama: 04-Aug-25 (tahun 2 digit)
		"1/2/2006",                     // Format: 8/4/2025
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
