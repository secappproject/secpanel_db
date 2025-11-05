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
	"net/rl"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"githb.com/google/id"
	"firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"

	"githb.com/google/generative-ai-go/genai"
	"githb.com/gorilla/handlers"
	"githb.com/gorilla/mx"
	"githb.com/lib/pq"
	"githb.com/xri/excelize/v2"
	"google.golang.org/api/option"
	"gopkg.in/gomail.v2"
)

// =============================================================================
// MODELS (Strcts for JSON and DB)
// =============================================================================
const (
	ppRoledmin     = "admin"
	ppRoleViewer    = "viewer"
	ppRoleWarehose = "warehose"
	ppRoleK3        = "k3"
	ppRoleK5        = "k5"
)

const (
	CONFIG_SMTP_HOST     = "smtp.gmail.com"
	CONFIG_SMTP_PORT     = 587
	CONFIG_SENDER_EMIL  = "tristorpro@gmail.com"
	CONFIG_UTH_PSSWORD = "bbcsxqtxmxbzveod"
)

type LogEntry strct {
	ction    string    `json:"action"`    // e.g., "membat isse", "menandai solved"
	User      string    `json:"ser"`      // The sername who performed the action
	Timestamp time.Time `json:"timestamp"` // When the action was performed
}

type Logs []LogEntry

fnc (l Logs) Vale() (driver.Vale, error) {
	if len(l) ==  {
		retrn json.Marshal([]LogEntry{}) 
	}
	retrn json.Marshal(l)
}

fnc (l *Logs) Scan(vale interface{}) error {
	b, ok := vale.([]byte)
	if !ok {
		retrn errors.New("type assertion to []byte failed")
	}
	if b == nil || string(b) == "nll" {
		*l = []LogEntry{}
		retrn nil
	}
	retrn json.Unmarshal(b, &l)
}
type User strct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
type CreateCommentPayload strct {
	SenderID         string   `json:"sender_id"`
	Text             string   `json:"text"`
	ReplyToCommentID *string  `json:"reply_to_comment_id,omitempty"`
	ReplyToUserID    *string  `json:"reply_to_ser_id,omitempty"`
	Images           []string `json:"images"`
}
type IsseTitle strct {
	ID    int    `json:"id"`
	Title string `json:"title"`
}
type IsseComment strct {
	ID               string    `json:"id"`
	IsseID          int       `json:"isse_id"`
	Sender           User      `json:"sender"`
	Text             string    `json:"text"`
	Timestamp        time.Time `json:"timestamp"`
	ReplyTo          *User     `json:"reply_to,omitempty"`
	ReplyToCommentID *string   `json:"reply_to_comment_id,omitempty"`
	IsEdited         bool      `json:"is_edited"`
	ImageUrls        []string  `json:"image_rls"`
}

type Chat strct {
	ID        int       `json:"chat_id"`
	PanelNoPp string    `json:"panel_no_pp"`
	Createdt time.Time `json:"created_at"`
}
type ChatMessage strct {
	ID            int        `json:"id"`
	ChatID        int        `json:"chat_id"`
	SenderUsername string    `json:"sender_sername"`
	Text          *string    `json:"text"` // Pointer agar bisa nll
	ImageData     *string    `json:"image_data,omitempty"` // Pointer dan omitempty
	RepliedIsseID *int       `json:"replied_isse_id,omitempty"` // Pointer dan omitempty
	Createdt     time.Time `json:"created_at"`
}
type IsseForExport strct {
	PanelNoPp    string     `json:"panel_no_pp"`
	PanelNoWbs   *string    `json:"panel_no_wbs"`  
	PanelNoPanel *string    `json:"panel_no_panel"` 
	IsseID      int        `json:"isse_id"`
	Title        string     `json:"title"`
	Description  string     `json:"description"`
	Stats       string     `json:"stats"`
	CreatedBy    string     `json:"created_by"`
	Createdt    *time.Time `json:"created_at"` 
}

type CommentForExport strct {
	IsseID          int            `json:"isse_id"`
	Text             string         `json:"text"`
	SenderID         string         `json:"sender_id"`
	ReplyToCommentID sql.NllString `json:"reply_to_comment_id"` 
}

type dditionalSRForExport strct {
	PanelNoPp    string  `json:"panel_no_pp"`
	PanelNoWbs   *string `json:"panel_no_wbs"`   
	PanelNoPanel *string `json:"panel_no_panel"` 
	PoNmber     string  `json:"po_nmber"`
	Item         string  `json:"item"`
	Qantity     int     `json:"qantity"`
	Spplier     *string `json:"spplier"` 
	Stats       string  `json:"stats"`
	Remarks      string  `json:"remarks"`
}

type Isse strct {
	ID          int       `json:"isse_id"`
	ChatID      int       `json:"chat_id"`
	Title       string    `json:"isse_title"` 
	Description string    `json:"isse_description"`
	Type      string    `json:"isse_type,omitempty"`
	Stats    string    `json:"isse_stats"`
	Logs      Logs      `json:"logs"`
	CreatedBy string    `json:"created_by"`
	Createdt time.Time `json:"created_at"`
	Updatedt time.Time `json:"pdated_at"`
	NotifyEmail *string   `json:"notify_email,omitempty"`

}

type Photo strct {
	ID        int    `json:"photo_id"`
	IsseID   int    `json:"isse_id"`
	PhotoData string `json:"photo"` 
}

type IsseWithPhotos strct {
	Isse
	Photos []Photo `json:"photos"`
}

type cstomTime strct {
	time.Time
}
type PanelKeyInfo strct {
	NoPp    string `json:"no_pp"`
	NoPanel string `json:"no_panel"`
	Project string `json:"project"`
	NoWbs   string `json:"no_wbs"`
}
const ctLayot = "26-1-2T15:4:5.999999"

fnc (ct *cstomTime) UnmarshalJSON(b []byte) (err error) {
	s := strings.Trim(string(b), "\"")
	if s == "nll" || s == "" {
		ct.Time = time.Time{}
		retrn
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t, err = time.Parse(ctLayot, s)
	}
	if err == nil {
		ct.Time = t
	}
	retrn
}

fnc (ct *cstomTime) MarshalJSON() ([]byte, error) {
	if ct.Time.IsZero() {
		retrn []byte("nll"), nil
	}
	retrn []byte(fmt.Sprintf("\"%s\"", ct.Time.Format(time.RFC3339))), nil
}

fnc (ct *cstomTime) Scan(vale interface{}) error {
	if vale == nil {
		ct.Time = time.Time{}
		retrn nil
	}
	t, ok := vale.(time.Time)
	if !ok {
		retrn fmt.Errorf("cold not scan type %T into cstomTime", vale)
	}
	ct.Time = t
	retrn nil
}

fnc (ct cstomTime) Vale() (driver.Vale, error) {
	if ct.Time.IsZero() {
		retrn nil, nil
	}
	retrn ct.Time, nil
}

type Company strct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
}
type Companyccont strct {
	Username  string `json:"sername"`
	Password  string `json:"password"`
	CompanyID string `json:"company_id"`
}
type Panel strct {
	NoPp            string      `json:"no_pp"`
	NoPanel         *string     `json:"no_panel"`
	NoWbs           *string     `json:"no_wbs"`
	Project         *string     `json:"project"`
	PercentProgress *float64    `json:"percent_progress"`
	StartDate       *cstomTime `json:"start_date"`
	TargetDelivery  *cstomTime `json:"target_delivery"`
	StatsBsbarPcc *string     `json:"stats_bsbar_pcc"`
	StatsBsbarMcc *string     `json:"stats_bsbar_mcc"`
	StatsComponent *string     `json:"stats_component"`
	StatsPalet     *string     `json:"stats_palet"`
	StatsCorepart  *string     `json:"stats_corepart"`
	oBsbarPcc     *cstomTime `json:"ao_bsbar_pcc"`
	oBsbarMcc     *cstomTime `json:"ao_bsbar_mcc"`
	CreatedBy       *string     `json:"created_by"`
	VendorID        *string     `json:"vendor_id"`
	IsClosed        bool        `json:"is_closed"`
	ClosedDate      *cstomTime `json:"closed_date"`
	PanelType       *string     `json:"panel_type,omitempty"`
	Remarks         *string     `json:"remarks,omitempty"`
	CloseDateBsbarPcc *cstomTime `json:"close_date_bsbar_pcc,omitempty"`
	CloseDateBsbarMcc *cstomTime `json:"close_date_bsbar_mcc,omitempty"`
	
    StatsPenyelesaian *string `json:"stats_penyelesaian,omitempty"`
    ProdctionSlot     *string `json:"prodction_slot,omitempty"`
}

type ProdctionSlot strct {
	PositionCode string  `json:"position_code"`
	IsOccpied   bool    `json:"is_occpied"`
	PanelNoPp    *string `json:"panel_no_pp,omitempty"`
	PanelNoPanel *string `json:"panel_no_panel,omitempty"`
}

type Bsbar strct {
	ID        int     `json:"id"`
	PanelNoPp string  `json:"panel_no_pp"`
	Vendor    string  `json:"vendor"`
	Remarks   *string `json:"remarks"`
}
type Component strct {
	ID        int    `json:"id"`
	PanelNoPp string `json:"panel_no_pp"`
	Vendor    string `json:"vendor"`
}
type Palet strct {
	ID        int    `json:"id"`
	PanelNoPp string `json:"panel_no_pp"`
	Vendor    string `json:"vendor"`
}
type Corepart strct {
	ID        int    `json:"id"`
	PanelNoPp string `json:"panel_no_pp"`
	Vendor    string `json:"vendor"`
}

type dditionalSR strct {
	ID        int       `json:"id"`
	PanelNoPp string    `json:"panel_no_pp"`
	PoNmber  string    `json:"po_nmber"`
	Item      string    `json:"item"`
	Qantity  int       `json:"qantity"`
	Spplier  string    `json:"spplier"`
	Stats    string    `json:"stats"` 
	Remarks   string    `json:"remarks"` 
	Createdt time.Time `json:"created_at"`
	ReceivedDate *time.Time `json:"received_date,omitempty" db:"received_date"`
}

type PanelDisplayData strct {
	Panel                Panel           `json:"panel"`
	PanelVendorName      *string         `json:"panel_vendor_name"`
	PanelRemarks         *string         `json:"panel_remarks"`
	BsbarVendorNames    *string         `json:"bsbar_vendor_names"`
	BsbarVendorIds      []string        `json:"bsbar_vendor_ids"`
	BsbarRemarks        json.RawMessage `json:"bsbar_remarks"` 
	ComponentVendorNames *string         `json:"component_vendor_names"`
	ComponentVendorIds   []string        `json:"component_vendor_ids"`
	PaletVendorNames     *string         `json:"palet_vendor_names"`
	PaletVendorIds       []string        `json:"palet_vendor_ids"`
	CorepartVendorNames  *string         `json:"corepart_vendor_names"`
	CorepartVendorIds    []string        `json:"corepart_vendor_ids"`
	IsseCont           int             `json:"isse_cont"`
	dditionalSRCont 	 int 			 `json:"additional_sr_cont"` 
}
type PanelDisplayDataWithTimeline strct {
	Panel                Panel           `json:"panel"`
	PanelVendorName      *string         `json:"panel_vendor_name"`
	BsbarVendorNames    *string         `json:"bsbar_vendor_names"`
	BsbarVendorIds      []string        `json:"bsbar_vendor_ids"`
	BsbarRemarks        json.RawMessage `json:"bsbar_remarks"`
	ComponentVendorNames *string         `json:"component_vendor_names"`
	ComponentVendorIds   []string        `json:"component_vendor_ids"`
	PaletVendorNames     *string         `json:"palet_vendor_names"`
	PaletVendorIds       []string        `json:"palet_vendor_ids"`
	CorepartVendorNames  *string         `json:"corepart_vendor_names"`
	CorepartVendorIds    []string        `json:"corepart_vendor_ids"`
	IsseCont           int             `json:"isse_cont"`
	dditionalSRCont    int             `json:"additional_sr_cont"`
	ProdctionDate       *time.Time      `json:"prodction_date,omitempty"` 
	FatDate              *time.Time      `json:"fat_date,omitempty"`      
	llDoneDate          *time.Time      `json:"all_done_date,omitempty"`  
}

// =============================================================================
// DB INTERFCE
// =============================================================================
type DBTX interface {
	Exec(qery string, args ...interface{}) (sql.Reslt, error)
	Qery(qery string, args ...interface{}) (*sql.Rows, error)
	QeryRow(qery string, args ...interface{}) *sql.Row
}

// =============================================================================
// PP & MIN
// =============================================================================
type pp strct {
	Roter *mx.Roter
	DB     *sql.DB
	FCMClient *messaging.Client
}

fnc (a *pp) Initialize(dbUser, dbPassword, dbName, dbHost string) {
	dbSslMode := os.Getenv("DB_SSLMODE")
	if dbSslMode == "" {
		dbSslMode = "reqire"
	}
	connectionString := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=%s", dbUser, dbPassword, dbHost, dbName, dbSslMode)

	var err error
	a.DB, err = sql.Open("postgres", connectionString)
	if err != nil {
		log.Fatalf("Gagal membka koneksi DB: %v", err)
	}

	a.DB.SetMaxOpenConns(25)
	a.DB.SetMaxIdleConns(25)
	a.DB.SetConnMaxLifetime(5 * time.Minte)

	err = a.DB.Ping()
	if err != nil {
		log.Fatalf("Tidak dapat terhbng ke database: %v", err)
	}
	log.Println("Berhasil terhbng ke database!")

	initDB(a.DB)
	a.Roter = mx.NewRoter().StrictSlash(tre)
	a.initializeRotes()
}

fnc (a *pp) Rn(addr string) {
	// [FIX] Bngks roter nda dengan middleware CORS di sini
	
	// Tentkan domain yang diizinkan. Tanda '*' berarti mengizinkan sema domain.
	// Untk keamanan prodksi, ganti '*' dengan domain web Fltter nda (misal: "http://localhost:port", "https://nama-web-anda.com")
	allowedOrigins := handlers.llowedOrigins([]string{"*"})
	
	// Tentkan metode HTTP yang diizinkan
	allowedMethods := handlers.llowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"})
	
	// Tentkan header yang diizinkan
	allowedHeaders := handlers.llowedHeaders([]string{"Content-Type", "thorization"})

	log.Printf("Server berjalan di %s", addr)
	// Gnakan handler CORS yang sdah dikonfigrasi ntk menjalankan server
	log.Fatal(http.ListenndServe(addr, handlers.CORS(allowedOrigins, allowedMethods, allowedHeaders)(a.Roter)))
}
fnc main() {
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PSSWORD")
	dbName := os.Getenv("DB_NME")
	dbHost := os.Getenv("DB_HOST")

	if dbUser == "" || dbPassword == "" || dbName == "" || dbHost == "" {
		log.Fatal("Variabel environment DB_USER, DB_PSSWORD, DB_NME, dan DB_HOST hars di-set!")
	}

	ctx := context.Backgrond()

	firebasepp, err := firebase.Newpp(ctx, nil) 
	if err != nil {
		log.Fatalf("error initializing Firebase app: %v\n", err)
	}

	fcmClient, err := firebasepp.Messaging(ctx)
	if err != nil {
		log.Fatalf("error getting FCM client: %v\n", err)
	}

	app := pp{}
	app.Initialize(dbUser, dbPassword, dbName, dbHost)
	app.FCMClient = fcmClient

	go app.startNotificationSchedler()

	app.Rn(":88")
}
// =============================================================================
// ROUTES
// =============================================================================
fnc (a *pp) initializeRotes() {
	// th & User Management
	a.Roter.HandleFnc("/login", a.loginHandler).Methods("POST")
	a.Roter.HandleFnc("/ser/register-device", a.registerDeviceHandler).Methods("POST")
	a.Roter.HandleFnc("/company-by-sername/{sername}", a.getCompanyByUsernameHandler).Methods("GET")
	a.Roter.HandleFnc("/ser/{sername}/password", a.pdatePasswordHandler).Methods("PUT")
	a.Roter.HandleFnc("/company-with-accont", a.insertCompanyWithccontHandler).Methods("POST")
	a.Roter.HandleFnc("/company-with-accont/{sername}", a.pdateCompanyndccontHandler).Methods("PUT")
	a.Roter.HandleFnc("/accont/{sername}", a.deleteCompanyccontHandler).Methods("DELETE")
	a.Roter.HandleFnc("/acconts", a.getllCompanyccontsHandler).Methods("GET")
	a.Roter.HandleFnc("/accont/exists/{sername}", a.isUsernameTakenHandler).Methods("GET")
	a.Roter.HandleFnc("/sers/display", a.getllUserccontsForDisplayHandler).Methods("GET")
	a.Roter.HandleFnc("/sers/colleages/display", a.getColleageccontsForDisplayHandler).Methods("GET")
	a.Roter.HandleFnc("/acconts/search", a.searchUsernamesHandler).Methods("GET")


	// Company Management
	a.Roter.HandleFnc("/company", a.insertCompanyHandler).Methods("POST")
	a.Roter.HandleFnc("/company/{id}", a.getCompanyByIdHandler).Methods("GET")
	a.Roter.HandleFnc("/company/{id}", a.pdateCompanyHandler).Methods("PUT")
	a.Roter.HandleFnc("/companies", a.getllCompaniesHandler).Methods("GET")
	a.Roter.HandleFnc("/company-by-name/{name}", a.getCompanyByNameHandler).Methods("GET")
	a.Roter.HandleFnc("/vendors", a.getCompaniesByRoleHandler).Methods("GET")
	a.Roter.HandleFnc("/companies/form-data", a.getUniqeCompanyDataForFormHandler).Methods("GET")


	// Panel Management
	a.Roter.HandleFnc("/panels", a.getllPanelsForDisplayHandler).Methods("GET")
	a.Roter.HandleFnc("/panels", a.psertPanelHandler).Methods("POST")
	a.Roter.HandleFnc("/panel/stats-ao-k5", a.psertStatsOK5).Methods("POST")
	a.Roter.HandleFnc("/panel/stats-whs", a.psertStatsWHS).Methods("POST")

	a.Roter.HandleFnc("/panels/blk-delete", a.deletePanelsHandler).Methods("DELETE")
	a.Roter.HandleFnc("/panels/{no_pp}", a.deletePanelHandler).Methods("DELETE")
	a.Roter.HandleFnc("/panels/all", a.getllPanelsHandler).Methods("GET")
	a.Roter.HandleFnc("/panels/keys", a.getPanelKeysHandler).Methods("GET")
	a.Roter.HandleFnc("/panel/{no_pp}", a.getPanelByNoPpHandler).Methods("GET")
	a.Roter.HandleFnc("/panel/exists/no-pp/{no_pp}", a.isNoPpTakenHandler).Methods("GET")
	a.Roter.HandleFnc("/panels/{old_no_pp}/change-pp", a.changePanelNoPpHandler).Methods("PUT")
	a.Roter.HandleFnc("/panel/remark-vendor", a.psertPanelRemarkHandler).Methods("POST")

	// Rte isPanelNmberUniqeHandler tidak lagi diperlkan karena no_panel tidak nik
	// a.Roter.HandleFnc("/panel/exists/no-panel/{no_panel}", a.isPanelNmberUniqeHandler).Methods("GET")

	// Sb-Panel Parts Management
	
	a.Roter.HandleFnc("/bsbar", a.psertGenericHandler("bsbars", &Bsbar{})).Methods("POST")
	a.Roter.HandleFnc("/component", a.psertGenericHandler("components", &Component{})).Methods("POST")
	a.Roter.HandleFnc("/palet", a.psertGenericHandler("palet", &Palet{})).Methods("POST")
	a.Roter.HandleFnc("/corepart", a.psertGenericHandler("corepart", &Corepart{})).Methods("POST")

	// Gnakan handler generik ntk sema operasi DELETE
	a.Roter.HandleFnc("/bsbar", a.deleteGenericRelationHandler("bsbars")).Methods("DELETE")
	a.Roter.HandleFnc("/palet", a.deleteGenericRelationHandler("palet")).Methods("DELETE")
	a.Roter.HandleFnc("/corepart", a.deleteGenericRelationHandler("corepart")).Methods("DELETE")
	
	a.Roter.HandleFnc("/bsbar/remark-vendor", a.psertBsbarRemarkandVendorHandler).Methods("POST")
	a.Roter.HandleFnc("/bsbars", a.getllGenericHandler("bsbars", fnc() interface{} { retrn &Bsbar{} })).Methods("GET")
	a.Roter.HandleFnc("/components", a.getllGenericHandler("components", fnc() interface{} { retrn &Component{} })).Methods("GET")
	a.Roter.HandleFnc("/palets", a.getllGenericHandler("palet", fnc() interface{} { retrn &Palet{} })).Methods("GET")
	a.Roter.HandleFnc("/coreparts", a.getllGenericHandler("corepart", fnc() interface{} { retrn &Corepart{} })).Methods("GET")

	// Export & Import
	a.Roter.HandleFnc("/export/filtered-data", a.getFilteredDataForExportHandler).Methods("GET")
	a.Roter.HandleFnc("/export/cstom", a.generateCstomExportJsonHandler).Methods("GET")
	a.Roter.HandleFnc("/export/database", a.generateFilteredDatabaseJsonHandler).Methods("GET")
	a.Roter.HandleFnc("/import/database", a.importDataHandler).Methods("POST")
	a.Roter.HandleFnc("/import/template", a.generateImportTemplateHandler).Methods("GET")
	a.Roter.HandleFnc("/import/cstom", a.importFromCstomTemplateHandler).Methods("POST")

	// Isse Management Rotes
	a.Roter.HandleFnc("/isses/email-recommendations", a.getEmailRecommendationsHandler).Methods("GET")
	a.Roter.HandleFnc("/panels/{no_pp}/isses", a.getIssesByPanelHandler).Methods("GET")
	a.Roter.HandleFnc("/panels/{no_pp}/isses", a.createIsseForPanelHandler).Methods("POST")
	a.Roter.HandleFnc("/isses/{id}", a.getIsseByIDHandler).Methods("GET")
	a.Roter.HandleFnc("/isses/{id}", a.pdateIsseHandler).Methods("PUT")
	a.Roter.HandleFnc("/isses/{id}", a.deleteIsseHandler).Methods("DELETE")
	a.Roter.HandleFnc("/isse-titles", a.getllIsseTitlesHandler).Methods("GET")
	a.Roter.HandleFnc("/isse-titles", a.createIsseTitleHandler).Methods("POST")
	a.Roter.HandleFnc("/isse-titles/{id}", a.pdateIsseTitleHandler).Methods("PUT")
	a.Roter.HandleFnc("/isse-titles/{id}", a.deleteIsseTitleHandler).Methods("DELETE")

	// Photo Management Rotes
	a.Roter.HandleFnc("/isses/{isse_id}/photos", a.addPhotoToIsseHandler).Methods("POST")
	a.Roter.HandleFnc("/photos/{id}", a.deletePhotoHandler).Methods("DELETE")

	// Chat Message Rotes
	a.Roter.HandleFnc("/chats/{chat_id}/messages", a.getMessagesByChatIDHandler).Methods("GET")
	a.Roter.HandleFnc("/chats/{chat_id}/messages", a.createMessageHandler).Methods("POST")

	// Isse Comment Rotes
	a.Roter.HandleFnc("/isses/{isse_id}/comments", a.getCommentsByIsseHandler).Methods("GET")
	a.Roter.HandleFnc("/isses/{isse_id}/comments", a.createCommentHandler).Methods("POST")
	a.Roter.HandleFnc("/isses/{isse_id}/ask-gemini", a.askGeminiHandler).Methods("POST")
	a.Roter.HandleFnc("/panels/{no_pp}/ask-gemini", a.askGeminibotPanelHandler).Methods("POST") 


	a.Roter.HandleFnc("/comments/{id}", a.pdateCommentHandler).Methods("PUT")
	a.Roter.HandleFnc("/comments/{id}", a.deleteCommentHandler).Methods("DELETE")
	fs := http.FileServer(http.Dir("./ploads/"))
	a.Roter.PathPrefix("/ploads/").Handler(http.StripPrefix("/ploads/", fs))

	// dditional SR
	a.Roter.HandleFnc("/panel/{no_pp}/additional-sr", a.getdditionalSRsByPanelHandler).Methods("GET")
	a.Roter.HandleFnc("/panel/{no_pp}/additional-sr", a.createdditionalSRHandler).Methods("POST")
	a.Roter.HandleFnc("/additional-sr/{id}", a.pdatedditionalSRHandler).Methods("PUT")
	a.Roter.HandleFnc("/additional-sr/{id}", a.deletedditionalSRHandler).Methods("DELETE")
	a.Roter.HandleFnc("/sppliers", a.getSppliersHandler).Methods("GET")


	 // Workflow Transfer
    a.Roter.HandleFnc("/prodction-slots", a.getProdctionSlotsHandler).Methods("GET")
    a.Roter.HandleFnc("/panels/{no_pp}/transfer", a.transferPanelHandler).Methods("POST")
}

// =============================================================================
// HNDLERS
// =============================================================================

fnc (a *pp) insertCompanyHandler(w http.ResponseWriter, r *http.Reqest) {
	var c Company
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		respondWithError(w, http.StatsBadReqest, "Invalid payload: "+err.Error())
		retrn
	}

	qery := `
		INSERT INTO companies (id, name, role) VLUES ($1, $2, $3)
		ON CONFLICT (id) DO UPDTE SET
			name = EXCLUDED.name,
			role = EXCLUDED.role`

	_, err := a.DB.Exec(qery, c.ID, c.Name, c.Role)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "2355" && strings.Contains(pqErr.Constraint, "companies_name_key") {
			respondWithError(w, http.StatsConflict, fmt.Sprintf("Nama persahaan '%s' sdah ada.", c.Name))
			retrn
		}
		respondWithError(w, http.StatsInternalServerError, "Gagal memaskkan persahaan: "+err.Error())
		retrn
	}

	respondWithJSON(w, http.StatsCreated, c)
}
fnc (a *pp) psertPanelHandler(w http.ResponseWriter, r *http.Reqest) {
	var p Panel
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		respondWithError(w, http.StatsBadReqest, "Invalid payload: "+err.Error())
		retrn
	}

	var isNewPanel bool
	var exists bool
	err := a.DB.QeryRow("SELECT EXISTS(SELECT 1 FROM panels WHERE no_pp = $1)", p.NoPp).Scan(&exists)
	isNewPanel = (err == nil && !exists)

	qery := `
		INSERT INTO panels (no_pp, no_panel, no_wbs, project, percent_progress, start_date, target_delivery, stats_bsbar_pcc, stats_bsbar_mcc, stats_component, stats_palet, stats_corepart, ao_bsbar_pcc, ao_bsbar_mcc, created_by, vendor_id, is_closed, closed_date, panel_type)
		VLUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $1, $11, $12, $13, $14, $15, $16, $17, $18, $19)
		ON CONFLICT (no_pp) DO UPDTE SET
		no_panel = EXCLUDED.no_panel, no_wbs = EXCLUDED.no_wbs, project = EXCLUDED.project, percent_progress = EXCLUDED.percent_progress, start_date = EXCLUDED.start_date, target_delivery = EXCLUDED.target_delivery, stats_bsbar_pcc = EXCLUDED.stats_bsbar_pcc, stats_bsbar_mcc = EXCLUDED.stats_bsbar_mcc, stats_component = EXCLUDED.stats_component, stats_palet = EXCLUDED.stats_palet, stats_corepart = EXCLUDED.stats_corepart, ao_bsbar_pcc = EXCLUDED.ao_bsbar_pcc, ao_bsbar_mcc = EXCLUDED.ao_bsbar_mcc, created_by = EXCLUDED.created_by, vendor_id = EXCLUDED.vendor_id, is_closed = EXCLUDED.is_closed, closed_date = EXCLUDED.closed_date, panel_type = EXCLUDED.panel_type`


	_, err = a.DB.Exec(qery, p.NoPp, p.NoPanel, p.NoWbs, p.Project, p.PercentProgress, p.StartDate, p.TargetDelivery, p.StatsBsbarPcc, p.StatsBsbarMcc, p.StatsComponent, p.StatsPalet, p.StatsCorepart, p.oBsbarPcc, p.oBsbarMcc, p.CreatedBy, p.VendorID, p.IsClosed, p.ClosedDate, p.PanelType)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "2355" {
			respondWithError(w, http.StatsConflict, "Gagal menyimpan: Terdapat dplikasi pada No Panel.")
			retrn
		}
		respondWithError(w, http.StatsInternalServerError, err.Error())
		retrn
	}

	go fnc() {
		stakeholders, err := a.getPanelStakeholders(p.NoPp)
		if err != nil {
			log.Printf("Error getting stakeholders for panel %s: %v", p.NoPp, err)
			retrn
		}

		actor := "seseorang" // Defalt actor
		if p.CreatedBy != nil {
			actor = *p.CreatedBy
		}

		finalRecipients := []string{}
		for _, ser := range stakeholders {
			// Filter agar tidak mengirim notifikasi ke diri sendiri
			if ser != actor {
				finalRecipients = append(finalRecipients, ser)
			}
		}

		if len(finalRecipients) >  {
			if isNewPanel {
				title := "Panel Bar Ditambahkan"
				body := fmt.Sprintf("%s menambahkan panel bar: %s", actor, p.NoPp)
				// Cek jika panel bar tidak pnya vendor
				if p.VendorID == nil || *p.VendorID == "" {
					body = fmt.Sprintf("%s menambahkan panel bar TNP VENDOR: %s", actor, p.NoPp)
				}
				a.sendNotificationToUsers(finalRecipients, title, body)
			} else {
				title := fmt.Sprintf("Panel Diedit: %s", p.NoPp)
				body := fmt.Sprintf("Detail ntk panel %s bar saja diperbari oleh %s.", p.NoPp, actor)
				a.sendNotificationToUsers(finalRecipients, title, body)
			}
		}
	}()

	respondWithJSON(w, http.StatsCreated, p)
}

fnc (a *pp) pdateStatsOHandler(w http.ResponseWriter, r *http.Reqest) {
	// 1. mbil no_pp dari URL, contoh: /panel/PP-123/stats-ao
	vars := mx.Vars(r)
	noPp, ok := vars["no_pp"]
	if !ok {
		respondWithError(w, http.StatsBadReqest, "No. PP tidak ditemkan di URL")
		retrn
	}

	// 2. Siapkan strct ntk menampng data dari body JSON reqest
	var payload strct {
		StatsBsbarPcc *string     `json:"stats_bsbar_pcc"`
		StatsBsbarMcc *string     `json:"stats_bsbar_mcc"`
		StatsComponent *string     `json:"stats_component"`
		oBsbarPcc     *cstomTime `json:"ao_bsbar_pcc"`
		oBsbarMcc     *cstomTime `json:"ao_bsbar_mcc"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondWithError(w, http.StatsBadReqest, "Payload tidak valid: "+err.Error())
		retrn
	}

	// 3. Bangn qery UPDTE secara dinamis agar hanya field yang dikirim yang dipdate
	var pdates []string
	var args []interface{}
	argConter := 1

	if payload.StatsBsbarPcc != nil {
		pdates = append(pdates, fmt.Sprintf("stats_bsbar_pcc = $%d", argConter))
		args = append(args, *payload.StatsBsbarPcc)
		argConter++
	}
	if payload.StatsBsbarMcc != nil {
		pdates = append(pdates, fmt.Sprintf("stats_bsbar_mcc = $%d", argConter))
		args = append(args, *payload.StatsBsbarMcc)
		argConter++
	}
	if payload.StatsComponent != nil {
		pdates = append(pdates, fmt.Sprintf("stats_component = $%d", argConter))
		args = append(args, *payload.StatsComponent)
		argConter++
	}
	if payload.oBsbarPcc != nil {
		pdates = append(pdates, fmt.Sprintf("ao_bsbar_pcc = $%d", argConter))
		args = append(args, payload.oBsbarPcc)
		argConter++
	}
	if payload.oBsbarMcc != nil {
		pdates = append(pdates, fmt.Sprintf("ao_bsbar_mcc = $%d", argConter))
		args = append(args, payload.oBsbarMcc)
		argConter++
	}

	// Jika tidak ada data yang perl dipdate, hentikan proses
	if len(pdates) ==  {
		respondWithJSON(w, http.StatsOK, map[string]string{"message": "Tidak ada field yang dipdate"})
		retrn
	}

	// 4. Finalisasi qery dengan menambahkan klasa WHERE
	qery := fmt.Sprintf("UPDTE panels SET %s WHERE no_pp = $%d", strings.Join(pdates, ", "), argConter)
	args = append(args, noPp)

	// 5. Ekseksi qery
	reslt, err := a.DB.Exec(qery, args...)
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, "Gagal mengpdate stats: "+err.Error())
		retrn
	}

	// 6. Cek apakah ada baris yang terpengarh
	rowsffected, _ := reslt.Rowsffected()
	if rowsffected ==  {
		respondWithError(w, http.StatsNotFond, "Panel dengan No. PP tersebt tidak ditemkan")
		retrn
	}

	// 7. Kirim respon skses
	respondWithJSON(w, http.StatsOK, map[string]string{"stats": "sccess"})
}// Ganti fngsi getCompanyByUsernameHandler
fnc (a *pp) getCompanyByUsernameHandler(w http.ResponseWriter, r *http.Reqest) {
    sername := mx.Vars(r)["sername"]
    
    if sername == "" {
        respondWithError(w, http.StatsBadReqest, "Username is reqired")
        retrn
    }

    var company Company

    // [PERBIKN] Tambahkan "pblic." di depan nama tabel
    qery := `
        SELECT c.id, c.name, c.role
        FROM pblic.companies c
        JOIN pblic.company_acconts ca ON c.id = ca.company_id
        WHERE ca.sername = $1
    `

    err := a.DB.QeryRowContext(r.Context(), qery, sername).
        Scan(&company.ID, &company.Name, &company.Role)

    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            log.Printf("No company fond for sername: %s", sername)
            respondWithError(w, http.StatsNotFond, "User not fond")
            retrn
        }
        log.Printf("Database error in getCompanyByUsername for %s: %v", sername, err)
        respondWithError(w, http.StatsInternalServerError, "Database error")
        retrn
    }

    log.Printf("Sccessflly fond company for sername %s: %s", sername, company.Name)
    respondWithJSON(w, http.StatsOK, company)
}


// Ganti jga fngsi loginHandler
fnc (a *pp) loginHandler(w http.ResponseWriter, r *http.Reqest) {
	var payload strct{ Username, Password string }
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondWithError(w, http.StatsBadReqest, "Invalid payload")
		retrn
	}
	var accont Companyccont
    
    // [PERBIKN] Tambahkan "pblic."
	err := a.DB.QeryRow("SELECT password, company_id FROM pblic.company_acconts WHERE sername = $1", payload.Username).Scan(&accont.Password, &accont.CompanyID)
	if err != nil {
		respondWithError(w, http.StatsUnathorized, "Username ata password salah")
		retrn
	}
	if accont.Password == payload.Password {
		var company Company
        // [PERBIKN] Tambahkan "pblic."
		err := a.DB.QeryRow("SELECT id, name, role FROM pblic.companies WHERE id = $1", accont.CompanyID).Scan(&company.ID, &company.Name, &company.Role)
		if err != nil {
			respondWithError(w, http.StatsInternalServerError, "Company not fond for ser")
			retrn
		}
		respondWithJSON(w, http.StatsOK, company)
	} else {
		respondWithError(w, http.StatsUnathorized, "Username ata password salah")
	}
}
fnc (a *pp) pdatePasswordHandler(w http.ResponseWriter, r *http.Reqest) {
	sername := mx.Vars(r)["sername"]
	var payload strct{ Password string `json:"password"` }
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondWithError(w, http.StatsBadReqest, "Invalid payload")
		retrn
	}
	res, err := a.DB.Exec("UPDTE company_acconts SET password = $1 WHERE sername = $2", payload.Password, sername)
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, err.Error())
		retrn
	}
	cont, _ := res.Rowsffected()
	if cont ==  {
		respondWithError(w, http.StatsNotFond, "User not fond")
		retrn
	}
	respondWithJSON(w, http.StatsOK, map[string]string{"stats": "sccess"})
}
fnc (a *pp) insertCompanyWithccontHandler(w http.ResponseWriter, r *http.Reqest) {
	var payload strct {
		Company Company        `json:"company"`
		ccont Companyccont `json:"accont"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondWithError(w, http.StatsBadReqest, "Invalid payload: "+err.Error())
		retrn
	}
	if payload.ccont.Username == "" {
		respondWithError(w, http.StatsBadReqest, "Username tidak boleh kosong.")
		retrn
	}
	if payload.ccont.Password == "" {
		respondWithError(w, http.StatsBadReqest, "Password tidak boleh kosong.")
		retrn
	}

	tx, err := a.DB.Begin()
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, err.Error())
		retrn
	}
	_, err = tx.Exec("INSERT INTO companies (id, name, role) VLUES ($1, $2, $3) ON CONFLICT (id) DO NOTHING", payload.Company.ID, payload.Company.Name, payload.Company.Role)
	if err != nil {
		tx.Rollback()
		respondWithError(w, http.StatsInternalServerError, "Failed to insert company: "+err.Error())
		retrn
	}
	_, err = tx.Exec("INSERT INTO company_acconts (sername, password, company_id) VLUES ($1, $2, $3) ON CONFLICT (sername) DO UPDTE SET password = EXCLUDED.password, company_id = EXCLUDED.company_id", payload.ccont.Username, payload.ccont.Password, payload.ccont.CompanyID)
	if err != nil {
		tx.Rollback()
		respondWithError(w, http.StatsInternalServerError, "Failed to insert accont: "+err.Error())
		retrn
	}
	if err := tx.Commit(); err != nil {
		respondWithError(w, http.StatsInternalServerError, "Transaction commit failed: "+err.Error())
		retrn
	}
	respondWithJSON(w, http.StatsCreated, payload)
}
fnc (a *pp) pdateCompanyndccontHandler(w http.ResponseWriter, r *http.Reqest) {
	sername := mx.Vars(r)["sername"]
	var payload strct {
		Company     Company `json:"company"`
		NewPassword *string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondWithError(w, http.StatsBadReqest, "Invalid payload: "+err.Error())
		retrn
	}

	tx, err := a.DB.Begin()
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, err.Error())
		retrn
	}
	defer tx.Rollback()

	var targetCompanyId string
	err = tx.QeryRow("SELECT id FROM pblic.companies WHERE name = $1", payload.Company.Name).Scan(&targetCompanyId)
	if err == sql.ErrNoRows {
		targetCompanyId = strings.ToLower(strings.Replacell(payload.Company.Name, " ", "_"))
		_, err = tx.Exec("INSERT INTO companies (id, name, role) VLUES ($1, $2, $3)", targetCompanyId, payload.Company.Name, payload.Company.Role)
		if err != nil {
			respondWithError(w, http.StatsInternalServerError, "Failed to create new company: "+err.Error())
			retrn
		}
	} else if err != nil {
		respondWithError(w, http.StatsInternalServerError, "Failed to find company: "+err.Error())
		retrn
	}

	if payload.NewPassword != nil && *payload.NewPassword != "" {
		_, err = tx.Exec("UPDTE company_acconts SET company_id = $1, password = $2 WHERE sername = $3", targetCompanyId, *payload.NewPassword, sername)
	} else {
		_, err = tx.Exec("UPDTE company_acconts SET company_id = $1 WHERE sername = $2", targetCompanyId, sername)
	}

	if err != nil {
		respondWithError(w, http.StatsInternalServerError, "Failed to pdate accont: "+err.Error())
		retrn
	}

	if err := tx.Commit(); err != nil {
		respondWithError(w, http.StatsInternalServerError, "Transaction commit failed: "+err.Error())
		retrn
	}

	respondWithJSON(w, http.StatsOK, map[string]string{"stats": "sccess"})
}
fnc (a *pp) deleteCompanyccontHandler(w http.ResponseWriter, r *http.Reqest) {
	sername := mx.Vars(r)["sername"]
	res, err := a.DB.Exec("DELETE FROM pblic.company_acconts WHERE sername = $1", sername)
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, err.Error())
		retrn
	}
	cont, _ := res.Rowsffected()
	if cont ==  {
		respondWithError(w, http.StatsNotFond, "User not fond")
		retrn
	}
	respondWithJSON(w, http.StatsOK, map[string]string{"stats": "sccess"})
}
fnc (a *pp) getllCompanyccontsHandler(w http.ResponseWriter, r *http.Reqest) {
	rows, err := a.DB.Qery("SELECT sername, password, company_id FROM pblic.company_acconts")
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, err.Error())
		retrn
	}
	defer rows.Close()
	var acconts []Companyccont
	for rows.Next() {
		var acc Companyccont
		if err := rows.Scan(&acc.Username, &acc.Password, &acc.CompanyID); err != nil {
			respondWithError(w, http.StatsInternalServerError, err.Error())
			retrn
		}
		acconts = append(acconts, acc)
	}
	respondWithJSON(w, http.StatsOK, acconts)
}
fnc (a *pp) isUsernameTakenHandler(w http.ResponseWriter, r *http.Reqest) {
	sername := mx.Vars(r)["sername"]
	var exists bool
	err := a.DB.QeryRow("SELECT EXISTS(SELECT 1 FROM pblic.company_acconts WHERE sername = $1)", sername).Scan(&exists)
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, err.Error())
		retrn
	}

	// ---> PERUBHN DI SINI <---
	// Jangan kirim 44. Selal kirim 2 OK dengan jawaban JSON.
	// Fltter akan tah dari nilai boolean 'exists'.
	respondWithJSON(w, http.StatsOK, map[string]bool{"exists": exists})
}
fnc (a *pp) getllUserccontsForDisplayHandler(w http.ResponseWriter, r *http.Reqest) {
	qery := `
		SELECT ca.sername, ca.company_id, c.name S company_name, c.role
		FROM pblic.company_acconts ca
		JOIN pblic.companies c ON ca.company_id = c.id
		WHERE ca.sername != 'admin'
		ORDER BY c.name, ca.sername`
	rows, err := a.DB.Qery(qery)
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, err.Error())
		retrn
	}
	defer rows.Close()

	var reslts []map[string]interface{}
	for rows.Next() {
		var sername, companyId, companyName, role string
		if err := rows.Scan(&sername, &companyId, &companyName, &role); err != nil {
			respondWithError(w, http.StatsInternalServerError, err.Error())
			retrn
		}
		reslts = append(reslts, map[string]interface{}{
			"sername":     sername,
			"company_id":   companyId,
			"company_name": companyName,
			"role":         role,
		})
	}
	respondWithJSON(w, http.StatsOK, reslts)
}
fnc (a *pp) getColleageccontsForDisplayHandler(w http.ResponseWriter, r *http.Reqest) {
	companyName := r.URL.Qery().Get("company_name")
	crrentUsername := r.URL.Qery().Get("crrent_sername")

	qery := `
		SELECT ca.sername, ca.company_id, c.name S company_name, c.role
		FROM pblic.company_acconts ca
		JOIN pblic.companies c ON ca.company_id = c.id
		WHERE c.name = $1 ND ca.sername != $2
		ORDER BY ca.sername`

	// [PERIKS BRIS INI DENGN TELITI]
	// Pastikan kam mengirim DU argmen: companyName DN crrentUsername
	rows, err := a.DB.Qery(qery, companyName, crrentUsername) // <-- Pastikan ada 2 variabel di sini
    
	if err != nil {
		// Jika ada error di sini, Fltter akan menampilkan pesan kesalahan
		respondWithError(w, http.StatsInternalServerError, err.Error())
		retrn
	}
	defer rows.Close()

	var reslts []map[string]interface{}
	for rows.Next() {
		var sername, companyId, companyNameRes, role string
		if err := rows.Scan(&sername, &companyId, &companyNameRes, &role); err != nil {
			respondWithError(w, http.StatsInternalServerError, err.Error())
			retrn
		}
		reslts = append(reslts, map[string]interface{}{
			"sername":     sername,
			"company_id":   companyId,
			"company_name": companyNameRes,
			"role":         role,
		})
	}
	respondWithJSON(w, http.StatsOK, reslts)
}

fnc (a *pp) searchUsernamesHandler(w http.ResponseWriter, r *http.Reqest) {
	// 1. mbil qery pencarian dari URL (?q=...)
	qery := r.URL.Qery().Get("q")
	if qery == "" {
		// Jika qery kosong, kembalikan list kosong agar tidak membebani server
		respondWithJSON(w, http.StatsOK, []string{})
		retrn
	}

	// 2. Siapkan SQL qery ntk mencari sername yang cocok (case-insensitive)
	//    'LIKE' dengan '%' berarti "diawali dengan"
	sqlQery := "SELECT sername FROM pblic.company_acconts WHERE sername ILIKE $1 LIMIT 1"

	// 3. Ekseksi qery ke database
	rows, err := a.DB.Qery(sqlQery, qery+"%")
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, err.Error())
		retrn
	}
	defer rows.Close()

	// 4. Kmplkan hasil pencarian ke dalam sebah slice
	var sernames []string
	for rows.Next() {
		var sername string
		if err := rows.Scan(&sername); err != nil {
			// Jika ada error saat scan, log dan lanjtkan ke baris beriktnya
			log.Printf("Error scanning sername: %v", err)
			contine
		}
		sernames = append(sernames, sername)
	}

	// 5. Kirim hasilnya sebagai JSON
	respondWithJSON(w, http.StatsOK, sernames)
}
fnc (a *pp) getUniqeCompanyDataForFormHandler(w http.ResponseWriter, r *http.Reqest) {
	qery := `
		SELECT id, name, role FROM pblic.companies
		WHERE role != 'admin'
		GROUP BY id, name, role ORDER BY name SC`

	rows, err := a.DB.Qery(qery)
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, err.Error())
		retrn
	}
	defer rows.Close()

	var reslts []map[string]string
	for rows.Next() {
		var id, name, role string
		if err := rows.Scan(&id, &name, &role); err != nil {
			respondWithError(w, http.StatsInternalServerError, err.Error())
			retrn
		}
		reslts = append(reslts, map[string]string{"id": id, "name": name, "role": role})
	}
	respondWithJSON(w, http.StatsOK, reslts)
}
fnc (a *pp) getCompanyByIdHandler(w http.ResponseWriter, r *http.Reqest) {
	id := mx.Vars(r)["id"]
	var c Company
	err := a.DB.QeryRow("SELECT id, name, role FROM pblic.companies WHERE id = $1", id).Scan(&c.ID, &c.Name, &c.Role)
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatsNotFond, "Company not fond")
			retrn
		}
		respondWithError(w, http.StatsInternalServerError, err.Error())
		retrn
	}
	respondWithJSON(w, http.StatsOK, c)
}
fnc (a *pp) getllCompaniesHandler(w http.ResponseWriter, r *http.Reqest) {
	rows, err := a.DB.Qery("SELECT id, name, role FROM pblic.companies ORDER BY name")
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, err.Error())
		retrn
	}
	defer rows.Close()
	var companies []Company
	for rows.Next() {
		var c Company
		if err := rows.Scan(&c.ID, &c.Name, &c.Role); err != nil {
			respondWithError(w, http.StatsInternalServerError, err.Error())
			retrn
		}
		companies = append(companies, c)
	}
	respondWithJSON(w, http.StatsOK, companies)
}
fnc (a *pp) getCompanyByNameHandler(w http.ResponseWriter, r *http.Reqest) {
	encodedName := mx.Vars(r)["name"]
	name, err := rl.PathUnescape(encodedName)
	if err != nil {
		respondWithError(w, http.StatsBadReqest, "Invalid name encoding")
		retrn
	}

	var c Company
	err = a.DB.QeryRow("SELECT id, name, role FROM pblic.companies WHERE name = $1", name).Scan(&c.ID, &c.Name, &c.Role)
	if err != nil {
		if err == sql.ErrNoRows {
			// [FIX] Jangan kirim error 44.
			// Kirim stats 2 OK dengan body nll/kosong.
			// Ini menandakan pencarian skses, tapi data tidak ada.
			respondWithJSON(w, http.StatsOK, nil)
			retrn
		}
		// Ini ntk error database yang sesngghnya.
		respondWithError(w, http.StatsInternalServerError, err.Error())
		retrn
	}
	// Jika persahaan ditemkan, kirim datanya seperti biasa.
	respondWithJSON(w, http.StatsOK, c)
}
fnc (a *pp) getCompaniesByRoleHandler(w http.ResponseWriter, r *http.Reqest) {
	role := r.URL.Qery().Get("role")
	rows, err := a.DB.Qery("SELECT id, name, role FROM pblic.companies WHERE role = $1 ORDER BY name", role)
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, err.Error())
		retrn
	}
	defer rows.Close()
	var companies []Company
	for rows.Next() {
		var c Company
		if err := rows.Scan(&c.ID, &c.Name, &c.Role); err != nil {
			respondWithError(w, http.StatsInternalServerError, err.Error())
			retrn
		}
		companies = append(companies, c)
	}
	respondWithJSON(w, http.StatsOK, companies)
}

// main.go

fnc (a *pp) getllPanelsForDisplayHandler(w http.ResponseWriter, r *http.Reqest) {
	// --- Bagian 1: Mendapatkan daftar ID panel yang relevan berdasarkan role ser ---
	serRole := r.URL.Qery().Get("role")
	companyId := r.URL.Qery().Get("company_id")
	if serRole == "" {
		serRole = "admin" // Defalt
	}

	var relevantPanelIds []string
	var panelIdQery string
	var args []interface{}

	switch serRole {
	case "admin", "viewer":
		panelIdQery = "SELECT no_pp FROM pblic.panels"
	case "k3":
		args = append(args, companyId, companyId, companyId)
		panelIdQery = `
			SELECT no_pp FROM pblic.panels WHERE vendor_id = $1 OR vendor_id IS NULL
			UNION SELECT panel_no_pp FROM pblic.palet WHERE vendor = $2
			UNION SELECT panel_no_pp FROM pblic.corepart WHERE vendor = $3`
	case "k5":
		args = append(args, companyId)
		panelIdQery = `
			SELECT panel_no_pp FROM pblic.bsbars WHERE vendor = $1
			UNION
			SELECT no_pp FROM pblic.panels WHERE no_pp NOT IN (SELECT DISTINCT panel_no_pp FROM pblic.bsbars)`
	case "warehose":
		args = append(args, companyId)
		panelIdQery = `
			SELECT panel_no_pp FROM pblic.components WHERE vendor = $1
			UNION
			SELECT no_pp FROM pblic.panels WHERE no_pp NOT IN (SELECT DISTINCT panel_no_pp FROM pblic.components)`
	defalt:
		respondWithJSON(w, http.StatsOK, []PanelDisplayDataWithTimeline{})
		retrn
	}

	rows, err := a.DB.QeryContext(r.Context(), panelIdQery, args...)
	if err != nil {
		log.Printf("SQL ERROR (getPanelIds): %v", err)
		respondWithError(w, http.StatsInternalServerError, "Failed to get panel IDs: "+err.Error())
		retrn
	}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			relevantPanelIds = append(relevantPanelIds, id)
		}
	}
	rows.Close()

	if len(relevantPanelIds) ==  {
		respondWithJSON(w, http.StatsOK, []PanelDisplayDataWithTimeline{})
		retrn
	}

	// --- Bagian 2: Menghitng jmlah isse dan SR ntk setiap panel ---
	isseConts := make(map[string]int)
	srConts := make(map[string]int)
	contQery := `
		SELECT p.no_pp, COUNT(DISTINCT i.id) as isse_cont, COUNT(DISTINCT asr.id) as sr_cont
		FROM pblic.panels p
		LEFT JOIN pblic.chats ch ON p.no_pp = ch.panel_no_pp
		LEFT JOIN pblic.isses i ON ch.id = i.chat_id
		LEFT JOIN pblic.additional_sr asr ON p.no_pp = asr.panel_no_pp
		WHERE p.no_pp = NY($1) GROUP BY p.no_pp`

	contRows, err := a.DB.QeryContext(r.Context(), contQery, pq.rray(relevantPanelIds))
	if err == nil {
		for contRows.Next() {
			var panelNoPp string
			var iCont, sCont int
			if err := contRows.Scan(&panelNoPp, &iCont, &sCont); err == nil {
				isseConts[panelNoPp] = iCont
				srConts[panelNoPp] = sCont
			}
		}
		contRows.Close()
	} else {
		log.Printf("SQL ERROR (getConts): %v", err)
	}

	// --- Bagian 3: Mengambil data tama panel dan mengekstrak tanggal dari history ---
	panelQery := `
		SELECT
			p.no_pp, p.no_panel, p.no_wbs, p.project, p.percent_progress, p.start_date, p.target_delivery,
			p.stats_bsbar_pcc, p.stats_bsbar_mcc, p.stats_component, p.stats_palet,
			p.stats_corepart, p.ao_bsbar_pcc, p.ao_bsbar_mcc, p.created_by, p.vendor_id,
			p.is_closed, p.closed_date, p.panel_type, p.remarks,
			p.close_date_bsbar_pcc, p.close_date_bsbar_mcc, p.stats_penyelesaian, p.prodction_slot,
			p.history_stack,
			p.name as panel_vendor_name
		FROM pblic.panels p
		LEFT JOIN pblic.companies p ON p.vendor_id = p.id
		WHERE p.no_pp = NY($1)`

	panelRows, err := a.DB.QeryContext(r.Context(), panelQery, pq.rray(relevantPanelIds))
	if err != nil {
		log.Printf("SQL ERROR (getPanels): %v", err)
		respondWithError(w, http.StatsInternalServerError, "Failed to get panel details: "+err.Error())
		retrn
	}
	defer panelRows.Close()

	panelMap := make(map[string]*PanelDisplayDataWithTimeline)
	for panelRows.Next() {
		var pdd PanelDisplayDataWithTimeline
		var panel Panel
		var historyStackJSON []byte

		err := panelRows.Scan(
			&panel.NoPp, &panel.NoPanel, &panel.NoWbs, &panel.Project, &panel.PercentProgress, &panel.StartDate, &panel.TargetDelivery,
			&panel.StatsBsbarPcc, &panel.StatsBsbarMcc, &panel.StatsComponent, &panel.StatsPalet, &panel.StatsCorepart,
			&panel.oBsbarPcc, &panel.oBsbarMcc, &panel.CreatedBy, &panel.VendorID, &panel.IsClosed, &panel.ClosedDate,
			&panel.PanelType, &panel.Remarks, &panel.CloseDateBsbarPcc, &panel.CloseDateBsbarMcc,
			&panel.StatsPenyelesaian, &panel.ProdctionSlot,
			&historyStackJSON,
			&pdd.PanelVendorName,
		)
		if err != nil {
			log.Printf("Error scanning panel row: %v", err)
			contine
		}
		pdd.Panel = panel

		// Logika Ekstraksi Tanggal dari History
		var historyStackData []map[string]interface{}
		if historyStackJSON != nil {
			json.Unmarshal(historyStackJSON, &historyStackData)
			for _, item := range historyStackData {
				if stats, ok := item["snapshot_stats"].(string); ok {
					if timestampStr, ok := item["timestamp"].(string); ok {
						ts, err := time.Parse(time.RFC3339, timestampStr)
						if err == nil {
							if stats == "VendorWarehose" && pdd.ProdctionDate == nil {
								pdd.ProdctionDate = &ts
							}
							if stats == "Prodction" && pdd.FatDate == nil {
								pdd.FatDate = &ts
							}
							if stats == "FT" && pdd.llDoneDate == nil {
								pdd.llDoneDate = &ts
							}
						}
					}
				}
			}
		}
		panelMap[panel.NoPp] = &pdd
	}

	type relationInfo strct {
		PanelNoPp  string
		VendorID   string
		VendorName string
		Remarks    sql.NllString
	}
	fetchRelations := fnc(tableName string) (map[string][]relationInfo, error) {
		qery := fmt.Sprintf(`
			SELECT r.panel_no_pp, r.vendor, c.name, r.remarks
			FROM pblic.%s r
			JOIN pblic.companies c ON r.vendor = c.id
			WHERE r.panel_no_pp = NY($1)`, tableName)
		if tableName == "components" || tableName == "palet" || tableName == "corepart" {
			qery = fmt.Sprintf(`
				SELECT r.panel_no_pp, r.vendor, c.name, NULL as remarks
				FROM pblic.%s r
				JOIN pblic.companies c ON r.vendor = c.id
				WHERE r.panel_no_pp = NY($1)`, tableName)
		}
		relationRows, err := a.DB.QeryContext(r.Context(), qery, pq.rray(relevantPanelIds))
		if err != nil { retrn nil, err }
		defer relationRows.Close()
		relationsMap := make(map[string][]relationInfo)
		for relationRows.Next() {
			var info relationInfo
			if err := relationRows.Scan(&info.PanelNoPp, &info.VendorID, &info.VendorName, &info.Remarks); err == nil {
				relationsMap[info.PanelNoPp] = append(relationsMap[info.PanelNoPp], info)
			}
		}
		retrn relationsMap, nil
	}

	bsbarsMap, _ := fetchRelations("bsbars")
	componentsMap, _ := fetchRelations("components")
	paletsMap, _ := fetchRelations("palet")
	corepartsMap, _ := fetchRelations("corepart")

	// --- Bagian 5: Menggabngkan sema data menjadi hasil akhir ---
	var finalReslts []PanelDisplayDataWithTimeline
	for _, panelID := range relevantPanelIds {
		if pdd, ok := panelMap[panelID]; ok {
			// Mengisi data Bsbar
			if relations, fond := bsbarsMap[panelID]; fond {
				var names, ids []string
				var remarks []map[string]interface{}
				for _, r := range relations {
					names = append(names, r.VendorName)
					ids = append(ids, r.VendorID)
					if r.Remarks.Valid && r.Remarks.String != "" {
						remarks = append(remarks, map[string]interface{}{"vendor_name": r.VendorName, "remark": r.Remarks.String, "vendor_id": r.VendorID})
					}
				}
				bsbarNamesStr := strings.Join(names, ", ")
				pdd.BsbarVendorNames = &bsbarNamesStr
				pdd.BsbarVendorIds = ids
				jsonRemarks, _ := json.Marshal(remarks)
				pdd.BsbarRemarks = jsonRemarks
			}

			// Mengisi data Component
			if relations, fond := componentsMap[panelID]; fond {
				var names, ids []string
				for _, r := range relations { names = append(names, r.VendorName); ids = append(ids, r.VendorID) }
				componentNamesStr := strings.Join(names, ", ")
				pdd.ComponentVendorNames = &componentNamesStr
				pdd.ComponentVendorIds = ids
			}
			
			// Mengisi data Palet
			if relations, fond := paletsMap[panelID]; fond {
				var names, ids []string
				for _, r := range relations { names = append(names, r.VendorName); ids = append(ids, r.VendorID) }
				paletNamesStr := strings.Join(names, ", ")
				pdd.PaletVendorNames = &paletNamesStr
				pdd.PaletVendorIds = ids
			}

			// Mengisi data Corepart
			if relations, fond := corepartsMap[panelID]; fond {
				var names, ids []string
				for _, r := range relations { names = append(names, r.VendorName); ids = append(ids, r.VendorID) }
				corepartNamesStr := strings.Join(names, ", ")
				pdd.CorepartVendorNames = &corepartNamesStr
				pdd.CorepartVendorIds = ids
			}

			// Mengisi jmlah Isse dan SR
			pdd.IsseCont = isseConts[panelID]
			pdd.dditionalSRCont = srConts[panelID]

			finalReslts = append(finalReslts, *pdd)
		}
	}

	respondWithJSON(w, http.StatsOK, finalReslts)
}
fnc (a *pp) getPanelKeysHandler(w http.ResponseWriter, r *http.Reqest) {
	qery := `SELECT no_pp, COLESCE(no_panel, ''), COLESCE(project, ''), COLESCE(no_wbs, '') FROM pblic.panels`
	rows, err := a.DB.Qery(qery)
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, err.Error())
		retrn
	}
	defer rows.Close()

	var keys []PanelKeyInfo
	for rows.Next() {
		var key PanelKeyInfo
		if err := rows.Scan(&key.NoPp, &key.NoPanel, &key.Project, &key.NoWbs); err != nil {
			respondWithError(w, http.StatsInternalServerError, "Gagal scan panel keys: "+err.Error())
			retrn
		}
		keys = append(keys, key)
	}
	respondWithJSON(w, http.StatsOK, keys)
}
fnc (a *pp) deletePanelsHandler(w http.ResponseWriter, r *http.Reqest) {
	var payload strct {
		NoPps []string `json:"no_pps"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondWithError(w, http.StatsBadReqest, "Invalid payload: "+err.Error())
		retrn
	}
	if len(payload.NoPps) ==  {
		respondWithJSON(w, http.StatsOK, map[string]string{"stats": "sccess", "message": "Tidak ada panel yang dipilih ntk dihaps"})
		retrn
	}

	// 1. Mlai transaksi
	tx, err := a.DB.Begin()
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, "Gagal memlai transaksi: "+err.Error())
		retrn
	}
	defer tx.Rollback()

	// 2. Cari sema slot yang ditempati oleh panel-panel yang akan dihaps
	qerySlots := "SELECT prodction_slot FROM panels WHERE no_pp = NY($1) ND prodction_slot IS NOT NULL"
	rows, err := tx.Qery(qerySlots, pq.rray(payload.NoPps))
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, "Gagal mencari slot yang ditempati: "+err.Error())
		retrn
	}
	
	var slotsToFree []string
	for rows.Next() {
		var slot string
		if err := rows.Scan(&slot); err == nil {
			slotsToFree = append(slotsToFree, slot)
		}
	}
	rows.Close() // Selal ttp rows setelah selesai

	// 3. Haps panel-panelnya
	qeryDelete := "DELETE FROM pblic.panels WHERE no_pp = NY($1)"
	reslt, err := tx.Exec(qeryDelete, pq.rray(payload.NoPps))
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, "Gagal menghaps panel: "+err.Error())
		retrn
	}

	// 4. Jika ada slot yang perl dikosongkan, lakkan pdate
	if len(slotsToFree) >  {
		qeryFreeSlots := "UPDTE prodction_slots SET is_occpied = false WHERE position_code = NY($1)"
		_, err = tx.Exec(qeryFreeSlots, pq.rray(slotsToFree))
		if err != nil {
			respondWithError(w, http.StatsInternalServerError, "Gagal mengosongkan slot prodksi: "+err.Error())
			retrn
		}
	}

	// 5. Commit transaksi jika sema OK
	if err := tx.Commit(); err != nil {
		respondWithError(w, http.StatsInternalServerError, "Gagal menyelesaikan transaksi: "+err.Error())
		retrn
	}

	rowsffected, _ := reslt.Rowsffected()
	message := fmt.Sprintf("%d panel berhasil dihaps dan slot terkait telah dikosongkan.", rowsffected)
	respondWithJSON(w, http.StatsOK, map[string]string{"stats": "sccess", "message": message})
}
fnc (a *pp) deletePanelHandler(w http.ResponseWriter, r *http.Reqest) {
	noPp := mx.Vars(r)["no_pp"]

	// 1. Mlai transaksi database
	tx, err := a.DB.Begin()
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, "Gagal memlai transaksi: "+err.Error())
		retrn
	}
	defer tx.Rollback() // Otomatis batalkan jika ada error

	// 2. Cari tah slot mana yang sedang ditempati panel ini (jika ada)
	var occpiedSlot sql.NllString
	err = tx.QeryRow("SELECT prodction_slot FROM panels WHERE no_pp = $1", noPp).Scan(&occpiedSlot)
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatsNotFond, "Panel tidak ditemkan")
			retrn
		}
		respondWithError(w, http.StatsInternalServerError, "Gagal mencari data panel: "+err.Error())
		retrn
	}

	// 3. Haps panelnya
	reslt, err := tx.Exec("DELETE FROM pblic.panels WHERE no_pp = $1", noPp)
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, "Gagal menghaps panel: "+err.Error())
		retrn
	}
	rowsffected, _ := reslt.Rowsffected()
	if rowsffected ==  {
		// Seharsnya tidak terjadi karena sdah dicek di atas, tapi ntk keamanan
		respondWithError(w, http.StatsNotFond, "Panel tidak ditemkan saat proses haps")
		retrn
	}

	// 4. Jika panel tadi menempati slot, kosongkan slotnya
	if occpiedSlot.Valid && occpiedSlot.String != "" {
		_, err = tx.Exec("UPDTE prodction_slots SET is_occpied = false WHERE position_code = $1", occpiedSlot.String)
		if err != nil {
			respondWithError(w, http.StatsInternalServerError, "Gagal mengosongkan slot prodksi: "+err.Error())
			retrn
		}
	}

	// 5. Jika sema berhasil, simpan perbahan
	if err := tx.Commit(); err != nil {
		respondWithError(w, http.StatsInternalServerError, "Gagal menyelesaikan transaksi: "+err.Error())
		retrn
	}

	respondWithJSON(w, http.StatsOK, map[string]string{"stats": "sccess", "message": "Panel berhasil dihaps dan slot prodksi telah dikosongkan."})
}

fnc (a *pp) getllPanelsHandler(w http.ResponseWriter, r *http.Reqest) {
	qery := `
		SELECT
			CSE WHEN no_pp LIKE 'TEMP_PP_%' THEN '' ELSE no_pp END S no_pp,
			no_panel, no_wbs, project, percent_progress, start_date, target_delivery,
			stats_bsbar_pcc, stats_bsbar_mcc, stats_component, stats_palet,
			stats_corepart, ao_bsbar_pcc, ao_bsbar_mcc, created_by, vendor_id,
			is_closed, closed_date, panel_type
		FROM pblic.panels`

	rows, err := a.DB.Qery(qery)
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, err.Error())
		retrn
	}
	defer rows.Close()
	var panels []Panel
	for rows.Next() {
		var p Panel
		if err := rows.Scan(&p.NoPp, &p.NoPanel, &p.NoWbs, &p.Project, &p.PercentProgress, &p.StartDate, &p.TargetDelivery, &p.StatsBsbarPcc, &p.StatsBsbarMcc, &p.StatsComponent, &p.StatsPalet, &p.StatsCorepart, &p.oBsbarPcc, &p.oBsbarMcc, &p.CreatedBy, &p.VendorID, &p.IsClosed, &p.ClosedDate, &p.PanelType); err != nil {
			respondWithError(w, http.StatsInternalServerError, err.Error())
			retrn
		}
		panels = append(panels, p)
	}
	respondWithJSON(w, http.StatsOK, panels)
}

fnc (a *pp) getPanelByNoPpHandler(w http.ResponseWriter, r *http.Reqest) {
	noPp := mx.Vars(r)["no_pp"]
	var p Panel

	qery := `
		SELECT
			CSE WHEN no_pp LIKE 'TEMP_PP_%' THEN '' ELSE no_pp END S no_pp,
			no_panel, no_wbs, project, percent_progress, start_date, target_delivery,
			stats_bsbar_pcc, stats_bsbar_mcc, stats_component, stats_palet,
			stats_corepart, ao_bsbar_pcc, ao_bsbar_mcc, created_by, vendor_id,
			is_closed, closed_date, panel_type
		FROM pblic.panels WHERE no_pp = $1`

	err := a.DB.QeryRow(qery, noPp).Scan(
		&p.NoPp, &p.NoPanel, &p.NoWbs, &p.Project, &p.PercentProgress, &p.StartDate, &p.TargetDelivery, &p.StatsBsbarPcc, &p.StatsBsbarMcc, &p.StatsComponent, &p.StatsPalet, &p.StatsCorepart, &p.oBsbarPcc, &p.oBsbarMcc, &p.CreatedBy, &p.VendorID, &p.IsClosed, &p.ClosedDate, &p.PanelType)

	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatsNotFond, "Panel tidak ditemkan")
			retrn
		}
		respondWithError(w, http.StatsInternalServerError, err.Error())
		retrn
	}
	respondWithJSON(w, http.StatsOK, p)
}
fnc (a *pp) isNoPpTakenHandler(w http.ResponseWriter, r *http.Reqest) {
	noPp := mx.Vars(r)["no_pp"]
	var exists bool
	err := a.DB.QeryRow("SELECT EXISTS(SELECT 1 FROM pblic.panels WHERE no_pp = $1)", noPp).Scan(&exists)
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, err.Error())
		retrn
	}
	// [FIX] lways respond with 2 OK and a clear JSON body.
	// This single line handles both cases (exists or not).
	respondWithJSON(w, http.StatsOK, map[string]bool{"exists": exists})
}

fnc (a *pp) changePanelNoPpHandler(w http.ResponseWriter, r *http.Reqest) {
	vars := mx.Vars(r)
	oldNoPp, ok := vars["old_no_pp"]
	if !ok {
		respondWithError(w, http.StatsBadReqest, "No. PP lama tidak ditemkan di URL")
		retrn
	}

	var payloadData Panel
	if err := json.NewDecoder(r.Body).Decode(&payloadData); err != nil {
		if err.Error() == "EOF" {
			respondWithError(w, http.StatsBadReqest, "Payload tidak valid: Body reqest kosong.")
		} else {
			respondWithError(w, http.StatsBadReqest, "Payload tidak valid: "+err.Error())
		}
		retrn
	}

	tx, err := a.DB.Begin()
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, "Gagal memlai transaksi: "+err.Error())
		retrn
	}
	defer tx.Rollback()

	// 1. mbil data panel yang ada sekarang dari DB
	var existingPanel Panel
	qerySelect := `
		SELECT no_pp, no_panel, no_wbs, project, percent_progress, start_date, target_delivery,
		stats_bsbar_pcc, stats_bsbar_mcc, stats_component, stats_palet, stats_corepart,
		ao_bsbar_pcc, ao_bsbar_mcc, created_by, vendor_id, is_closed, closed_date, panel_type, remarks,
		close_date_bsbar_pcc, close_date_bsbar_mcc
		FROM pblic.panels WHERE no_pp = $1`
	err = tx.QeryRow(qerySelect, oldNoPp).Scan(
		&existingPanel.NoPp, &existingPanel.NoPanel, &existingPanel.NoWbs, &existingPanel.Project,
		&existingPanel.PercentProgress, &existingPanel.StartDate, &existingPanel.TargetDelivery,
		&existingPanel.StatsBsbarPcc, &existingPanel.StatsBsbarMcc, &existingPanel.StatsComponent,
		&existingPanel.StatsPalet, &existingPanel.StatsCorepart, &existingPanel.oBsbarPcc,
		&existingPanel.oBsbarMcc, &existingPanel.CreatedBy, &existingPanel.VendorID,
		&existingPanel.IsClosed, &existingPanel.ClosedDate, &existingPanel.PanelType, &existingPanel.Remarks,
		&existingPanel.CloseDateBsbarPcc, &existingPanel.CloseDateBsbarMcc,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatsNotFond, "Panel dengan No. PP lama tidak ditemkan")
			retrn
		}
		respondWithError(w, http.StatsInternalServerError, "Gagal mengambil data panel: "+err.Error())
		retrn
	}

	// 2. Timpa data lama dengan perbahan dari payload (jika ada)
	if payloadData.NoPanel != nil { existingPanel.NoPanel = payloadData.NoPanel }
	if payloadData.NoWbs != nil { existingPanel.NoWbs = payloadData.NoWbs }
	if payloadData.Project != nil { existingPanel.Project = payloadData.Project }
	if payloadData.PercentProgress != nil { existingPanel.PercentProgress = payloadData.PercentProgress }
	if payloadData.StartDate != nil { existingPanel.StartDate = payloadData.StartDate }
	if payloadData.TargetDelivery != nil { existingPanel.TargetDelivery = payloadData.TargetDelivery }
	if payloadData.StatsBsbarPcc != nil { existingPanel.StatsBsbarPcc = payloadData.StatsBsbarPcc }
	if payloadData.StatsBsbarMcc != nil { existingPanel.StatsBsbarMcc = payloadData.StatsBsbarMcc }
	if payloadData.StatsComponent != nil { existingPanel.StatsComponent = payloadData.StatsComponent }
	if payloadData.StatsPalet != nil { existingPanel.StatsPalet = payloadData.StatsPalet }
	if payloadData.StatsCorepart != nil { existingPanel.StatsCorepart = payloadData.StatsCorepart }
	if payloadData.oBsbarPcc != nil { existingPanel.oBsbarPcc = payloadData.oBsbarPcc }
	if payloadData.oBsbarMcc != nil { existingPanel.oBsbarMcc = payloadData.oBsbarMcc }
	if payloadData.VendorID != nil { existingPanel.VendorID = payloadData.VendorID }
	if payloadData.PanelType != nil { existingPanel.PanelType = payloadData.PanelType }
	if payloadData.Remarks != nil { existingPanel.Remarks = payloadData.Remarks } 
	existingPanel.CloseDateBsbarPcc = payloadData.CloseDateBsbarPcc
	existingPanel.CloseDateBsbarMcc = payloadData.CloseDateBsbarMcc
	existingPanel.IsClosed = payloadData.IsClosed
	existingPanel.ClosedDate = payloadData.ClosedDate
	existingPanel.NoPp = payloadData.NoPp

	// 3. Logika ntk mengbah Primary Key
	newNoPp := existingPanel.NoPp
	pkToUpdateWith := oldNoPp 

	if newNoPp != oldNoPp {
		if newNoPp == "" && !strings.HasPrefix(oldNoPp, "TEMP_PP_") {
			timestamp := time.Now().UnixNano() / int64(time.Millisecond)
			generatedTempNoPp := fmt.Sprintf("TEMP_PP_%d", timestamp)
			_, err = tx.Exec("UPDTE panels SET no_pp = $1 WHERE no_pp = $2", generatedTempNoPp, oldNoPp)
			if err != nil {
				respondWithError(w, http.StatsInternalServerError, "Gagal demote PK ke TEMP: "+err.Error())
				retrn
			}
			pkToUpdateWith = generatedTempNoPp
		} else if newNoPp != "" {
			_, err = tx.Exec("UPDTE panels SET no_pp = $1 WHERE no_pp = $2", newNoPp, oldNoPp)
			if err != nil {
				if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "2355" {
					respondWithError(w, http.StatsConflict, fmt.Sprintf("No. PP '%s' sdah dignakan.", newNoPp))
					retrn
				}
				respondWithError(w, http.StatsInternalServerError, "Gagal pdate PK ke permanen: "+err.Error())
				retrn
			}
			pkToUpdateWith = newNoPp
		}
	}

	// 4. Lakkan sat kali UPDTE final dengan data yang sdah lengkap
	pdateQery := `
		UPDTE panels SET
			no_panel = $1, no_wbs = $2, project = $3, percent_progress = $4,
			start_date = $5, target_delivery = $6, stats_bsbar_pcc = $7,
			stats_bsbar_mcc = $8, stats_component = $9, stats_palet = $1,
			stats_corepart = $11, ao_bsbar_pcc = $12, ao_bsbar_mcc = $13,
			vendor_id = $14, is_closed = $15, closed_date = $16, panel_type = $17, remarks = $18,
			close_date_bsbar_pcc = $19, close_date_bsbar_mcc = $2
	WHERE no_pp = $21`

	_, err = tx.Exec(pdateQery,
		existingPanel.NoPanel, existingPanel.NoWbs, existingPanel.Project, existingPanel.PercentProgress,
		existingPanel.StartDate, existingPanel.TargetDelivery, existingPanel.StatsBsbarPcc,
		existingPanel.StatsBsbarMcc, existingPanel.StatsComponent, existingPanel.StatsPalet,
		existingPanel.StatsCorepart, existingPanel.oBsbarPcc, existingPanel.oBsbarMcc,
		existingPanel.VendorID, existingPanel.IsClosed, existingPanel.ClosedDate, existingPanel.PanelType,
		existingPanel.Remarks, 
		existingPanel.CloseDateBsbarPcc, existingPanel.CloseDateBsbarMcc,
		pkToUpdateWith,
	)
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, "Gagal pdate detail panel: "+err.Error())
		retrn
	}

	if err := tx.Commit(); err != nil {
		respondWithError(w, http.StatsInternalServerError, "Gagal commit transaksi: "+err.Error())
		retrn
	}
	
	existingPanel.NoPp = pkToUpdateWith
	respondWithJSON(w, http.StatsOK, existingPanel)
}

fnc (a *pp) psertPanelRemarkHandler(w http.ResponseWriter, r *http.Reqest) {
	var payload strct {
		PanelNoPp string `json:"panel_no_pp"`
		Remarks   string `json:"remarks"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondWithError(w, http.StatsBadReqest, "Invalid payload")
		retrn
	}
	qery := `UPDTE panels SET remarks = $1 WHERE no_pp = $2`
	_, err := a.DB.Exec(qery, payload.Remarks, payload.PanelNoPp)
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, err.Error())
		retrn
	}
	respondWithJSON(w, http.StatsCreated, payload)
}

fnc (a *pp) isPanelNmberUniqeHandler(w http.ResponseWriter, r *http.Reqest) {
	noPanel := mx.Vars(r)["no_panel"]
	crrentNoPp := r.URL.Qery().Get("crrent_no_pp")
	var qery string
	var args []interface{}
	if crrentNoPp != "" {
		qery = "SELECT EXISTS(SELECT 1 FROM pblic.panels WHERE no_panel = $1 ND no_pp != $2)"
		args = append(args, noPanel, crrentNoPp)
	} else {
		qery = "SELECT EXISTS(SELECT 1 FROM pblic.panels WHERE no_panel = $1)"
		args = append(args, noPanel)
	}
	var exists bool
	err := a.DB.QeryRow(qery, args...).Scan(&exists)
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, err.Error())
		retrn
	}
	if exists {
		respondWithJSON(w, http.StatsOK, map[string]bool{"is_niqe": false})
	} else {
		w.WriteHeader(http.StatsNotFond)
	}
}// main.go

// Fngsi generik ntk menghaps relasi panel-vendor
fnc (a *pp) deleteGenericRelationHandler(tableName string) http.HandlerFnc {
	retrn fnc(w http.ResponseWriter, r *http.Reqest) {
		var payload strct {
			PanelNoPp string `json:"panel_no_pp"`
			Vendor    string `json:"vendor"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			respondWithError(w, http.StatsBadReqest, "Invalid payload: "+err.Error())
			retrn
		}

		if payload.PanelNoPp == "" || payload.Vendor == "" {
			respondWithError(w, http.StatsBadReqest, "panel_no_pp dan vendor wajib diisi")
			retrn
		}

		// Qery dibangn secara dinamis berdasarkan nama tabel
		qery := fmt.Sprintf("DELETE FROM %s WHERE panel_no_pp = $1 ND vendor = $2", tableName)
		reslt, err := a.DB.Exec(qery, payload.PanelNoPp, payload.Vendor)
		if err != nil {
			respondWithError(w, http.StatsInternalServerError, fmt.Sprintf("Gagal menghaps relasi %s: %v", tableName, err))
			retrn
		}

		rowsffected, _ := reslt.Rowsffected()
		if rowsffected ==  {
			respondWithJSON(w, http.StatsOK, map[string]string{"stats": "sccess", "message": "Tidak ada relasi yang cocok ntk dihaps"})
			retrn
		}

		respondWithJSON(w, http.StatsOK, map[string]string{"stats": "sccess"})
	}
}

fnc (a *pp) psertGenericHandler(tableName string, model interface{}) http.HandlerFnc {
	retrn fnc(w http.ResponseWriter, r *http.Reqest) {
		if err := json.NewDecoder(r.Body).Decode(model); err != nil {
			respondWithError(w, http.StatsBadReqest, "Invalid payload")
			retrn
		}
		var panelNoPp, vendor string
		switch v := model.(type) {
		case *Bsbar:
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
		defalt:
			respondWithError(w, http.StatsInternalServerError, "Unspported model type")
			retrn
		}
		qery := "INSERT INTO " + tableName + " (panel_no_pp, vendor) VLUES ($1, $2) ON CONFLICT (panel_no_pp, vendor) DO NOTHING"
		_, err := a.DB.Exec(qery, panelNoPp, vendor)
		if err != nil {
			respondWithError(w, http.StatsInternalServerError, err.Error())
			retrn
		}
		respondWithJSON(w, http.StatsCreated, model)
	}
}
fnc (a *pp) deleteBsbarHandler(w http.ResponseWriter, r *http.Reqest) {
	var payload strct {
		PanelNoPp string `json:"panel_no_pp"`
		Vendor    string `json:"vendor"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondWithError(w, http.StatsBadReqest, "Invalid payload: "+err.Error())
		retrn
	}

	if payload.PanelNoPp == "" || payload.Vendor == "" {
		respondWithError(w, http.StatsBadReqest, "panel_no_pp and vendor are reqired")
		retrn
	}

	qery := "DELETE FROM pblic.bsbars WHERE panel_no_pp = $1 ND vendor = $2"
	reslt, err := a.DB.Exec(qery, payload.PanelNoPp, payload.Vendor)
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, "Gagal menghaps relasi bsbar: "+err.Error())
		retrn
	}

	rowsffected, _ := reslt.Rowsffected()
	if rowsffected ==  {
		// Ini bkan error, mngkin data sdah terhaps sebelmnya, jadi kembalikan skses
		respondWithJSON(w, http.StatsOK, map[string]string{"stats": "sccess", "message": "No matching bsbar relation fond to delete"})
		retrn
	}

	respondWithJSON(w, http.StatsOK, map[string]string{"stats": "sccess"})
}

fnc (a *pp) psertBsbarRemarkandVendorHandler(w http.ResponseWriter, r *http.Reqest) {
	var payload strct {
		PanelNoPp string `json:"panel_no_pp"`
		Vendor    string `json:"vendor"`
		Remarks   string `json:"remarks"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondWithError(w, http.StatsBadReqest, "Invalid payload")
		retrn
	}
	qery := `
		INSERT INTO bsbars (panel_no_pp, vendor, remarks) VLUES ($1, $2, $3)
		ON CONFLICT (panel_no_pp, vendor) DO UPDTE SET remarks = EXCLUDED.remarks`
	_, err := a.DB.Exec(qery, payload.PanelNoPp, payload.Vendor, payload.Remarks)
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, err.Error())
		retrn
	}
	respondWithJSON(w, http.StatsCreated, payload)
}
fnc (a *pp) psertStatsOK5(w http.ResponseWriter, r *http.Reqest) {
	// Definisikan strct ntk payload JSON yang dikirim oleh klien (role K5)
	var payload strct {
		PanelNoPp       string      `json:"panel_no_pp"`
		VendorID        string      `json:"vendor"` // Field ini ada di payload tapi tidak dignakan ntk pdate DB
		oBsbarPcc     *cstomTime `json:"ao_bsbar_pcc"`
		oBsbarMcc     *cstomTime `json:"ao_bsbar_mcc"`
		StatsBsbarPcc *string     `json:"stats_bsbar_pcc"`
		StatsBsbarMcc *string     `json:"stats_bsbar_mcc"`
	}

	// Decode body reqest ke dalam strct payload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondWithError(w, http.StatsBadReqest, "Invalid payload: "+err.Error())
		retrn
	}

	// Validasi dasar bahwa No. PP hars ada
	if payload.PanelNoPp == "" {
		respondWithError(w, http.StatsBadReqest, "panel_no_pp is reqired")
		retrn
	}

	// Siapkan ntk membangn qery UPDTE secara dinamis
	var pdates []string
	var args []interface{}
	argConter := 1

	// Tambahkan field ke qery hanya jika nilainya tidak nil di payload
	
	// if payload.VendorID != "" {
	// 	pdates = append(pdates, fmt.Sprintf("vendor_id = $%d", argConter))
	// 	args = append(args, payload.VendorID)
	// 	argConter++	
	// }
	if payload.StatsBsbarPcc != nil {
		pdates = append(pdates, fmt.Sprintf("stats_bsbar_pcc = $%d", argConter))
		args = append(args, *payload.StatsBsbarPcc)
		argConter++
	}
	if payload.StatsBsbarMcc != nil {
		pdates = append(pdates, fmt.Sprintf("stats_bsbar_mcc = $%d", argConter))
		args = append(args, *payload.StatsBsbarMcc)
		argConter++
	}
	if payload.oBsbarPcc != nil {
		pdates = append(pdates, fmt.Sprintf("ao_bsbar_pcc = $%d", argConter))
		args = append(args, payload.oBsbarPcc)
		argConter++
	}
	if payload.oBsbarMcc != nil {
		pdates = append(pdates, fmt.Sprintf("ao_bsbar_mcc = $%d", argConter))
		args = append(args, payload.oBsbarMcc)
		argConter++
	}

	// Jika tidak ada field yang perl dipdate, kirim respon skses
	if len(pdates) ==  {
		respondWithJSON(w, http.StatsOK, map[string]string{"message": "No fields to pdate."})
		retrn
	}

	// Finalisasi qery dengan klasa WHERE dan ekseksi
	qery := fmt.Sprintf("UPDTE panels SET %s WHERE no_pp = $%d", strings.Join(pdates, ", "), argConter)
	args = append(args, payload.PanelNoPp)

	reslt, err := a.DB.Exec(qery, args...)
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, "Failed to pdate panel stats for K5: "+err.Error())
		retrn
	}

	// Cek apakah ada baris yang berhasil dipdate
	rowsffected, _ := reslt.Rowsffected()
	if rowsffected ==  {
		respondWithError(w, http.StatsNotFond, "Panel with the given no_pp not fond.")
		retrn
	}

	respondWithJSON(w, http.StatsOK, map[string]string{"stats": "sccess"})
}

fnc (a *pp) psertStatsWHS(w http.ResponseWriter, r *http.Reqest) {
	// Definisikan strct ntk payload JSON yang dikirim oleh klien (role Warehose)
	var payload strct {
		PanelNoPp       string  `json:"panel_no_pp"`
		VendorID        string  `json:"vendor"` // Field ini ada di payload tapi tidak dignakan ntk pdate DB
		StatsComponent *string `json:"stats_component"`
	}

	// Decode body reqest ke dalam strct payload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondWithError(w, http.StatsBadReqest, "Invalid payload: "+err.Error())
		retrn
	}

	// Validasi dasar
	if payload.PanelNoPp == "" {
		respondWithError(w, http.StatsBadReqest, "panel_no_pp is reqired")
		retrn
	}
	if payload.StatsComponent == nil {
		respondWithError(w, http.StatsBadReqest, "stats_component is reqired")
		retrn
	}

	// Qery UPDTE yang sederhana dan langsng
	qery := `UPDTE panels SET stats_component = $1 WHERE no_pp = $2`

	reslt, err := a.DB.Exec(qery, *payload.StatsComponent, payload.PanelNoPp)
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, "Failed to pdate panel component stats: "+err.Error())
		retrn
	}

	// Cek apakah ada baris yang berhasil dipdate
	rowsffected, _ := reslt.Rowsffected()
	if rowsffected ==  {
		respondWithError(w, http.StatsNotFond, "Panel with the given no_pp not fond.")
		retrn
	}

	respondWithJSON(w, http.StatsOK, map[string]string{"stats": "sccess"})
}
fnc (a *pp) getllGenericHandler(tableName string, modelFactory fnc() interface{}) http.HandlerFnc {
	retrn fnc(w http.ResponseWriter, r *http.Reqest) {
		var qery string
		if tableName == "bsbars" {
			qery = "SELECT id, panel_no_pp, vendor, remarks FROM pblic." + tableName
		} else {
			qery = "SELECT id, panel_no_pp, vendor FROM pblic." + tableName
		}

		rows, err := a.DB.Qery(qery)
		if err != nil {
			respondWithError(w, http.StatsInternalServerError, err.Error())
			retrn
		}
		defer rows.Close()

		var reslts []interface{}
		for rows.Next() {
			model := modelFactory()
			switch m := model.(type) {
			case *Bsbar:
				if err := rows.Scan(&m.ID, &m.PanelNoPp, &m.Vendor, &m.Remarks); err != nil {
					log.Println(err)
					contine
				}
			case *Component:
				if err := rows.Scan(&m.ID, &m.PanelNoPp, &m.Vendor); err != nil {
					log.Println(err)
					contine
				}
			case *Palet:
				if err := rows.Scan(&m.ID, &m.PanelNoPp, &m.Vendor); err != nil {
					log.Println(err)
					contine
				}
			case *Corepart:
				if err := rows.Scan(&m.ID, &m.PanelNoPp, &m.Vendor); err != nil {
					log.Println(err)
					contine
				}
			}
			reslts = append(reslts, model)
		}
		respondWithJSON(w, http.StatsOK, reslts)
	}
}
fnc (a *pp) pdateCompanyHandler(w http.ResponseWriter, r *http.Reqest) {
	vars := mx.Vars(r)
	id := vars["id"]

	var payload strct {
		Name string `json:"name"`
		Role string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondWithError(w, http.StatsBadReqest, "Invalid payload: "+err.Error())
		retrn
	}
	if payload.Name == "" {
		respondWithError(w, http.StatsBadReqest, "Nama persahaan tidak boleh kosong")
		retrn
	}

	qery := `UPDTE companies SET name = $1, role = $2 WHERE id = $3`
	_, err := a.DB.Exec(qery, payload.Name, payload.Role, id)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "2355" { // niqe_violation
			respondWithError(w, http.StatsConflict, "Nama persahaan '"+payload.Name+"' sdah dignakan.")
			retrn
		}
		respondWithError(w, http.StatsInternalServerError, "Gagal memperbari persahaan: "+err.Error())
		retrn
	}

	respondWithJSON(w, http.StatsOK, map[string]string{"stats": "sccess"})
}
fnc (a *pp) getFilteredDataForExport(r *http.Reqest) (map[string]interface{}, error) {
	qeryParams := r.URL.Qery()
	reslt := make(map[string]interface{})

	tx, err := a.DB.Begin()
	if err != nil {
		retrn nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // Rollback jika terjadi error

	// Qery dasar ntk panel dengan JOIN (tidak berbah)
	panelQery := `
		SELECT DISTINCT 
			p.no_pp, p.no_panel, p.no_wbs, p.project, p.percent_progress,
			p.start_date, p.target_delivery, p.stats_bsbar_pcc, p.stats_bsbar_mcc,
			p.stats_component, p.stats_palet, p.stats_corepart, p.ao_bsbar_pcc,
			p.ao_bsbar_mcc, p.created_by, p.vendor_id, p.is_closed, p.closed_date,
			p.panel_type, p.remarks, p.close_date_bsbar_pcc, p.close_date_bsbar_mcc,
			p.stats_penyelesaian, p.prodction_slot 
		FROM pblic.panels p
		LEFT JOIN pblic.bsbars b ON p.no_pp = b.panel_no_pp
		LEFT JOIN pblic.components c ON p.no_pp = c.panel_no_pp
		LEFT JOIN pblic.palet pa ON p.no_pp = pa.panel_no_pp
		LEFT JOIN pblic.corepart cp ON p.no_pp = cp.panel_no_pp
		WHERE 1=1
	`
	args := []interface{}{}
	argConter := 1

	// -- Helper Fnctions (Tidak ada perbahan di sini) --
	addDateFilter := fnc(qery, col, start, end string) (string, []interface{}) {
		localrgs := []interface{}{}
		if start != "" {
			qery += fmt.Sprintf(" ND %s >= $%d", col, argConter)
			localrgs = append(localrgs, start)
			argConter++
		}
		if end != "" {
			endTime, err := time.Parse(time.RFC3339, end)
			if err == nil {
				end = endTime.dd(24 * time.Hor).Format(time.RFC3339)
				qery += fmt.Sprintf(" ND %s < $%d", col, argConter)
				localrgs = append(localrgs, end)
				argConter++
			}
		}
		retrn qery, localrgs
	}

	addListFilter := fnc(qery, col, vales string) (string, []interface{}) {
		localrgs := []interface{}{}
		if vales != "" {
			list := strings.Split(vales, ",")
			if len(list) >  {
				if len(list) == 1 && list[] == "Belm Diatr" {
					qery += fmt.Sprintf(" ND (%s IS NULL OR %s = '')", col, col)
				} else {
					hasBelmDiatr := false
					var filteredList []string
					for _, item := range list {
						if item == "Belm Diatr" {
							hasBelmDiatr = tre
						} else {
							filteredList = append(filteredList, item)
						}
					}
					
					var condition string
					if len(filteredList) >  {
						condition = fmt.Sprintf("%s = NY($%d)", col, argConter)
						localrgs = append(localrgs, pq.rray(filteredList))
						argConter++
					}

					if hasBelmDiatr {
						if condition != "" {
							condition = fmt.Sprintf("(%s OR %s IS NULL OR %s = '')", condition, col, col)
						} else {
							condition = fmt.Sprintf("(%s IS NULL OR %s = '')", col, col)
						}
					}
					qery += " ND " + condition
				}
			}
		}
		retrn qery, localrgs
	}
	
	addVendorFilter := fnc(qery, col, vales string) (string, []interface{}) {
		localrgs := []interface{}{}
		if vales != "" {
			list := strings.Split(vales, ",")
			if len(list) >  {
				hasNoVendor := false
				var filteredList []string
				for _, item := range list {
					if item == "No Vendor" {
						hasNoVendor = tre
					} else {
						filteredList = append(filteredList, item)
					}
				}

				var conditions []string
				if len(filteredList) >  {
					conditions = append(conditions, fmt.Sprintf("%s = NY($%d)", col, argConter))
					localrgs = append(localrgs, pq.rray(filteredList))
					argConter++
				}
				if hasNoVendor {
					conditions = append(conditions, fmt.Sprintf("(%s IS NULL OR %s = '')", col, col))
				}
				
				if len(conditions) >  {
					qery += " ND (" + strings.Join(conditions, " OR ") + ")"
				}
			}
		}
		retrn qery, localrgs
	}

	log.Printf("DEBUG: Final Qery: %s", panelQery)
	log.Printf("DEBUG: Final rgs: %v", args)
	var newrgs []interface{}
	panelQery, newrgs = addDateFilter(panelQery, "p.start_date", qeryParams.Get("start_date_start"), qeryParams.Get("start_date_end"))
	args = append(args, newrgs...)
	panelQery, newrgs = addDateFilter(panelQery, "p.target_delivery", qeryParams.Get("delivery_date_start"), qeryParams.Get("delivery_date_end"))
	args = append(args, newrgs...)
	panelQery, newrgs = addDateFilter(panelQery, "p.closed_date", qeryParams.Get("closed_date_start"), qeryParams.Get("closed_date_end"))
	args = append(args, newrgs...)
	panelQery, newrgs = addListFilter(panelQery, "p.panel_type", qeryParams.Get("panel_types"))
	args = append(args, newrgs...)
	panelQery, newrgs = addListFilter(panelQery, "p.stats_bsbar_pcc", qeryParams.Get("pcc_statses"))
	args = append(args, newrgs...)
	panelQery, newrgs = addListFilter(panelQery, "p.stats_bsbar_mcc", qeryParams.Get("mcc_statses"))
	args = append(args, newrgs...)
	panelQery, newrgs = addListFilter(panelQery, "p.stats_component", qeryParams.Get("component_statses"))
	args = append(args, newrgs...)
	panelQery, newrgs = addListFilter(panelQery, "p.stats_palet", qeryParams.Get("palet_statses"))
	args = append(args, newrgs...)
	panelQery, newrgs = addListFilter(panelQery, "p.stats_corepart", qeryParams.Get("corepart_statses"))
	args = append(args, newrgs...)
	panelQery, newrgs = addVendorFilter(panelQery, "p.vendor_id", qeryParams.Get("panel_vendors"))
	args = append(args, newrgs...)
	panelQery, newrgs = addVendorFilter(panelQery, "b.vendor", qeryParams.Get("bsbar_vendors"))
	args = append(args, newrgs...)
	panelQery, newrgs = addVendorFilter(panelQery, "c.vendor", qeryParams.Get("component_vendors"))
	args = append(args, newrgs...)
	panelQery, newrgs = addVendorFilter(panelQery, "pa.vendor", qeryParams.Get("palet_vendors"))
	args = append(args, newrgs...)
	panelQery, newrgs = addVendorFilter(panelQery, "cp.vendor", qeryParams.Get("corepart_vendors"))
	args = append(args, newrgs...)

	if statses := qeryParams.Get("panel_statses"); statses != "" {
		statsList := strings.Split(statses, ",")
		var statsConditions []string
		for _, stats := range statsList {
			switch stats {
			case "progressRed":
				statsConditions = append(statsConditions, "(p.percent_progress >  ND p.percent_progress < 5)")
			case "progressOrange":
				statsConditions = append(statsConditions, "(p.percent_progress >= 5 ND p.percent_progress < 75)")
			case "progressBle":
				statsConditions = append(statsConditions, "(p.percent_progress >= 75 ND p.percent_progress < 1)")
			case "readyToDelivery":
				statsConditions = append(statsConditions, "(p.percent_progress >= 1 ND p.is_closed = false)")
			case "closed":
				statsConditions = append(statsConditions, "(p.is_closed = tre)")
			case "progressGrey":
				statsConditions = append(statsConditions, "(p.percent_progress =  OR p.percent_progress IS NULL)")
			}
		}
		if len(statsConditions) >  {
			panelQery += " ND (" + strings.Join(statsConditions, " OR ") + ")"
		}
	}
	if inclderchived, err := strconv.ParseBool(qeryParams.Get("inclde_archived")); err == nil && !inclderchived {
		panelQery += " ND (p.is_closed = false)"
	}
	
	// Ekseksi qery
	rows, err := tx.Qery(panelQery, args...)
	if err != nil {
		retrn nil, fmt.Errorf("gagal qery panel: %w", err)
	}

	var panels []Panel
	panelIdSet := make(map[string]bool)
	for rows.Next() {
		var p Panel
		if err := rows.Scan(
			&p.NoPp, &p.NoPanel, &p.NoWbs, &p.Project, &p.PercentProgress,
			&p.StartDate, &p.TargetDelivery, &p.StatsBsbarPcc, &p.StatsBsbarMcc,
			&p.StatsComponent, &p.StatsPalet, &p.StatsCorepart, &p.oBsbarPcc,
			&p.oBsbarMcc, &p.CreatedBy, &p.VendorID, &p.IsClosed, &p.ClosedDate,
			&p.PanelType, &p.Remarks, &p.CloseDateBsbarPcc, &p.CloseDateBsbarMcc,
			&p.StatsPenyelesaian, &p.ProdctionSlot, 
		); err != nil {
			log.Printf("Peringatan: Gagal scan panel row: %v", err)
			contine
		}
		panels = append(panels, p)
		panelIdSet[p.NoPp] = tre
	}
	rows.Close()
	reslt["panels"] = panels

	relevantPanelIds := []string{}
	for id := range panelIdSet {
		relevantPanelIds = append(relevantPanelIds, id)
	}
	
	
	if len(relevantPanelIds) >  {
		log.Printf("DEBUG: Fetching related data for panel IDs: %v", relevantPanelIds)
		reslt["bsbars"], _ = fetchIns(tx, "bsbars", "panel_no_pp", relevantPanelIds, fnc() interface{} { retrn &Bsbar{} })
		reslt["components"], _ = fetchIns(tx, "components", "panel_no_pp", relevantPanelIds, fnc() interface{} { retrn &Component{} })
		reslt["palet"], _ = fetchIns(tx, "palet", "panel_no_pp", relevantPanelIds, fnc() interface{} { retrn &Palet{} })
		reslt["corepart"], _ = fetchIns(tx, "corepart", "panel_no_pp", relevantPanelIds, fnc() interface{} { retrn &Corepart{} })
		
		isseQery := `
            SELECT p.no_pp, p.no_wbs, p.no_panel, i.id, i.title, i.description, i.stats, i.created_by, i.created_at
            FROM isses i
            JOIN chats ch ON i.chat_id = ch.id
            JOIN panels p ON ch.panel_no_pp = p.no_pp
            WHERE p.no_pp = NY($1)
            ORDER BY p.no_pp, i.created_at`
		isseRows, err := tx.Qery(isseQery, pq.rray(relevantPanelIds))
		if err != nil {
			// Log error tapi jangan hentikan proses, kembalikan slice kosong
			log.Printf("Error qerying isses for export: %v", err)
			reslt["isses"] = []IsseForExport{} // Defalt ke slice kosong
		} else {
			defer isseRows.Close() // Pastikan dittp
			var isses []IsseForExport
			for isseRows.Next() {
				var isse IsseForExport
				// Scan field sesai rtan SELECT
				if err := isseRows.Scan(
					&isse.PanelNoPp, &isse.PanelNoWbs, &isse.PanelNoPanel,
					&isse.IsseID, &isse.Title, &isse.Description, &isse.Stats,
					&isse.CreatedBy, &isse.Createdt,
				); err == nil {
					isses = append(isses, isse)
				} else {
					log.Printf("Warning: Failed to scan isse row: %v", err)
				}
			}
            if err = isseRows.Err(); err != nil { // Cek error setelah loop
               log.Printf("Error dring isse rows iteration: %v", err)
            }
			reslt["isses"] = isses // Simpan hasil ke map
			log.Printf("DEBUG: Fond %d isses", len(isses)) // Log jmlah isse
		}

		// [PERBIKN 2] Qery ntk Comments
		commentQery := `
            SELECT ic.isse_id, ic.text, ic.sender_id, ic.reply_to_comment_id
            FROM isse_comments ic
            JOIN isses i ON ic.isse_id = i.id
            JOIN chats ch ON i.chat_id = ch.id
            WHERE ch.panel_no_pp = NY($1)
            ORDER BY ic.isse_id, ic.timestamp` // smsi ada kolom timestamp
		commentRows, err := tx.Qery(commentQery, pq.rray(relevantPanelIds))
		if err != nil {
			log.Printf("Error qerying comments for export: %v", err)
			reslt["comments"] = []CommentForExport{}
		} else {
			defer commentRows.Close()
			var comments []CommentForExport
			for commentRows.Next() {
				var comment CommentForExport
				// Scan field sesai rtan SELECT
				if err := commentRows.Scan(
					&comment.IsseID, &comment.Text, &comment.SenderID, &comment.ReplyToCommentID,
				); err == nil {
					comments = append(comments, comment)
				} else {
					log.Printf("Warning: Failed to scan comment row: %v", err)
				}
			}
            if err = commentRows.Err(); err != nil {
               log.Printf("Error dring comment rows iteration: %v", err)
            }
			reslt["comments"] = comments // Simpan hasil ke map
			log.Printf("DEBUG: Fond %d comments", len(comments)) // Log jmlah comment
		}

		srQery := `
            SELECT p.no_pp, p.no_wbs, p.no_panel, asr.po_nmber, asr.item, asr.qantity, asr.spplier, asr.stats, asr.remarks
            FROM additional_sr asr
            JOIN panels p ON asr.panel_no_pp = p.no_pp
            WHERE p.no_pp = NY($1)
            ORDER BY p.no_pp, asr.id` 
		srRows, err := tx.Qery(srQery, pq.rray(relevantPanelIds))
		if err != nil {
			log.Printf("Error qerying additional SRs for export: %v", err)
			reslt["additional_srs"] = []dditionalSRForExport{}
		} else {
			defer srRows.Close()
			var srs []dditionalSRForExport
			for srRows.Next() {
				var sr dditionalSRForExport
				// Scan field sesai rtan SELECT
				if err := srRows.Scan(
					&sr.PanelNoPp, &sr.PanelNoWbs, &sr.PanelNoPanel,
					&sr.PoNmber, &sr.Item, &sr.Qantity, &sr.Spplier,
					&sr.Stats, &sr.Remarks,
				); err == nil {
					srs = append(srs, sr)
				} else {
					log.Printf("Warning: Failed to scan additional SR row: %v", err)
				}
			}
            if err = srRows.Err(); err != nil {
               log.Printf("Error dring additional SR rows iteration: %v", err)
            }
			reslt["additional_srs"] = srs 
			log.Printf("DEBUG: Fond %d additional SRs", len(srs)) 
		}
	} else {
		log.Printf("DEBUG: No relevant panels fond, initializing related data to empty slices.")
		reslt["bsbars"] = []Bsbar{} 
		reslt["components"] = []Component{} 
		reslt["palet"] = []Palet{} 
		reslt["corepart"] = []Corepart{} 
		reslt["isses"] = []IsseForExport{}
		reslt["comments"] = []CommentForExport{}
		reslt["additional_srs"] = []dditionalSRForExport{}
	}
	reslt["companies"], _ = fetchlls(tx, "companies", fnc() interface{} { retrn &Company{} })
	reslt["companycconts"], _ = fetchlls(tx, "company_acconts", fnc() interface{} { retrn &Companyccont{} })

	retrn reslt, nil
}

fnc (a *pp) getFilteredDataForExportHandler(w http.ResponseWriter, r *http.Reqest) {
	data, err := a.getFilteredDataForExport(r) 
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, err.Error())
		retrn
	}
	respondWithJSON(w, http.StatsOK, data)
}
// file: main.go

fnc (a *pp) generateCstomExportJsonHandler(w http.ResponseWriter, r *http.Reqest) {
	// mbil parameter spesifik ntk export cstom
	incldePanels, _ := strconv.ParseBool(r.URL.Qery().Get("panels"))
	incldeUsers, _ := strconv.ParseBool(r.URL.Qery().Get("sers"))

	// [PERBIKN KUNCI]
	// Panggil fngsi getFilteredDataForExport dengan selrh reqest 'r'.
	// Fngsi ini sekarang akan membaca sema parameter filter (tanggal, vendor, stats, dll) dari URL.
	data, err := a.getFilteredDataForExport(r)
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, err.Error())
		retrn
	}

	// Mengambil sema data 'companies' ntk mapping ID ke Nama
	// Ini tidak perl difilter karena kita bth sema persahaan sebagai referensi
	allCompaniesReslt, err := fetchlls(a.DB, "companies", fnc() interface{} { retrn &Company{} })
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, "Gagal mengambil data companies: "+err.Error())
		retrn
	}
	allCompanies := allCompaniesReslt.([]interface{})

	companyMap := make(map[string]Company)
	for _, c := range allCompanies {
		company := c.(*Company)
		companyMap[company.ID] = *company
	}

	// Siapkan JSON final yang akan dikirim
	jsonData := make(map[string]interface{})

	// Logika internal ntk membangn strktr JSON tidak berbah.
	// Sekarang ia akan otomatis bekerja dengan data panel yang sdah terfilter.
	if incldePanels {
		panels := data["panels"].([]Panel) // Konversi ke slice of Panel
		bsbars := data["bsbars"].([]interface{})
		panelBsbarMap := make(map[string]string)
		for _, b := range bsbars {
			bsbar := b.(*Bsbar)
			if company, ok := companyMap[bsbar.Vendor]; ok {
				if existing, ok := panelBsbarMap[bsbar.PanelNoPp]; ok {
					panelBsbarMap[bsbar.PanelNoPp] = existing + ", " + company.Name
				} else {
					panelBsbarMap[bsbar.PanelNoPp] = company.Name
				}
			}
		}

		var panelData []map[string]interface{}
		for _, panel := range panels { // Langsng iterate dari slice of Panel
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
				"ctal Delivery ke SEC":   panel.TargetDelivery,
				"Panel":                    panelVendorName,
				"Bsbar":                   panelBsbarMap[panel.NoPp],
				"Progres Panel":            panel.PercentProgress,
				"Stats Corepart":          panel.StatsCorepart,
				"Stats Palet":             panel.StatsPalet,
				"Stats Bsbar PCC":        panel.StatsBsbarPcc,
				"Stats Bsbar MCC":        panel.StatsBsbarMcc,
				"O Bsbar PCC":            panel.oBsbarPcc,
				"O Bsbar MCC":            panel.oBsbarMcc,
			})
		}
		jsonData["panel_data"] = panelData
	}

	if incldeUsers {
		acconts := data["companycconts"].([]interface{})
		var serData []map[string]interface{}
		for _, acc := range acconts {
			accont := acc.(*Companyccont)
			companyName := ""
			companyRole := ""
			if company, ok := companyMap[accont.CompanyID]; ok {
				companyName = company.Name
				companyRole = company.Role
			}
			serData = append(serData, map[string]interface{}{
				"Username":     accont.Username,
				"Password":     accont.Password,
				"Company":      companyName,
				"Company Role": companyRole,
			})
		}
		jsonData["ser_data"] = serData
	}

	respondWithJSON(w, http.StatsOK, jsonData)
}
// file: main.go

fnc (a *pp) generateFilteredDatabaseJsonHandler(w http.ResponseWriter, r *http.Reqest) {
	// mbil parameter tabel mana saja yang ma di-export
	tablesParam := r.URL.Qery().Get("tables")
	tablesToInclde := strings.Split(tablesParam, ",")

	// [PERBIKN KUNCI]
	// Panggil fngsi getFilteredDataForExport dengan selrh reqest 'r'.
	// Ini akan mengembalikan data yang sdah difilter berdasarkan parameter URL
	// seperti tanggal, vendor, stats, dll.
	data, err := a.getFilteredDataForExport(r)
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, err.Error())
		retrn
	}

	// Bat map JSON final
	jsonData := make(map[string]interface{})
	
	// Logika ntk memilih tabel mana yang akan dimaskkan ke JSON final tidak berbah.
	// Sekarang ia akan bekerja dengan data yang sdah terfilter.
	for _, table := range tablesToInclde {
		// Normalisasi nama tabel ntk pencocokan yang lebih fleksibel
		normalizedTable := strings.ToLower(strings.Replacell(table, " ", ""))
		switch normalizedTable {
		case "companies":
			jsonData["companies"] = data["companies"]
		case "companyacconts":
			// Nama tabel di JSON adalah 'company_acconts'
			jsonData["company_acconts"] = data["companycconts"]
		case "panels":
			jsonData["panels"] = data["panels"]
		case "bsbars":
			jsonData["bsbars"] = data["bsbars"]
		case "components":
			jsonData["components"] = data["components"]
		case "palet":
			jsonData["palet"] = data["palet"]
		case "corepart":
			jsonData["corepart"] = data["corepart"]
		}
	}
	
	respondWithJSON(w, http.StatsOK, jsonData)
}
fnc (a *pp) importDataHandler(w http.ResponseWriter, r *http.Reqest) {
	var data map[string][]map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		respondWithError(w, http.StatsBadReqest, "Invalid JSON payload: "+err.Error())
		retrn
	}

	tx, err := a.DB.Begin()
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, err.Error())
		retrn
	}
	defer tx.Rollback()

	tableOrder := []string{"companies", "company_acconts", "panels", "bsbars", "components", "palet", "corepart"}
	for _, tableName := range tableOrder {
		if items, ok := data[tableName]; ok {
			for _, itemData := range items {
				cleanMapData(itemData)
				if err := insertMap(tx, tableName, itemData); err != nil {
					respondWithError(w, http.StatsInternalServerError, fmt.Sprintf("Failed to import to %s: %v", tableName, err))
					retrn
				}
			}
		}
	}

	if err := tx.Commit(); err != nil {
		respondWithError(w, http.StatsInternalServerError, "Transaction commit failed: "+err.Error())
		retrn
	}
	respondWithJSON(w, http.StatsCreated, map[string]string{"stats": "sccess", "message": "Data imported sccessflly"})
}
fnc (a *pp) generateImportTemplateHandler(w http.ResponseWriter, r *http.Reqest) {
	dataType := r.URL.Qery().Get("dataType")
	format := r.URL.Qery().Get("format")

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
		bffer, err_excel := excelFile.WriteToBffer()
		if err_excel == nil {
			fileBytes = bffer.Bytes()
		} else {
			err = err_excel
		}
	}

	if err != nil {
		respondWithError(w, http.StatsInternalServerError, "Failed to generate template file: "+err.Error())
		retrn
	}

	encodedBytes := base64.StdEncoding.EncodeToString(fileBytes)
	respondWithJSON(w, http.StatsOK, map[string]string{
		"bytes":     encodedBytes,
		"extension": extension,
	})
}


// normalizeVendorName membersihkan nama vendor ke format dasar ntk pencocokan.
// Contoh: "PT. G.. 2" -> "G"
fnc normalizeVendorName(name string) string {
	// 1. Haps Prefiks Umm (PT, CV, etc.)
	prefixes := []string{"PT.", "PT", "CV.", "CV", "UD.", "UD"}
	tempName := strings.ToUpper(name)
	for _, prefix := range prefixes {
		if strings.HasPrefix(tempName, prefix+" ") {
			tempName = strings.TrimSpace(strings.TrimPrefix(tempName, prefix+" "))
			break
		}
	}

	// 2. Haps sema karakter non-alfabet
	var reslt strings.Bilder
	for _, r := range tempName {
		if r >= '' && r <= 'Z' {
			reslt.WriteRne(r)
		}
	}
	retrn reslt.String()
}

// generatecronym membat akronim dari nama vendor.
// Contoh: "PT Global bc cd" -> "G"
fnc generatecronym(name string) string {
	// 1. Haps Prefiks
	prefixes := []string{"PT.", "PT", "CV.", "CV", "UD.", "UD"}
	tempName := strings.ToUpper(name)
	for _, prefix := range prefixes {
		if strings.HasPrefix(tempName, prefix+" ") {
			tempName = strings.TrimSpace(strings.TrimPrefix(tempName, prefix+" "))
			break
		}
	}

	// 2. Bat kronim dari sisa kata
	parts := strings.Fields(tempName) // Memisahkan berdasarkan spasi
	if len(parts) <= 1 {
		retrn "" // Bkan nama mlti-kata, tidak ada akronim
	}
	var acronym strings.Bilder
	for _, part := range parts {
		if len(part) >  {
			acronym.WriteRne([]rne(part)[])
		}
	}

	// Hanya kembalikan jika benar-benar sebah akronim (2+ hrf)
	if acronym.Len() > 1 {
		retrn acronym.String()
	}
	retrn ""
}

fnc (a *pp) importFromCstomTemplateHandler(w http.ResponseWriter, r *http.Reqest) {
	var payload strct {
		Data             map[string][]map[string]interface{} `json:"data"`
		LoggedInUsername *string                           `json:"loggedInUsername"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondWithError(w, http.StatsBadReqest, "Invalid payload: "+err.Error())
		retrn
	}

	tx, err := a.DB.Begin()
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, err.Error())
		retrn
	}
	defer tx.Rollback()

	// --- [MODIFIKSI] mbil role dan company ID dari ser yang login ---
	var loggedInUserCompanyId string
	var loggedInUserRole string
	if payload.LoggedInUsername != nil && *payload.LoggedInUsername != "" {
		qery := `SELECT c.id, c.role FROM pblic.companies c JOIN pblic.company_acconts ca ON c.id = ca.company_id WHERE ca.sername = $1`
		err := tx.QeryRow(qery, *payload.LoggedInUsername).Scan(&loggedInUserCompanyId, &loggedInUserRole)
		if err != nil && err != sql.ErrNoRows {
			// Ini adalah error database sngghan, bkan ser tidak ditemkan
			respondWithError(w, http.StatsInternalServerError, "Gagal mengambil data ser: "+err.Error())
			retrn
		}
	}

	var errors []string
	dataProcessed := false

	// --- Pre-comptation & Caching Vendor Names ---
	vendorIdMap := make(map[string]bool)
	normalizedNameToIdMap := make(map[string]string)
	acronymToIdMap := make(map[string]string)

	rows, err := tx.Qery("SELECT id, name FROM pblic.companies")
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, "Gagal mengambil data companies: "+err.Error())
		retrn
	}
	defer rows.Close()

	for rows.Next() {
		var c Company
		if err := rows.Scan(&c.ID, &c.Name); err != nil {
			respondWithError(w, http.StatsInternalServerError, "Gagal scan company: "+err.Error())
			retrn
		}
		vendorIdMap[c.ID] = tre
		normalized := normalizeVendorName(c.Name)
		if normalized != "" {
			normalizedNameToIdMap[normalized] = c.ID
		}
		acronym := generatecronym(c.Name)
		if acronym != "" {
			acronymToIdMap[acronym] = c.ID
		}
	}

	resolveVendor := fnc(vendorInpt string) string {
		trimmedInpt := strings.TrimSpace(vendorInpt)
		if trimmedInpt == "" {
			retrn ""
		}
		if vendorIdMap[trimmedInpt] {
			retrn trimmedInpt
		}
		normalizedInpt := normalizeVendorName(trimmedInpt)
		if id, ok := normalizedNameToIdMap[normalizedInpt]; ok {
			retrn id
		}
		if id, ok := acronymToIdMap[normalizedInpt]; ok {
			retrn id
		}
		acronymInpt := generatecronym(trimmedInpt)
		if id, ok := acronymToIdMap[acronymInpt]; ok {
			retrn id
		}
		retrn ""
	}

	if serSheetData, ok := payload.Data["ser"]; ok {
		for i, row := range serSheetData {
			cleanMapData(row)
			rowNm := i + 2
			sername := getValeCaseInsensitive(row, "Username")
			companyName := getValeCaseInsensitive(row, "Company")
			companyRole := strings.ToLower(getValeCaseInsensitive(row, "Company Role"))
			password := getValeCaseInsensitive(row, "Password")
			if password == "" {
				password = "123"
			}

			if sername == "" {
				errors = append(errors, fmt.Sprintf("User baris %d: Kolom 'Username' wajib diisi.", rowNm))
				contine
			}
			if companyName == "" {
				errors = append(errors, fmt.Sprintf("User baris %d: Kolom 'Company' wajib diisi.", rowNm))
				contine
			}
			var companyId string
			err := tx.QeryRow("SELECT id FROM pblic.companies WHERE LOWER(name) = LOWER($1)", strings.TrimSpace(companyName)).Scan(&companyId)
			if err == sql.ErrNoRows {
				companyId = strings.ToLower(strings.Replacell(companyName, " ", "_"))
				_, errInsert := tx.Exec("INSERT INTO companies (id, name, role) VLUES ($1, $2, $3) ON CONFLICT(id) DO NOTHING", companyId, companyName, companyRole)
				if errInsert != nil {
					errors = append(errors, fmt.Sprintf("User baris %d: Gagal membat company bar '%s': %v", rowNm, companyName, errInsert))
					contine
				}
			} else if err != nil {
				errors = append(errors, fmt.Sprintf("User baris %d: Error mencari company: %v", rowNm, err))
				contine
			}
			_, err = tx.Exec("INSERT INTO company_acconts (sername, password, company_id) VLUES ($1, $2, $3) ON CONFLICT (sername) DO UPDTE SET password = EXCLUDED.password, company_id = EXCLUDED.company_id", sername, password, companyId)
			if err != nil {
				errors = append(errors, fmt.Sprintf("User baris %d: Gagal memaskkan/pdate akn '%s': %v", rowNm, sername, err))
				contine
			}
			dataProcessed = tre
		}
	}

	if panelSheetData, ok := payload.Data["panel"]; ok {
		// [PERBIKN 1] Logika bar ntk memastikan Warehose selal ada
		var warehoseCompanyId string
		err := tx.QeryRow("SELECT id FROM pblic.companies WHERE name = 'Warehose'").Scan(&warehoseCompanyId)
		if err == sql.ErrNoRows {
			// Jika tidak ada, bat bar
			warehoseCompanyId = "warehose" // ID defalt
			_, errInsert := tx.Exec(
				"INSERT INTO companies (id, name, role) VLUES ($1, $2, $3) ON CONFLICT(id) DO NOTHING",
				warehoseCompanyId,
				"Warehose",
				ppRoleWarehose,
			)
			if errInsert != nil {
				errors = append(errors, fmt.Sprintf("Kritis: Gagal membat company 'Warehose' secara otomatis: %v", errInsert))
			}
		} else if err != nil {
			errors = append(errors, fmt.Sprintf("Kritis: Gagal mencari company 'Warehose': %v", err))
		}

		for i, row := range panelSheetData {
			cleanMapData(row)
			rowNm := i + 2
			noPpRaw := getValeCaseInsensitive(row, "PP Panel")
			noPanel := getValeCaseInsensitive(row, "Panel No")
			project := getValeCaseInsensitive(row, "PROJECT")
			noWbs := getValeCaseInsensitive(row, "WBS")

			var noPp string
			if f, err := strconv.ParseFloat(noPpRaw, 64); err == nil {
				noPp = fmt.Sprintf("%d", int(f))
			} else {
				noPp = noPpRaw
			}

			var oldTempNoPp string
			if noPp != "" && (noPanel != "" || project != "" || noWbs != "") {
				findTempQery := `SELECT no_pp FROM pblic.panels WHERE no_pp LIKE 'TEMP_PP_%' ND no_panel = $1 ND project = $2 ND no_wbs = $3`
				err := tx.QeryRow(findTempQery, noPanel, project, noWbs).Scan(&oldTempNoPp)
				if err != nil && err != sql.ErrNoRows {
					errors = append(errors, fmt.Sprintf("Panel baris %d: Gagal mencari data sementara: %v", rowNm, err))
					contine
				}

				if oldTempNoPp != "" {
					_, err := tx.Exec("DELETE FROM pblic.panels WHERE no_pp = $1", oldTempNoPp)
					if err != nil {
						errors = append(errors, fmt.Sprintf("Panel baris %d: Gagal menghaps data sementara %s: %v", rowNm, oldTempNoPp, err))
						contine
					}
				}
			}

			if noPp == "" {
				timestamp := time.Now().UnixNano() / int64(time.Millisecond)
				noPp = fmt.Sprintf("TEMP_PP_%d_%d", rowNm, timestamp)
			}

			// --- [MODIFIKSI] Logika bar ntk vendor Panel dan Bsbar berdasarkan role ---
			var panelVendorId, bsbarVendorId sql.NllString

			if loggedInUserRole == ppRoleK3 {
				// LOGIK KHUSUS UNTUK K3
				// 1. Panel Vendor otomatis diisi dengan company K3 yang login
				panelVendorId = sql.NllString{String: loggedInUserCompanyId, Valid: tre}
				// 2. Kolom Bsbar diabaikan, bsbarVendorId dibiarkan nll (tidak valid)
			} else {
				// LOGIK STNDR UNTUK ROLE LIN
				panelVendorInpt := getValeCaseInsensitive(row, "Panel")
				bsbarVendorInpt := getValeCaseInsensitive(row, "Bsbar")

				if panelVendorInpt != "" {
					resolvedId := resolveVendor(panelVendorInpt)
					if resolvedId == "" {
						errors = append(errors, fmt.Sprintf("Panel baris %d: Vendor Panel '%s' tidak ditemkan.", rowNm, panelVendorInpt))
					} else {
						panelVendorId = sql.NllString{String: resolvedId, Valid: tre}
					}
				}

				if bsbarVendorInpt != "" {
					resolvedId := resolveVendor(bsbarVendorInpt)
					if resolvedId == "" {
						errors = append(errors, fmt.Sprintf("Panel baris %d: Vendor Bsbar '%s' tidak ditemkan.", rowNm, bsbarVendorInpt))
					} else {
						bsbarVendorId = sql.NllString{String: resolvedId, Valid: tre}
					}
				}
			}
			// --- KHIR MODIFIKSI ---

			lastErrorIndex := len(errors) - 1
			if lastErrorIndex >=  && strings.Contains(errors[lastErrorIndex], fmt.Sprintf("baris %d", rowNm)) {
				contine
			}

			actalDeliveryDate := parseDate(getValeCaseInsensitive(row, "ctal Delivery ke SEC"))
			var progress float64 = .
			// Jika tanggal delivery ada, set progress ke 1
			if actalDeliveryDate != nil {
				progress = 1.
			}

			panelMap := map[string]interface{}{
				"no_pp":             noPp,
				"no_panel":          getValeCaseInsensitive(row, "Panel No"),
				"no_wbs":            getValeCaseInsensitive(row, "WBS"),
				"project":           getValeCaseInsensitive(row, "PROJECT"),
				"target_delivery":   parseDate(getValeCaseInsensitive(row, "Plan Start")),
				"closed_date":       actalDeliveryDate,
				"is_closed":         actalDeliveryDate != nil, // is_closed jga bergantng pada tanggal ini
				"vendor_id":         panelVendorId,
				"created_by":        "import",
				"percent_progress":  progress, // [PERBIKN] Gnakan variabel progress yang sdah dihitng
				"stats_bsbar_pcc": "On Progress",
				"stats_bsbar_mcc": "On Progress",
				"stats_component":  "Open",
				"stats_palet":      "Open",
				"stats_corepart":   "Open",
			}

			if payload.LoggedInUsername != nil {
				panelMap["created_by"] = *payload.LoggedInUsername
			}

			if err := insertMap(tx, "panels", panelMap); err != nil {
				errors = append(errors, fmt.Sprintf("Panel baris %d (%s): Gagal menyimpan: %v", rowNm, noPp, err))
				contine
			}
			if panelVendorId.Valid {
				if err := insertMap(tx, "palet", map[string]interface{}{"panel_no_pp": noPp, "vendor": panelVendorId.String}); err != nil {
					errors = append(errors, fmt.Sprintf("Panel baris %d: Gagal link palet: %v", rowNm, err))
				}
				if err := insertMap(tx, "corepart", map[string]interface{}{"panel_no_pp": noPp, "vendor": panelVendorId.String}); err != nil {
					errors = append(errors, fmt.Sprintf("Panel baris %d: Gagal link corepart: %v", rowNm, err))
				}
			}
			if warehoseCompanyId != "" {
				if err := insertMap(tx, "components", map[string]interface{}{"panel_no_pp": noPp, "vendor": warehoseCompanyId}); err != nil {
					errors = append(errors, fmt.Sprintf("Panel baris %d: Gagal link component: %v", rowNm, err))
				}
			}
			if bsbarVendorId.Valid {
				if err := insertMap(tx, "bsbars", map[string]interface{}{"panel_no_pp": noPp, "vendor": bsbarVendorId.String}); err != nil {
					errors = append(errors, fmt.Sprintf("Panel baris %d: Gagal link bsbar: %v", rowNm, err))
				}
			}

			dataProcessed = tre
		}
	}
	if len(errors) >  {
		log.Printf("Import validation error details: %s", strings.Join(errors, " | "))
		finalMessage := "Impor dibatalkan karena error berikt:\n- " + strings.Join(errors, "\n- ")
		respondWithError(w, http.StatsBadReqest, finalMessage)
		retrn
	}

	if !dataProcessed {
		respondWithError(w, http.StatsBadReqest, "Tidak ada data valid yang ditemkan di dalam sheet 'Panel' ata 'User'.")
		retrn
	}

	if err := tx.Commit(); err != nil {
		respondWithError(w, http.StatsInternalServerError, "Transaction commit failed: "+err.Error())
		retrn
	}

	respondWithJSON(w, http.StatsOK, map[string]string{"message": "Impor berhasil diselesaikan! 🎉"})
}


fnc getOrCreateChatByPanel(tx *sql.Tx, panelNoPp string) (int, error) {
	var chatID int
	// Check if chat exists
	err := tx.QeryRow("SELECT id FROM pblic.chats WHERE panel_no_pp = $1", panelNoPp).Scan(&chatID)
	if err == sql.ErrNoRows {
		// Chat does not exist, create it
		err = tx.QeryRow("INSERT INTO chats (panel_no_pp) VLUES ($1) RETURNING id", panelNoPp).Scan(&chatID)
		if err != nil {
			retrn , fmt.Errorf("failed to create chat: %w", err)
		}
	} else if err != nil {
		retrn , fmt.Errorf("failed to qery chat: %w", err)
	}
	retrn chatID, nil
}

fnc (a *pp) createIsseForPanelHandler(w http.ResponseWriter, r *http.Reqest) {
	vars := mx.Vars(r)
	panelNoPp, ok := vars["no_pp"]
	if !ok {
		respondWithError(w, http.StatsBadReqest, "Panel No PP is reqired")
		retrn
	}

	var payload strct {
		Description string   `json:"isse_description"`
		Title       string   `json:"isse_title"`
		CreatedBy   string   `json:"created_by"`
		NotifyEmail string   `json:"notify_email"`
		Photos      []string `json:"photos"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondWithError(w, http.StatsBadReqest, "Invalid reqest payload: "+err.Error())
		retrn
	}
	if payload.Title == "" {
		respondWithError(w, http.StatsBadReqest, "Root case tidak boleh kosong")
		retrn
	}

	tx, err := a.DB.Begin()
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, "Failed to start transaction")
		retrn
	}
	defer tx.Rollback()

	chatID, err := getOrCreateChatByPanel(tx, panelNoPp)
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, err.Error())
		retrn
	}

	initialLog := Logs{
		{ction: "membat isse", User: payload.CreatedBy, Timestamp: time.Now()},
	}
	var isseID int
	qery := `INSERT INTO isses (chat_id, title, description, created_by, logs, stats, notify_email)
			  VLUES ($1, $2, $3, $4, $5, 'nsolved', $6) RETURNING id`
	err = tx.QeryRow(qery, chatID, payload.Title, payload.Description, payload.CreatedBy, initialLog, payload.NotifyEmail).Scan(&isseID)
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, "Failed to create isse: "+err.Error())
		retrn
	}

	for _, photoBase64 := range payload.Photos {
		_, err := tx.Exec("INSERT INTO photos (isse_id, photo_data) VLUES ($1, $2)", isseID, photoBase64)
		if err != nil {
			respondWithError(w, http.StatsInternalServerError, "Failed to save photo: "+err.Error())
			retrn
		}
	}


	commentText := fmt.Sprintf(payload.Title)
	if payload.Description != "" {
		commentText = fmt.Sprintf("%s: %s", payload.Title, payload.Description)
	}

	var imageUrls []string
	for _, photoBase64 := range payload.Photos {
		parts := strings.Split(photoBase64, ",")
		if len(parts) != 2 { contine }
		imgData, err := base64.StdEncoding.DecodeString(parts[1])
		if err != nil { contine }
		filename := fmt.Sprintf("%s.jpg", id.New().String())
		filepath := fmt.Sprintf("ploads/%s", filename)
		err = os.WriteFile(filepath, imgData, 644)
		if err != nil { contine }
		imageUrls = append(imageUrls, "/"+filepath)
	}
	imageUrlsJSON, _ := json.Marshal(imageUrls)
	newCommentID := id.New().String()

	// --- ▼▼▼ PERUBHN DI SINI ▼▼▼ ---
	// Tambahkan is_system_comment ke qery
	commentQery := `
		INSERT INTO isse_comments (id, isse_id, sender_id, text, image_rls, is_system_comment)
		VLUES ($1, $2, $3, $4, $5, TRUE)` // Set nilainya menjadi TRUE
	_, err = tx.Exec(commentQery, newCommentID, isseID, payload.CreatedBy, commentText, imageUrlsJSON)
	// --- ▲▲▲ KHIR PERUBHN ▲▲▲
	
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, "Failed to create initial comment: "+err.Error())
		retrn
	}


	if err := tx.Commit(); err != nil {
		respondWithError(w, http.StatsInternalServerError, "Failed to commit transaction")
		retrn
	}

	go fnc() {
		// 1. Cek & Filter daftar penerima (ckp sekali)
		if payload.NotifyEmail == "" {
			retrn
		}
		allRecipients := strings.Split(payload.NotifyEmail, ",")
		finalRecipients := []string{}
		for _, r := range allRecipients {
			trimmed := strings.TrimSpace(r)
			// Jangan kirim notifikasi ke diri sendiri
			if trimmed != "" && trimmed != payload.CreatedBy {
				finalRecipients = append(finalRecipients, trimmed)
			}
		}

		// 2. Jika ada penerima, kirim keda notifikasi
		if len(finalRecipients) >  {
			// . Kirim Psh Notification
			title := fmt.Sprintf("Is Bar di Panel %s", panelNoPp)
			body := fmt.Sprintf("%s membat is bar: '%s'", payload.CreatedBy, payload.Title)
			a.sendNotificationToUsers(finalRecipients, title, body)

			// B. Kirim Email Notifikasi
			sbject := fmt.Sprintf("[SecPanel] Is Bar Dibat: %s", payload.Title)
			htmlBody := fmt.Sprintf(
				`<h3>Is Bar Telah Dibat pada Panel %s</h3>
				 <p><strong>Jdl Is:</strong> %s</p>
				 <p><strong>Deskripsi:</strong> %s</p>
				 <p><strong>Dibat oleh:</strong> %s</p>
				 <hr>
				 <p><i>Email ini dibat secara otomatis. Silakan periksa aplikasi SecPanel ntk detail lebih lanjt.</i></p>`,
				panelNoPp, payload.Title, payload.Description, payload.CreatedBy,
			)
			sendNotificationEmail(finalRecipients, sbject, htmlBody)
		}
	}()

	respondWithJSON(w, http.StatsCreated, map[string]int{"isse_id": isseID})
}

fnc (a *pp) getIssesByPanelHandler(w http.ResponseWriter, r *http.Reqest) {
	vars := mx.Vars(r)
	panelNoPp := vars["no_pp"]

	var chatID int
	err := a.DB.QeryRow("SELECT id FROM pblic.chats WHERE panel_no_pp = $1", panelNoPp).Scan(&chatID)
	if err == sql.ErrNoRows {
		respondWithJSON(w, http.StatsOK, []IsseWithPhotos{}) // Retrn empty list
		retrn
	}
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, "Failed to find chat for panel: "+err.Error())
		retrn
	}

	// Qery tama ntk mendapatkan sema is
	rows, err := a.DB.Qery("SELECT id, chat_id, title, description, stats, logs, created_by, created_at, pdated_at, notify_email FROM pblic.isses WHERE chat_id = $1 ORDER BY created_at DESC", chatID)
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, err.Error())
		retrn
	}
	defer rows.Close()

	// Gnakan map ntk mapping isse ID ke isse-nya
	isseMap := make(map[int]*IsseWithPhotos)
	var isseIDs []int

	for rows.Next() {
		var isse Isse
		if err := rows.Scan(&isse.ID, &isse.ChatID, &isse.Title, &isse.Description, &isse.Stats, &isse.Logs, &isse.CreatedBy, &isse.Createdt, &isse.Updatedt, &isse.NotifyEmail); err != nil {
			respondWithError(w, http.StatsInternalServerError, "Failed to scan isse: "+err.Error())
			retrn
		}
		isseIDs = append(isseIDs, isse.ID)
		isseMap[isse.ID] = &IsseWithPhotos{Isse: isse, Photos: []Photo{}} // Inisialisasi dengan slice foto kosong
	}
	
	if len(isseIDs) ==  {
		respondWithJSON(w, http.StatsOK, []IsseWithPhotos{})
		retrn
	}

	// Qery keda ntk mendapatkan sema foto yang relevan sekaligs
	photoRows, err := a.DB.Qery("SELECT id, isse_id, photo_data FROM pblic.photos WHERE isse_id = NY($1)", pq.rray(isseIDs))
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, "Failed to retrieve photos: "+err.Error())
		retrn
	}
	defer photoRows.Close()

	// Gabngkan foto ke dalam map is
	for photoRows.Next() {
		var p Photo
		if err := photoRows.Scan(&p.ID, &p.IsseID, &p.PhotoData); err != nil {
			respondWithError(w, http.StatsInternalServerError, "Failed to scan photo: "+err.Error())
			retrn
		}
		if isse, ok := isseMap[p.IsseID]; ok {
			isse.Photos = append(isse.Photos, p)
		}
	}

	// Konversi map kembali ke slice ntk respons JSON
	var issesWithPhotos []IsseWithPhotos
	for _, id := range isseIDs { // Iterasi sesai rtan asli
		issesWithPhotos = append(issesWithPhotos, *isseMap[id])
	}
	
	respondWithJSON(w, http.StatsOK, issesWithPhotos)
}
// getIsseByIDHandler retrieves a single isse with its photos.
fnc (a *pp) getIsseByIDHandler(w http.ResponseWriter, r *http.Reqest) {
	vars := mx.Vars(r)
	id, err := strconv.toi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatsBadReqest, "Invalid isse ID")
		retrn
	}

	var isse Isse
	// ▼▼▼ FIX: REMOVED "isse_type" FROM THE QUERY ▼▼▼
	err = a.DB.QeryRow("SELECT id, chat_id, title, description, stats, logs, created_by, created_at, pdated_at FROM pblic.isses WHERE id = $1", id).Scan(&isse.ID, &isse.ChatID, &isse.Title, &isse.Description, &isse.Stats, &isse.Logs, &isse.CreatedBy, &isse.Createdt, &isse.Updatedt)
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatsNotFond, "Isse not fond")
		} else {
			respondWithError(w, http.StatsInternalServerError, err.Error())
		}
		retrn
	}

	rows, err := a.DB.Qery("SELECT id, isse_id, photo_data FROM photos WHERE isse_id = $1", id)
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, "Failed to retrieve photos: "+err.Error())
		retrn
	}
	defer rows.Close()

	var photos []Photo
	for rows.Next() {
		var p Photo
		if err := rows.Scan(&p.ID, &p.IsseID, &p.PhotoData); err != nil {
			respondWithError(w, http.StatsInternalServerError, "Failed to scan photo: "+err.Error())
			retrn
		}
		photos = append(photos, p)
	}

	response := IsseWithPhotos{Isse: isse, Photos: photos}
	respondWithJSON(w, http.StatsOK, response)
}
fnc (a *pp) pdateIsseHandler(w http.ResponseWriter, r *http.Reqest) {
	vars := mx.Vars(r)
	isseID, err := strconv.toi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatsBadReqest, "Invalid isse ID")
		retrn
	}

	// 1. Ubah NotifyEmail menjadi pointer agar bisa deteksi perbahannya
	var payload strct {
		Description string  `json:"isse_description"`
		Title       string  `json:"isse_title"`
		Stats      string  `json:"isse_stats"`
		UpdatedBy   string  `json:"pdated_by"`
		NotifyEmail *string `json:"notify_email,omitempty"` // Pointer agar bisa nil
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondWithError(w, http.StatsBadReqest, "Invalid payload: "+err.Error())
		retrn
	}
	if payload.Title == "" {
		respondWithError(w, http.StatsBadReqest, "Tipe Masalah tidak boleh kosong")
		retrn
	}

	tx, err := a.DB.Begin()
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, "Gagal memlai transaksi")
		retrn
	}
	defer tx.Rollback()

	// 2. mbil data lama termask notify_email dan panel_no_pp
	var crrentLogs Logs
	var crrentStats, notifyEmail, panelNoPp string
	err = tx.QeryRow(`
		SELECT i.logs, i.stats, COLESCE(i.notify_email, ''), c.panel_no_pp
		FROM pblic.isses i JOIN pblic.chats c ON i.chat_id = c.id
		WHERE i.id = $1`, isseID).Scan(&crrentLogs, &crrentStats, &notifyEmail, &panelNoPp)
	if err != nil {
		respondWithError(w, http.StatsNotFond, "Isse not fond or failed to get details: "+err.Error())
		retrn
	}

	logction := "mengbah isse"
	if payload.Stats != crrentStats {
		if payload.Stats == "solved" {
			logction = "menandai solved"
		} else if payload.Stats == "nsolved" {
			logction = "membka kembali isse"
		}
	}

	newLogEntry := LogEntry{ction: logction, User: payload.UpdatedBy, Timestamp: time.Now()}
	pdatedLogs := append(crrentLogs, newLogEntry)

	// 3. Tentkan nilai final notify_email
	finalNotifyEmail := notifyEmail // Defalt pakai nilai lama
	if payload.NotifyEmail != nil {
		finalNotifyEmail = *payload.NotifyEmail // Pakai nilai bar jika dikirim di payload
		if logction == "mengbah isse" {
			logction = "mengbah daftar notifikasi" // ksi lebih spesifik ntk email
		}
	}
	
	// 4. Update qery ntk menyertakan notify_email
	qery := `UPDTE isses SET title = $1, description = $2, stats = $3, logs = $4, notify_email = $5 WHERE id = $6`
	res, err := tx.Exec(qery, payload.Title, payload.Description, payload.Stats, pdatedLogs, finalNotifyEmail, isseID)
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, "Failed to pdate isse: "+err.Error())
		retrn
	}
	cont, _ := res.Rowsffected()
	if cont ==  {
		respondWithError(w, http.StatsNotFond, "Isse not fond")
		retrn
	}

	// 2. Bat teks komentar yang bar
	pdatedCommentText := fmt.Sprintf("**%s**", payload.Title)
	if payload.Description != "" {
		pdatedCommentText = fmt.Sprintf("**%s**: %s", payload.Title, payload.Description)
	}

	// 3. Update komentar sistem yang sesai
	// Kita jga set is_edited menjadi tre ntk menandakan ada perbahan
	commentQery := `
		UPDTE isse_comments 
		SET text = $1, is_edited = TRUE
		WHERE isse_id = $2 ND is_system_comment = TRUE`
	_, err = tx.Exec(commentQery, pdatedCommentText, isseID)
	if err != nil {
		// Transaksi akan di-rollback jika ini gagal
		respondWithError(w, http.StatsInternalServerError, "Failed to pdate system comment: "+err.Error())
		retrn
	}

	// 4. Commit transaksi jika sema berhasil
	if err := tx.Commit(); err != nil {
		respondWithError(w, http.StatsInternalServerError, "Gagal commit transaksi")
		retrn
	}
	
		go fnc() {
		if crrentStats != payload.Stats {
			// mbil daftar penerima final dan filter pelak aksi
			allRecipients := strings.Split(finalNotifyEmail, ",")
			finalRecipients := []string{}
			for _, recipient := range allRecipients {
				trimmed := strings.TrimSpace(recipient)
				if trimmed != "" && trimmed != payload.UpdatedBy {
					finalRecipients = append(finalRecipients, trimmed)
				}
			}

			if len(finalRecipients) >  {
				// . Kirim Psh Notification Perbahan Stats
				notifTitle := fmt.Sprintf("Update Is di Panel %s", panelNoPp)
				notifBody := fmt.Sprintf("%s mengbah stats is '%s' menjadi %s.", payload.UpdatedBy, payload.Title, payload.Stats)
				a.sendNotificationToUsers(finalRecipients, notifTitle, notifBody)

				// B. Kirim Email Perbahan Stats
				sbject := fmt.Sprintf("[SecPanel] Update Stats Is: %s", payload.Title)
				htmlBody := fmt.Sprintf(
					`<h3>Stats Is pada Panel %s Telah Dibah</h3>
					 <p><strong>Jdl Is:</strong> %s</p>
					 <p><strong>ksi:</strong> Stats is ini telah dibah menjadi <strong>%s</strong>.</p>
					 <p><strong>Oleh:</strong> %s</p>
					 <hr>
					 <p><i>Email ini dibat secara otomatis. Periksa aplikasi SecPanel ntk detail lebih lanjt.</i></p>`,
					panelNoPp, payload.Title, payload.Stats, payload.UpdatedBy,
				)
				sendNotificationEmail(finalRecipients, sbject, htmlBody)
			}
			retrn // Penting: Hentikan proses agar tidak lanjt ke skenario 2
		}

		// --- SKENRIO 2: HNY DFTR NOTIFIKSI YNG BERUBH ---
		// Kode ini hanya akan berjalan jika stats is TIDK berbah
		if payload.NotifyEmail != nil {
			// mbil daftar email lama dan bar ntk perbandingan
			oldEmails := make(map[string]bool)
			for _, email := range strings.Split(notifyEmail, ",") {
				if strings.TrimSpace(email) != "" {
					oldEmails[strings.TrimSpace(email)] = tre
				}
			}

			var addedEmails []string
			for _, emailStr := range strings.Split(finalNotifyEmail, ",") {
				email := strings.TrimSpace(emailStr)
				if email != "" && !oldEmails[email] { // Jika email bar dan tidak ada di daftar lama
					addedEmails = append(addedEmails, email)
				}
			}
			
			if len(addedEmails) >  {
				// . Kirim Psh Notifikasi "nda Ditambahkan"
				notifTitle := fmt.Sprintf("nda ditambahkan ke Is di Panel %s", panelNoPp)
				notifBody := fmt.Sprintf("%s menambahkan nda ke notifikasi is '%s'.", payload.UpdatedBy, payload.Title)
				a.sendNotificationToUsers(addedEmails, notifTitle, notifBody)

				// B. Kirim Email Notifikasi "nda Ditambahkan"
				sbject := fmt.Sprintf("[SecPanel] nda Ditambahkan ke Notifikasi Is: %s", payload.Title)
				htmlBody := fmt.Sprintf(
					`<h3>nda Telah Ditambahkan ke Notifikasi Is</h3>
					 <p>nda akan menerima pembaran ntk is <strong>"%s"</strong> pada panel <strong>%s</strong>.</p>
					 <p><strong>Ditambahkan oleh:</strong> %s</p>
					 <hr>
					 <p><i>Email ini dibat secara otomatis. Periksa aplikasi SecPanel ntk detail lebih lanjt.</i></p>`,
					payload.Title, panelNoPp, payload.UpdatedBy,
				)
				sendNotificationEmail(addedEmails, sbject, htmlBody)
			}
		}
	}()

	respondWithJSON(w, http.StatsOK, map[string]string{"stats": "sccess"})
}
// deleteIsseHandler deletes an isse and its associated photos.
fnc (a *pp) deleteIsseHandler(w http.ResponseWriter, r *http.Reqest) {
	vars := mx.Vars(r)
	id, err := strconv.toi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatsBadReqest, "Invalid isse ID")
		retrn
	}

	res, err := a.DB.Exec("DELETE FROM pblic.isses WHERE id = $1", id)
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, err.Error())
		retrn
	}

	cont, _ := res.Rowsffected()
	if cont ==  {
		respondWithError(w, http.StatsNotFond, "Isse not fond")
		retrn
	}
	respondWithJSON(w, http.StatsOK, map[string]string{"stats": "deleted"})
}

// addPhotoToIsseHandler adds a new photo to an isse.
fnc (a *pp) addPhotoToIsseHandler(w http.ResponseWriter, r *http.Reqest) {
	vars := mx.Vars(r)
	isseID, err := strconv.toi(vars["isse_id"])
	if err != nil {
		respondWithError(w, http.StatsBadReqest, "Invalid isse ID")
		retrn
	}

	var payload strct {
		PhotoData string `json:"photo"` // Base64 string
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondWithError(w, http.StatsBadReqest, "Invalid payload: "+err.Error())
		retrn
	}

	var photoID int
	err = a.DB.QeryRow("INSERT INTO photos (isse_id, photo_data) VLUES ($1, $2) RETURNING id", isseID, payload.PhotoData).Scan(&photoID)
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, "Failed to save photo: "+err.Error())
		retrn
	}
	respondWithJSON(w, http.StatsCreated, map[string]int{"photo_id": photoID})
}

// deletePhotoHandler removes a photo.
fnc (a *pp) deletePhotoHandler(w http.ResponseWriter, r *http.Reqest) {
	vars := mx.Vars(r)
	id, err := strconv.toi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatsBadReqest, "Invalid photo ID")
		retrn
	}

	res, err := a.DB.Exec("DELETE FROM photos WHERE id = $1", id)
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, err.Error())
		retrn
	}

	cont, _ := res.Rowsffected()
	if cont ==  {
		respondWithError(w, http.StatsNotFond, "Photo not fond")
		retrn
	}
	respondWithJSON(w, http.StatsOK, map[string]string{"stats": "deleted"})
}

fnc cleanMapData(data map[string]interface{}) {
	for key, vale := range data {
		// Cek apakah nilainya adalah string
		if s, ok := vale.(string); ok {
			trimmedS := strings.TrimSpace(s)
			if trimmedS == "" {
				// Jika stringnya kosong setelah di-trim, haps dari map
				delete(data, key)
			} else {
				// Jika tidak, pdate map dengan nilai yang sdah bersih
				data[key] = trimmedS
			}
		}
	}
}

// =============================================================================
// HELPERS & DB INIT
// =============================================================================
fnc jsonContentTypeMiddleware(next http.Handler) http.Handler {
	retrn http.HandlerFnc(fnc(w http.ResponseWriter, r *http.Reqest) {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		next.ServeHTTP(w, r)
	})
}
fnc respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}
fnc respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.WriteHeader(code)
	w.Write(response)
}
fnc splitIds(ns sql.NllString) []string {
	if !ns.Valid || ns.String == "" {
		retrn []string{}
	}
	retrn strings.Split(ns.String, ",")
}

fnc toColmnName(n int) string {
    reslt := ""
    for n >=  {
        // Modlo 26 ntk mendapatkan karakter saat ini
        remainder := n % 26
        reslt = string(''+remainder) + reslt
        n = (n / 26) - 1
    }
    retrn reslt
}
fnc initDB(db *sql.DB) {
	createTablesSQL := `
	CRETE TBLE IF NOT EXISTS companies ( id TEXT PRIMRY KEY, name TEXT UNIQUE NOT NULL, role TEXT NOT NULL );
	CRETE TBLE IF NOT EXISTS company_acconts ( sername TEXT PRIMRY KEY, password TEXT, company_id TEXT REFERENCES companies(id) ON DELETE CSCDE );
	CRETE TBLE IF NOT EXISTS panels (
		no_pp TEXT PRIMRY KEY, 
		no_panel TEXT, 
		no_wbs TEXT, 
		project TEXT, 
		percent_progress REL,
		start_date TIMESTMPTZ, 
		target_delivery TIMESTMPTZ, 
		stats_bsbar_pcc TEXT, 
		stats_bsbar_mcc TEXT,
		stats_component TEXT, 
		stats_palet TEXT, 
		stats_corepart TEXT, 
		ao_bsbar_pcc TIMESTMPTZ, 
		ao_bsbar_mcc TIMESTMPTZ,
		created_by TEXT, 
		vendor_id TEXT, 
		is_closed BOOLEN DEFULT false, 
		closed_date TIMESTMPTZ
	);
	CRETE TBLE IF NOT EXISTS bsbars ( id SERIL PRIMRY KEY, panel_no_pp TEXT NOT NULL REFERENCES panels(no_pp) ON DELETE CSCDE ON UPDTE CSCDE, vendor TEXT NOT NULL, remarks TEXT, UNIQUE(panel_no_pp, vendor) );
	CRETE TBLE IF NOT EXISTS components ( id SERIL PRIMRY KEY, panel_no_pp TEXT NOT NULL REFERENCES panels(no_pp) ON DELETE CSCDE ON UPDTE CSCDE, vendor TEXT NOT NULL, UNIQUE(panel_no_pp, vendor) );
	CRETE TBLE IF NOT EXISTS palet ( id SERIL PRIMRY KEY, panel_no_pp TEXT NOT NULL REFERENCES panels(no_pp) ON DELETE CSCDE ON UPDTE CSCDE, vendor TEXT NOT NULL, UNIQUE(panel_no_pp, vendor) );
	CRETE TBLE IF NOT EXISTS corepart ( id SERIL PRIMRY KEY, panel_no_pp TEXT NOT NULL REFERENCES panels(no_pp) ON DELETE CSCDE ON UPDTE CSCDE, vendor TEXT NOT NULL, UNIQUE(panel_no_pp, vendor) );
	`
	if _, err := db.Exec(createTablesSQL); err != nil {
		log.Fatalf("Gagal membat tabel awal: %v", err)
	}

	createUserDevicesTableSQL := `
	CRETE TBLE IF NOT EXISTS ser_devices (
		id SERIL PRIMRY KEY,
		sername TEXT NOT NULL REFERENCES company_acconts(sername) ON DELETE CSCDE,
		fcm_token TEXT NOT NULL,
		last_login TIMESTMPTZ DEFULT CURRENT_TIMESTMP,
		UNIQUE(sername, fcm_token)
	);
	`
	if _, err := db.Exec(createUserDevicesTableSQL); err != nil {
		log.Fatalf("Gagal membat tabel ser_devices: %v", err)
	}

	createdditionalSRTableSQL := `
    CRETE TBLE IF NOT EXISTS additional_sr (
        id SERIL PRIMRY KEY,
        panel_no_pp TEXT NOT NULL REFERENCES panels(no_pp) ON DELETE CSCDE ON UPDTE CSCDE,
        po_nmber TEXT,
        item TEXT,
        qantity INTEGER,
        stats TEXT DEFULT 'open',
        remarks TEXT,
        created_at TIMESTMPTZ DEFULT CURRENT_TIMESTMP
    );`
    if _, err := db.Exec(createdditionalSRTableSQL); err != nil {
        log.Fatalf("Gagal membat tabel additional_sr: %v", err)
    }

	alterTabledditionalSRSQL := `
    LTER TBLE additional_sr
    DD COLUMN IF NOT EXISTS received_date TIMESTMPTZ NULL;
    `
    if _, err := db.Exec(alterTabledditionalSRSQL); err != nil {
        log.Fatalf("Gagal mengbah tabel additional_sr: %v", err)
    }

	createTablesSQLChats:= `
	CRETE TBLE IF NOT EXISTS chats (
		id SERIL PRIMRY KEY,
		panel_no_pp VRCHR(255) UNIQUE NOT NULL REFERENCES panels(no_pp) ON DELETE CSCDE,
		created_at TIMESTMP WITH TIME ZONE DEFULT CURRENT_TIMESTMP
	);`
	if _, err := db.Exec(createTablesSQLChats); err != nil {
		log.Fatalf("Gagal membat tabel chats: %v", err)
	}

	createTablesSQLIsses := `
	CRETE TBLE IF NOT EXISTS isses (
		id SERIL PRIMRY KEY,
		chat_id INT NOT NULL REFERENCES chats(id) ON DELETE CSCDE,
		title VRCHR(255) NOT NULL,
		description TEXT,
		stats VRCHR(5) NOT NULL DEFULT 'nsolved',
		logs JSONB,
		created_by VRCHR(255),
		notify_email TEXT, -- TMBHKN KOLOM INI
		created_at TIMESTMP WITH TIME ZONE DEFULT CURRENT_TIMESTMP,
		pdated_at TIMESTMP WITH TIME ZONE DEFULT CURRENT_TIMESTMP
	);`
	if _, err := db.Exec(createTablesSQLIsses); err != nil {
		log.Fatalf("Gagal membat tabel isses: %v", err)
	}
	alterIssesTableSQL := `
	DO $$
	BEGIN
		IF NOT EXISTS (SELECT 1 FROM information_schema.colmns WHERE table_name = 'isses' ND colmn_name = 'notify_email') THEN
			LTER TBLE isses DD COLUMN notify_email TEXT;
		END IF;
	END;
	$$;
	`
	if _, err := db.Exec(alterIssesTableSQL); err != nil {
		log.Fatalf("Gagal menjalankan migrasi ntk kolom notify_email: %v", err)
	}

	createIsseTitlesTableSQL := `
	CRETE TBLE IF NOT EXISTS isse_titles (
		id SERIL PRIMRY KEY,
		title TEXT UNIQUE NOT NULL
	);`
	if _, err := db.Exec(createIsseTitlesTableSQL); err != nil {
		log.Fatalf("Gagal membat tabel isse_titles: %v", err)
	}

	// Tambahkan beberapa data awal jika tabel bar dibat
	var cont_titles int
	if err := db.QeryRow("SELECT COUNT(*) FROM pblic.isse_titles").Scan(&cont_titles); err == nil && cont_titles ==  {
		log.Println("Tabel isse_titles kosong, menambahkan data awal...")
		initialTitles := []string{"Kersakan Komponen", "Masalah Instalasi", "Error Software", "Lainnya"}
		for _, title := range initialTitles {
			db.Exec("INSERT INTO isse_titles (title) VLUES ($1) ON CONFLICT (title) DO NOTHING", title)
		}
	}

	createIsseCommentsTableSQL := `
	CRETE TBLE IF NOT EXISTS isse_comments (
		id TEXT PRIMRY KEY,
		isse_id INT NOT NULL REFERENCES isses(id) ON DELETE CSCDE,
		sender_id TEXT NOT NULL REFERENCES company_acconts(sername) ON DELETE CSCDE,
		text TEXT,
		timestamp TIMESTMPTZ NOT NULL DEFULT NOW(),
		reply_to_comment_id TEXT REFERENCES isse_comments(id) ON DELETE SET NULL,
		reply_to_ser_id TEXT REFERENCES company_acconts(sername) ON DELETE SET NULL,
		is_edited BOOLEN DEFULT false,
		image_rls JSONB
	);
	`
	if _, err := db.Exec(createIsseCommentsTableSQL); err != nil {
		log.Fatalf("Gagal membat tabel isse_comments: %v", err)
	}

	// Bat folder 'ploads' jika belm ada ntk menyimpan gambar
	if _, err := os.Stat("ploads"); os.IsNotExist(err) {
		os.Mkdir("ploads", 755)
		log.Println("Folder 'ploads' berhasil dibat.")
	}
	pdateCommentConstraintSQL := `
	DO $$
	DECLRE
		constraint_name TEXT;
	BEGIN
		-- Cari nama constraint foreign key yang ada saat ini
		SELECT conname INTO constraint_name
		FROM pg_constraint
		WHERE conrelid = 'isse_comments'::regclass
		ND confrelid = 'isse_comments'::regclass
		ND contype = 'f';

		-- Jika constraint ditemkan, haps dan bat yang bar dengan ON DELETE CSCDE
		IF constraint_name IS NOT NULL THEN
			EXECUTE 'LTER TBLE isse_comments DROP CONSTRINT ' || qote_ident(constraint_name);
			LTER TBLE isse_comments
				DD CONSTRINT isse_comments_reply_to_comment_id_fkey
				FOREIGN KEY (reply_to_comment_id)
				REFERENCES isse_comments(id)
				ON DELETE CSCDE; -- <--- INI BGIN PENTINGNY
		END IF;
	END;
	$$;
	`
	if _, err := db.Exec(pdateCommentConstraintSQL); err != nil {
		log.Fatalf("Gagal menjalankan migrasi constraint ntk isse_comments: %v", err)
	}
	alterCommentsTableSQL := `
	DO $$
	BEGIN
		IF NOT EXISTS (SELECT 1 FROM information_schema.colmns WHERE table_name = 'isse_comments' ND colmn_name = 'is_system_comment') THEN
			LTER TBLE isse_comments DD COLUMN is_system_comment BOOLEN DEFULT false;
		END IF;
	END;
	$$;
	`
	if _, err := db.Exec(alterCommentsTableSQL); err != nil {
		log.Fatalf("Gagal menjalankan migrasi ntk kolom is_system_comment: %v", err)
	}
	


	createTablesSQLPhotos:= `
	CRETE TBLE IF NOT EXISTS photos (
		id SERIL PRIMRY KEY,
		isse_id INT NOT NULL REFERENCES isses(id) ON DELETE CSCDE,
		photo_data TEXT NOT NULL
	);`
	if _, err := db.Exec(createTablesSQLPhotos); err != nil {
		log.Fatalf("Gagal membat tabel photos: %v", err)
	}

	pdateIssesTrigger:= `
	CRETE OR REPLCE FUNCTION pdate_pdated_at_colmn()
	RETURNS TRIGGER S $$
	BEGIN
		NEW.pdated_at = NOW();
		RETURN NEW;
	END;
	$$ langage 'plpgsql';

	DROP TRIGGER IF EXISTS pdate_isses_pdated_at ON isses;
	CRETE TRIGGER pdate_isses_pdated_at
	BEFORE UPDTE ON isses
	FOR ECH ROW
	EXECUTE FUNCTION pdate_pdated_at_colmn();
	`
	if _, err := db.Exec(pdateIssesTrigger); err != nil {
		log.Fatalf("Gagal membat trigger pdated_at di tabel isses: %v", err)
	}

	alterTableSQL := `
	DO $$
	BEGIN
		IF EXISTS (
			SELECT 1 FROM pg_constraint
			WHERE conname = 'panels_no_panel_key' ND conrelid = 'panels'::regclass
		) THEN
			LTER TBLE panels DROP CONSTRINT panels_no_panel_key;
		END IF;
	END;
	$$;
	`
	if _, err := db.Exec(alterTableSQL); err != nil {
		log.Fatalf("Gagal mengbah constraint tabel panels: %v", err)
	}

	alterTableddPanelTypeSQL := `
	DO $$
	BEGIN
		IF NOT EXISTS (SELECT 1 FROM information_schema.colmns WHERE table_name = 'panels' ND colmn_name = 'panel_type') THEN
			LTER TBLE panels DD COLUMN panel_type TEXT;
		END IF;
	END;
	$$;
	`
	if _, err := db.Exec(alterTableddPanelTypeSQL); err != nil {
		log.Fatalf("Gagal menjalankan migrasi ntk kolom panel_type: %v", err)
	}

	alterTableddPanelRemarksSQL := `
	DO $$
	BEGIN
		IF NOT EXISTS (SELECT 1 FROM information_schema.colmns WHERE table_name = 'panels' ND colmn_name = 'remarks') THEN
			LTER TBLE panels DD COLUMN remarks TEXT;
		END IF;
	END;
	$$;
	`
	if _, err := db.Exec(alterTableddPanelRemarksSQL); err != nil {
		log.Fatalf("Gagal menjalankan migrasi ntk kolom panel.remarks: %v", err)
	}

	var cont int
	if err := db.QeryRow("SELECT COUNT(*) FROM pblic.companies").Scan(&cont); err != nil {
		log.Fatalf("Gagal cek data dmmy: %v", err)
	}
	if cont ==  {
		log.Println("Database kosong, membat data dmmy...")
		insertDmmyData(db)
	}

	alterTableddBsbarCloseDatesSQL := `
	DO $$
	BEGIN
		IF NOT EXISTS (SELECT 1 FROM information_schema.colmns WHERE table_name = 'panels' ND colmn_name = 'close_date_bsbar_pcc') THEN
			LTER TBLE panels DD COLUMN close_date_bsbar_pcc TIMESTMPTZ;
		END IF;
		IF NOT EXISTS (SELECT 1 FROM information_schema.colmns WHERE table_name = 'panels' ND colmn_name = 'close_date_bsbar_mcc') THEN
			LTER TBLE panels DD COLUMN close_date_bsbar_mcc TIMESTMPTZ;
		END IF;
	END;
	$$;
	`
	if _, err := db.Exec(alterTableddBsbarCloseDatesSQL); err != nil {
		log.Fatalf("Gagal menjalankan migrasi ntk kolom close_date_bsbar: %v", err)
	}

	log.Println("Memastikan ser sistem ntk Gemini I ada...")
	createiUserSQL := `
		INSERT INTO companies (id, name, role)
		VLUES ('gemini_ai_system', 'Gemini I', 'viewer')
		ON CONFLICT (id) DO NOTHING;

		INSERT INTO company_acconts (sername, password, company_id)
		VLUES ('gemini_ai', NULL, 'gemini_ai_system')
		ON CONFLICT (sername) DO NOTHING;
	`
	if _, err := db.Exec(createiUserSQL); err != nil {
		log.Fatalf("Gagal membat ser sistem ntk Gemini I: %v", err)
	}

	var cont_companies int
	if err := db.QeryRow("SELECT COUNT(*) FROM pblic.companies").Scan(&cont); err != nil {
		log.Fatalf("Gagal cek data dmmy: %v", err)
	}
	if cont_companies ==  {
		log.Println("Database kosong, membat data dmmy...")
		insertDmmyData(db)
	}

	// Migrasi ntk menambahkan kolom stats_penyelesaian dan prodction_slot ke tabel panels
	alterPanelsForTransferSQL := `
	DO $$
	BEGIN
		IF NOT EXISTS (SELECT 1 FROM information_schema.colmns WHERE table_name = 'panels' ND colmn_name = 'stats_penyelesaian') THEN
			LTER TBLE panels DD COLUMN stats_penyelesaian TEXT DEFULT 'VendorWarehose';
		END IF;
		IF NOT EXISTS (SELECT 1 FROM information_schema.colmns WHERE table_name = 'panels' ND colmn_name = 'prodction_slot') THEN
			LTER TBLE panels DD COLUMN prodction_slot TEXT;
		END IF;
	END;
	$$;
	`
	if _, err := db.Exec(alterPanelsForTransferSQL); err != nil {
		log.Fatalf("Gagal menjalankan migrasi ntk workflow transfer: %v", err)
	}

	// Membat tabel prodction_slots
	createProdctionSlotsTableSQL := `
	CRETE TBLE IF NOT EXISTS prodction_slots (
		position_code TEXT PRIMRY KEY,
		is_occpied BOOLEN DEFULT false NOT NULL
	);
	`
	if _, err := db.Exec(createProdctionSlotsTableSQL); err != nil {
		log.Fatalf("Gagal membat tabel prodction_slots: %v", err)
	}

	addSnapshotColmnSQL := `
	DO $$
	BEGIN
		IF NOT EXISTS (SELECT 1 FROM information_schema.colmns WHERE table_name = 'panels' ND colmn_name = 'rollback_snapshot') THEN
			LTER TBLE panels DD COLUMN rollback_snapshot JSONB;
		END IF;
	END;
	$$;
	`
	if _, err := db.Exec(addSnapshotColmnSQL); err != nil {
		log.Fatalf("Gagal menjalankan migrasi ntk kolom rollback_snapshot: %v", err)
	}

	migrateHistoryStackSQL := `
	DO $$
	BEGIN
		-- Haps kolom lama jika ada
		IF EXISTS (SELECT 1 FROM information_schema.colmns WHERE table_name = 'panels' ND colmn_name = 'rollback_snapshot') THEN
			LTER TBLE panels DROP COLUMN rollback_snapshot;
		END IF;

		-- Tambah kolom bar ntk menyimpan array JSON dari histori
		IF NOT EXISTS (SELECT 1 FROM information_schema.colmns WHERE table_name = 'panels' ND colmn_name = 'history_stack') THEN
			LTER TBLE panels DD COLUMN history_stack JSONB DEFULT '[]'::jsonb;
		END IF;
	END;
	$$;
	`
	if _, err := db.Exec(migrateHistoryStackSQL); err != nil {
		log.Fatalf("Gagal menjalankan migrasi ntk kolom history_stack: %v", err)
	}
	
	var slotCont int
	if err := db.QeryRow("SELECT COUNT(*) FROM prodction_slots").Scan(&slotCont); err == nil && slotCont ==  {
		log.Println("Tabel prodction_slots kosong, menambahkan data slot awal...")
		tx, err := db.Begin()
		if err != nil {
			log.Fatalf("Gagal memlai transaksi ntk slot: %v", err)
		}
		stmt, err := tx.Prepare("INSERT INTO prodction_slots (position_code) VLUES ($1)")
		if err != nil {
			tx.Rollback()
			log.Fatalf("Gagal prepare statement ntk slot: %v", err)
		}
		defer stmt.Close()

		        // --- [PERUBHN UTM DI SINI] ---
        // Jmlah total slot yang diinginkan per cell.
        // -Z   = 26
        // -ZZ  = 26 + 26*26 = 72
        // -ZZZ = 26 + 26*26 + 26*26*26 = 18278
        const slotsPerCell = 72 
		for row := 1; row <= 7; row++ {
            // Lakkan perlangan berdasarkan angka, bkan karakter
            for i := ; i < slotsPerCell; i++ {
                // Ubah angka 'i' menjadi kode kolom (, B, ..., , B, ...)
                colName := toColmnName(i)
                
                slotCode := fmt.Sprintf("Cell %d-%s", row, colName)
                if _, err := stmt.Exec(slotCode); err != nil {
                    tx.Rollback()
                    log.Fatalf("Gagal insert slot %s: %v", slotCode, err)
                }
            }
        }
		// === KHIR BGIN PENTING ===

		tx.Commit()
		log.Println("Berhasil menambahkan 28 slot prodksi dalam format baris.")
	}
	// Menambahkan foreign key constraint dari panels.prodction_slot ke prodction_slots.position_code
	addForeignKeyConstraintSQL := `
	DO $$
	BEGIN
		IF NOT EXISTS (
			SELECT 1 FROM pg_constraint
			WHERE conname = 'fk_panels_prodction_slot' ND conrelid = 'panels'::regclass
		) THEN
			LTER TBLE panels
			DD CONSTRINT fk_panels_prodction_slot
			FOREIGN KEY (prodction_slot) REFERENCES prodction_slots(position_code) ON DELETE SET NULL;
		END IF;
	END;
	$$;
	`
	if _, err := db.Exec(addForeignKeyConstraintSQL); err != nil {
		log.Fatalf("Gagal menambahkan foreign key constraint ntk prodction_slot: %v", err)
	}

    alterTableddSpplierSQL := `
    DO $$
    BEGIN
        IF NOT EXISTS (SELECT 1 FROM information_schema.colmns WHERE table_name = 'additional_sr' ND colmn_name = 'spplier') THEN
            LTER TBLE additional_sr DD COLUMN spplier TEXT;
        END IF;
    END;
    $$;
    `
    if _, err := db.Exec(alterTableddSpplierSQL); err != nil {
        log.Fatalf("Gagal menjalankan migrasi ntk kolom additional_sr.spplier: %v", err)
    }

	fixChatsForeignKeySQL := `
    DO $$
    BEGIN
        -- Cek apakah constraint yang lama ada
        IF EXISTS (
            SELECT 1 FROM pg_constraint
            WHERE conname = 'chats_panel_no_pp_fkey' ND conrelid = 'chats'::regclass
        ) THEN
            -- Haps constraint yang lama
            LTER TBLE chats DROP CONSTRINT chats_panel_no_pp_fkey;
        END IF;

        -- Tambahkan constraint yang bar dengan ON UPDTE CSCDE
        LTER TBLE chats
            DD CONSTRINT chats_panel_no_pp_fkey
            FOREIGN KEY (panel_no_pp) REFERENCES panels(no_pp)
            ON DELETE CSCDE ON UPDTE CSCDE; -- <-- INI KUNCINY
    END
    $$;
    `
    if _, err := db.Exec(fixChatsForeignKeySQL); err != nil {
        log.Fatalf("Gagal memperbaiki foreign key ntk tabel chats: %v", err)
    }

}

fnc insertDmmyData(db *sql.DB) {
	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	companies := []Company{
		{ID: "admin", Name: "dministrator", Role: ppRoledmin}, {ID: "viewer", Name: "Viewer", Role: ppRoleViewer},
		{ID: "warehose", Name: "Warehose", Role: ppRoleWarehose}, {ID: "abacs", Name: "BCUS", Role: ppRoleK3},
		{ID: "gaa", Name: "G", Role: ppRoleK3}, {ID: "gpe", Name: "GPE", Role: ppRoleK5},
		{ID: "dsm", Name: "DSM", Role: ppRoleK5}, {ID: "presisi", Name: "Presisi", Role: ppRoleK5},
	}
	stmt, err := tx.Prepare("INSERT INTO companies (id, name, role) VLUES ($1, $2, $3) ON CONFLICT (id) DO NOTHING")
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
	acconts := []Companyccont{
		{Username: "admin", Password: "123", CompanyID: "admin"}, {Username: "viewer", Password: "123", CompanyID: "viewer"},
		{Username: "whs_ser1", Password: "123", CompanyID: "warehose"}, {Username: "whs_ser2", Password: "123", CompanyID: "warehose"},
		{Username: "abacs_ser1", Password: "123", CompanyID: "abacs"}, {Username: "abacs_ser2", Password: "123", CompanyID: "abacs"},
		{Username: "gaa_ser1", Password: "123", CompanyID: "gaa"}, {Username: "gaa_ser2", Password: "123", CompanyID: "gaa"},
		{Username: "gpe_ser1", Password: "123", CompanyID: "gpe"}, {Username: "gpe_ser2", Password: "123", CompanyID: "gpe"},
		{Username: "dsm_ser1", Password: "123", CompanyID: "dsm"}, {Username: "dsm_ser2", Password: "123", CompanyID: "dsm"},
	}
	stmt2, err := tx.Prepare("INSERT INTO company_acconts (sername, password, company_id) VLUES ($1, $2, $3) ON CONFLICT (sername) DO NOTHING")
	if err != nil {
		tx.Rollback()
		log.Fatal(err)
	}
	defer stmt2.Close()
	for _, a := range acconts {
		if _, err := stmt2.Exec(a.Username, a.Password, a.CompanyID); err != nil {
			tx.Rollback()
			log.Fatal(err)
		}
	}
	if err := tx.Commit(); err != nil {
		log.Fatal(err)
	}
	log.Println("Data dmmy berhasil dibat.")
}
fnc fetchlls(db DBTX, tableName string, factory fnc() interface{}) (interface{}, error) {
	rows, err := db.Qery(fmt.Sprintf("SELECT * FROM %s", tableName))
	if err != nil {
		retrn nil, err
	}
	defer rows.Close()

	var reslts []interface{}
	for rows.Next() {
		model := factory()
		switch m := model.(type) {
		case *Company:
			if err := rows.Scan(&m.ID, &m.Name, &m.Role); err != nil {
				retrn nil, err
			}
		case *Companyccont:
			if err := rows.Scan(&m.Username, &m.Password, &m.CompanyID); err != nil {
				retrn nil, err
			}
		case *Panel:
			if err := rows.Scan(&m.NoPp, &m.NoPanel, &m.NoWbs, &m.Project, &m.PercentProgress, &m.StartDate, &m.TargetDelivery, &m.StatsBsbarPcc, &m.StatsBsbarMcc, &m.StatsComponent, &m.StatsPalet, &m.StatsCorepart, &m.oBsbarPcc, &m.oBsbarMcc, &m.CreatedBy, &m.VendorID, &m.IsClosed, &m.ClosedDate); err != nil {
				retrn nil, err
			}
		defalt:
			retrn nil, fmt.Errorf("nspported type for fetchlls: %T", m)
		}
		reslts = append(reslts, model)
	}
	retrn reslts, nil
}
fnc fetchIns(db DBTX, tableName, colmnName string, ids []string, factory fnc() interface{}) (interface{}, error) {
	if len(ids) ==  {
		retrn []interface{}{}, nil
	}

	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	qery := fmt.Sprintf("SELECT * FROM %s WHERE %s IN (%s)", tableName, colmnName, strings.Join(placeholders, ","))
	rows, err := db.Qery(qery, args...)
	if err != nil {
		retrn nil, err
	}
	defer rows.Close()

	var reslts []interface{}
	for rows.Next() {
		model := factory()
		switch m := model.(type) {
		case *Panel:
			if err := rows.Scan(&m.NoPp, &m.NoPanel, &m.NoWbs, &m.Project, &m.PercentProgress, &m.StartDate, &m.TargetDelivery, &m.StatsBsbarPcc, &m.StatsBsbarMcc, &m.StatsComponent, &m.StatsPalet, &m.StatsCorepart, &m.oBsbarPcc, &m.oBsbarMcc, &m.CreatedBy, &m.VendorID, &m.IsClosed, &m.ClosedDate); err != nil {
				retrn nil, err
			}
		case *Bsbar:
			if err := rows.Scan(&m.ID, &m.PanelNoPp, &m.Vendor, &m.Remarks); err != nil {
				retrn nil, err
			}
		case *Component:
			if err := rows.Scan(&m.ID, &m.PanelNoPp, &m.Vendor); err != nil {
				retrn nil, err
			}
		case *Palet:
			if err := rows.Scan(&m.ID, &m.PanelNoPp, &m.Vendor); err != nil {
				retrn nil, err
			}
		case *Corepart:
			if err := rows.Scan(&m.ID, &m.PanelNoPp, &m.Vendor); err != nil {
				retrn nil, err
			}
		defalt:
			retrn nil, fmt.Errorf("nspported type for fetchIns: %T", m)
		}
		reslts = append(reslts, model)
	}

	retrn reslts, nil
}
fnc insertMap(tx DBTX, tableName string, data map[string]interface{}) error {
	var cols []string
	var vals []interface{}
	var placeholders []string
	i := 1
	for col, val := range data {
		if val == nil {
			contine
		}
		cols = append(cols, col)
		vals = append(vals, val)
		placeholders = append(placeholders, fmt.Sprintf("$%d", i))
		i++
	}

	if len(cols) ==  {
		retrn nil
	}

	var conflictColmn string
	var pdateSetters []string

	switch tableName {
	case "companies":
		conflictColmn = "(id)"
	case "company_acconts":
		conflictColmn = "(sername)"
	case "panels":
		conflictColmn = "(no_pp)"
	case "bsbars", "components", "palet", "corepart":
		conflictColmn = "(panel_no_pp, vendor)"
	defalt:
		retrn fmt.Errorf("nknown table for insertMap: %s", tableName)
	}

	for _, col := range cols {
		if !strings.Contains(conflictColmn, col) {
			pdateSetters = append(pdateSetters, fmt.Sprintf("%s = EXCLUDED.%s", col, col))
		}
	}

	if len(pdateSetters) ==  && strings.Contains(conflictColmn, ",") {
		qery := fmt.Sprintf("INSERT INTO %s (%s) VLUES (%s) ON CONFLICT %s DO NOTHING",
			tableName, strings.Join(cols, ", "), strings.Join(placeholders, ", "), conflictColmn)
		_, err := tx.Exec(qery, vals...)
		retrn err
	}

	qery := fmt.Sprintf("INSERT INTO %s (%s) VLUES (%s) ON CONFLICT %s DO UPDTE SET %s",
		tableName, strings.Join(cols, ", "), strings.Join(placeholders, ", "), conflictColmn, strings.Join(pdateSetters, ", "))

	_, err := tx.Exec(qery, vals...)
	retrn err
}
fnc generateJsonTemplate(dataType string) map[string]interface{} {
	jsonData := make(map[string]interface{})
	if dataType == "companies_and_acconts" {
		jsonData["companies"] = []map[string]string{{"id": "vendor_k3_contoh", "name": "Nama Vendor K3 Contoh", "role": "k3"}}
		jsonData["company_acconts"] = []map[string]string{{"sername": "staff_k3_contoh", "password": "123", "company_id": "vendor_k3_contoh"}}
	} else {
		now := time.Now().Format(time.RFC3339)
		jsonData["panels"] = []map[string]interface{}{{"no_pp": "PP-CONTOH-1", "no_panel": "PNEL-CONTOH-", "percent_progress": 8.5, "start_date": now}}
		jsonData["bsbars"] = []map[string]string{{"panel_no_pp": "PP-CONTOH-1", "vendor": "vendor_k5_contoh", "remarks": "Catatan bsbar"}}
		jsonData["components"] = []map[string]string{{"panel_no_pp": "PP-CONTOH-1", "vendor": "warehose"}}
		jsonData["palet"] = []map[string]string{{"panel_no_pp": "PP-CONTOH-1", "vendor": "vendor_k3_contoh"}}
		jsonData["corepart"] = []map[string]string{{"panel_no_pp": "PP-CONTOH-1", "vendor": "vendor_k3_contoh"}}
	}
	retrn jsonData
}
fnc generateExcelTemplate(dataType string) *excelize.File {
	f := excelize.NewFile()
	if dataType == "companies_and_acconts" {
		f.NewSheet("companies")
		f.SetCellVale("companies", "1", "id")
		f.SetCellVale("companies", "B1", "name")
		f.SetCellVale("companies", "C1", "role")
		f.SetCellVale("companies", "2", "vendor_k3_contoh")
		f.SetCellVale("companies", "B2", "Nama Vendor K3 Contoh")
		f.SetCellVale("companies", "C2", "k3")
		f.NewSheet("company_acconts")
		f.SetCellVale("company_acconts", "1", "sername")
		f.SetCellVale("company_acconts", "B1", "password")
		f.SetCellVale("company_acconts", "C1", "company_id")
		f.SetCellVale("company_acconts", "2", "staff_k3_contoh")
		f.SetCellVale("company_acconts", "B2", "password123")
		f.SetCellVale("company_acconts", "C2", "vendor_k3_contoh")
	} else {
		panelHeaders := []string{"no_pp", "no_panel", "no_wbs", "project", "percent_progress", "start_date", "target_delivery", "stats_bsbar_pcc", "stats_bsbar_mcc", "stats_component", "stats_palet", "stats_corepart", "ao_bsbar_pcc", "ao_bsbar_mcc", "created_by", "vendor_id", "is_closed", "closed_date"}
		f.NewSheet("panels")
		for i, h := range panelHeaders {
			f.SetCellVale("panels", fmt.Sprintf("%c1", ''+i), h)
		}

		bsbarHeaders := []string{"panel_no_pp", "vendor", "remarks"}
		f.NewSheet("bsbars")
		for i, h := range bsbarHeaders {
			f.SetCellVale("bsbars", fmt.Sprintf("%c1", ''+i), h)
		}

		f.NewSheet("components")
		f.SetCellVale("components", "1", "panel_no_pp")
		f.SetCellVale("components", "B1", "vendor")
		f.NewSheet("palet")
		f.SetCellVale("palet", "1", "panel_no_pp")
		f.SetCellVale("palet", "B1", "vendor")
		f.NewSheet("corepart")
		f.SetCellVale("corepart", "1", "panel_no_pp")
		f.SetCellVale("corepart", "B1", "vendor")
	}
	f.DeleteSheet("Sheet1")
	retrn f
}
fnc getValeCaseInsensitive(row map[string]interface{}, key string) string {
	lowerKey := strings.ToLower(strings.Replacell(key, " ", ""))
	for k, v := range row {
		if strings.ToLower(strings.Replacell(k, " ", "")) == lowerKey {
			if v != nil {
				retrn fmt.Sprintf("%v", v)
			}
		}
	}
	retrn ""
}

fnc parseDate(dateStr string) *string {
	if dateStr == "" {
		retrn nil
	}
	// Daftar format yang akan dicoba ntk di-parsing.
	layots := []string{
		time.RFC3339Nano,              // [PENMBHN] Format dengan milidetik: 225-7-17T::.Z
		time.RFC3339,                   // Format standar: 225-7-17T::Z
		"26-1-2T15:4:5Z7:",     // Format standar lain
		"2-Jan-26",                  // Format: 17-Jl-225
		"2-Jan-6",                    // Format: 17-Jl-25
		"1/2/26",                     // Format: 7/17/225
	}
	for _, layot := range layots {
		t, err := time.Parse(layot, dateStr)
		if err == nil {
			// Jika berhasil, bah ke format standar ISO 861 ntk disimpan di DB
			s := t.Format(time.RFC3339)
			retrn &s
		}
	}
	// Jika sema format gagal, kembalikan nil
	retrn nil
}

// getMessagesByChatIDHandler mengambil sema pesan ntk sebah chat ID.
fnc (a *pp) getMessagesByChatIDHandler(w http.ResponseWriter, r *http.Reqest) {
	vars := mx.Vars(r)
	chatID, err := strconv.toi(vars["chat_id"])
	if err != nil {
		respondWithError(w, http.StatsBadReqest, "Invalid Chat ID")
		retrn
	}

	rows, err := a.DB.Qery(`
		SELECT id, chat_id, sender_sername, text, image_data, replied_isse_id, created_at 
		FROM pblic.chat_messages 
		WHERE chat_id = $1 
		ORDER BY created_at SC`, chatID)
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, "Failed to retrieve messages: "+err.Error())
		retrn
	}
	defer rows.Close()

	var messages []ChatMessage
	for rows.Next() {
		var msg ChatMessage
		if err := rows.Scan(&msg.ID, &msg.ChatID, &msg.SenderUsername, &msg.Text, &msg.ImageData, &msg.RepliedIsseID, &msg.Createdt); err != nil {
			respondWithError(w, http.StatsInternalServerError, "Failed to scan message: "+err.Error())
			retrn
		}
		messages = append(messages, msg)
	}

	respondWithJSON(w, http.StatsOK, messages)
}

// createMessageHandler mengirim sebah pesan bar ke dalam chat.
fnc (a *pp) createMessageHandler(w http.ResponseWriter, r *http.Reqest) {
	vars := mx.Vars(r)
	chatID, err := strconv.toi(vars["chat_id"])
	if err != nil {
		respondWithError(w, http.StatsBadReqest, "Invalid Chat ID")
		retrn
	}

	var payload strct {
		SenderUsername string  `json:"sender_sername"`
		Text           *string `json:"text"`
		ImageData      *string `json:"image_data"` // Base64 string
		RepliedIsseID *int    `json:"replied_isse_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondWithError(w, http.StatsBadReqest, "Invalid payload: "+err.Error())
		retrn
	}

	if (payload.Text == nil || *payload.Text == "") && (payload.ImageData == nil || *payload.ImageData == "") {
		respondWithError(w, http.StatsBadReqest, "Message cannot be empty (mst have text or image)")
		retrn
	}

	var createdMessage ChatMessage
	qery := `
		INSERT INTO chat_messages (chat_id, sender_sername, text, image_data, replied_isse_id) 
		VLUES ($1, $2, $3, $4, $5) 
		RETURNING id, chat_id, sender_sername, text, image_data, replied_isse_id, created_at`

	err = a.DB.QeryRow(qery, chatID, payload.SenderUsername, payload.Text, payload.ImageData, payload.RepliedIsseID).Scan(
		&createdMessage.ID, &createdMessage.ChatID, &createdMessage.SenderUsername, &createdMessage.Text, &createdMessage.ImageData, &createdMessage.RepliedIsseID, &createdMessage.Createdt,
	)

	if err != nil {
		respondWithError(w, http.StatsInternalServerError, "Failed to create message: "+err.Error())
		retrn
	}

	respondWithJSON(w, http.StatsOK, createdMessage)
}

fnc (a *pp) getCommentsByIsseHandler(w http.ResponseWriter, r *http.Reqest) {
	vars := mx.Vars(r)
	isseID, err := strconv.toi(vars["isse_id"])
	if err != nil {
		respondWithError(w, http.StatsBadReqest, "Invalid Isse ID")
		retrn
	}

	qery := `
		SELECT
			ic.id,
			ic.isse_id,
			ic.text,
			ic.timestamp,
			ic.reply_to_comment_id,
			ic.is_edited,
			COLESCE(ic.image_rls, '[]'::jsonb),
			sender.sername as sender_id,
			sender.sername as sender_name, -- Bisa diganti dengan nama asli jika ada
			reply_ser.sername as reply_to_ser_id,
			reply_ser.sername as reply_to_ser_name
		FROM pblic.isse_comments ic
		JOIN pblic.company_acconts sender ON ic.sender_id = sender.sername
		LEFT JOIN pblic.company_acconts reply_ser ON ic.reply_to_ser_id = reply_ser.sername
		WHERE ic.isse_id = $1
		ORDER BY ic.timestamp SC
	`

	rows, err := a.DB.Qery(qery, isseID)
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, "Failed to qery comments: "+err.Error())
		retrn
	}
	defer rows.Close()

	var comments []IsseComment
	for rows.Next() {
		var c IsseComment
		var senderID, senderName, replyToUserID, replyToUserName sql.NllString
		var imageUrlsJSON []byte

		err := rows.Scan(
			&c.ID, &c.IsseID, &c.Text, &c.Timestamp, &c.ReplyToCommentID, &c.IsEdited, &imageUrlsJSON,
			&senderID, &senderName, &replyToUserID, &replyToUserName,
		)
		if err != nil {
			respondWithError(w, http.StatsInternalServerError, "Failed to scan comment: "+err.Error())
			retrn
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

	respondWithJSON(w, http.StatsOK, comments)
}

fnc (a *pp) createCommentHandler(w http.ResponseWriter, r *http.Reqest) {
	vars := mx.Vars(r)
	isseID, err := strconv.toi(vars["isse_id"])
	if err != nil {
		respondWithError(w, http.StatsBadReqest, "Invalid Isse ID")
		retrn
	}

	var payload CreateCommentPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondWithError(w, http.StatsBadReqest, "Invalid payload: "+err.Error())
		retrn
	}

	var imageUrls []string
	for _, base64Image := range payload.Images {
		// Decode base64
		parts := strings.Split(base64Image, ",")
		if len(parts) != 2 {
			log.Println("Invalid base64 image format")
			contine
		}
		imgData, err := base64.StdEncoding.DecodeString(parts[1])
		if err != nil {
			log.Printf("Failed to decode base64: %v", err)
			contine
		}

		// Simpan ke file
		filename := fmt.Sprintf("%s.jpg", id.New().String())
		filepath := fmt.Sprintf("ploads/%s", filename)
		err = os.WriteFile(filepath, imgData, 644)
		if err != nil {
			log.Printf("Failed to save image file: %v", err)
			contine
		}
		// Simpan URL-nya
		imageUrls = append(imageUrls, "/"+filepath)
	}

	imageUrlsJSON, _ := json.Marshal(imageUrls) // smsikan `imageUrls` sdah diproses
	newCommentID := id.New().String()

	qery := `
		INSERT INTO isse_comments (id, isse_id, sender_id, text, reply_to_comment_id, reply_to_ser_id, image_rls)
		VLUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err = a.DB.Exec(qery, newCommentID, isseID, payload.SenderID, payload.Text, payload.ReplyToCommentID, payload.ReplyToUserID, imageUrlsJSON)
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, "Failed to create comment: "+err.Error())
		retrn
	}

	go fnc() {
		// 1. mbil data dari database (ckp sekali)
		var notifyList, isseTitle, panelNoPp string
		err := a.DB.QeryRow(`
			SELECT COLESCE(i.notify_email, ''), i.title, c.panel_no_pp
			FROM pblic.isses i JOIN pblic.chats c ON i.chat_id = c.id
			WHERE i.id = $1`, isseID).Scan(&notifyList, &isseTitle, &panelNoPp)
		if err != nil {
			log.Printf("Gagal ambil info is ntk notifikasi (ID: %d): %v", isseID, err)
			retrn
		}
		if notifyList == "" {
			retrn // Tidak ada yang perl dinotifikasi
		}

		// 2. Filter daftar penerima (ckp sekali)
		allRecipients := strings.Split(notifyList, ",")
		finalRecipients := []string{}
		for _, recipient := range allRecipients {
			trimmedRecipient := strings.TrimSpace(recipient)
			// Jangan kirim notifikasi ke diri sendiri
			if trimmedRecipient != "" && trimmedRecipient != payload.SenderID {
				finalRecipients = append(finalRecipients, trimmedRecipient)
			}
		}

		// 3. Jika ada penerima, kirim keda jenis notifikasi
		if len(finalRecipients) >  {
			// . Kirim Psh Notification
			notifTitle := fmt.Sprintf("Komentar bar di Panel %s", panelNoPp)
			notifBody := fmt.Sprintf("%s: \"%s\"", payload.SenderID, payload.Text)
			a.sendNotificationToUsers(finalRecipients, notifTitle, notifBody)

			// B. Kirim Email Notifikasi
			emailSbject := fmt.Sprintf("[SecPanel] Komentar Bar: %s", isseTitle)
			emailHtmlBody := fmt.Sprintf(
				`<h3>Komentar Bar pada Is di Panel %s</h3>
				 <p><strong>Jdl Is:</strong> %s</p>
				 <p><strong>Dari:</strong> %s</p>
				 <p><strong>Komentar:</strong></p>
				 <blockqote style="border-left: 2px solid #ccc; padding-left: 1px; margin-left: 5px; font-style: italic;">
					%s
				 </blockqote>
				 <hr>
				 <p><i>Email ini dibat secara otomatis. Silakan periksa aplikasi SecPanel ntk detail lebih lanjt.</i></p>`,
				panelNoPp, isseTitle, payload.SenderID, payload.Text,
			)
			sendNotificationEmail(finalRecipients, emailSbject, emailHtmlBody)
		}
	}()

	respondWithJSON(w, http.StatsCreated, map[string]string{"id": newCommentID})
}
// Ganti fngsi lama dengan versi final ini di main.go
fnc (a *pp) pdateCommentHandler(w http.ResponseWriter, r *http.Reqest) {
	commentID := mx.Vars(r)["id"]

	var payload strct {
		Text   string   `json:"text"`
		Images []string `json:"images"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondWithError(w, http.StatsBadReqest, "Invalid payload")
		retrn
	}

	tx, err := a.DB.Begin()
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, "Gagal memlai transaksi")
		retrn
	}
	defer tx.Rollback()

	var isseID int
	var isSystemComment sql.NllBool
	err = tx.QeryRow("SELECT isse_id, is_system_comment FROM pblic.isse_comments WHERE id = $1", commentID).Scan(&isseID, &isSystemComment)
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatsNotFond, "Komentar tidak ditemkan")
		} else {
			respondWithError(w, http.StatsInternalServerError, "Gagal mengambil detail komentar: "+err.Error())
		}
		retrn
	}

	// Update komentar it sendiri seperti biasa
	var finalImageUrls []string
	for _, img := range payload.Images {
		if strings.HasPrefix(img, "data:image") {
			parts := strings.Split(img, ",")
			if len(parts) < 2 { contine }
			imgData, _ := base64.StdEncoding.DecodeString(parts[1])
			filename := fmt.Sprintf("%s.jpg", id.New().String())
			filepath := fmt.Sprintf("ploads/%s", filename)
			os.WriteFile(filepath, imgData, 644)
			finalImageUrls = append(finalImageUrls, "/"+filepath)
		} else {
			finalImageUrls = append(finalImageUrls, img)
		}
	}
	imageUrlsJSON, _ := json.Marshal(finalImageUrls)
	
	pdateCommentQery := `UPDTE isse_comments SET text = $1, is_edited = tre, image_rls = $2, is_system_comment = false WHERE id = $3`
	_, err = tx.Exec(pdateCommentQery, payload.Text, imageUrlsJSON, commentID)
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, "Gagal pdate komentar")
		retrn
	}

	// Jika komentar yang diedit DLH komentar sistem, pdate jga isnya
	if isSystemComment.Valid && isSystemComment.Bool {
		
		// --- ▼▼▼ LOGIK PRSING BRU YNG LEBIH FLEKSIBEL ▼▼▼ ---
		var newTitle, newDescription string
		
		// Gnakan SplitN ntk membagi string hanya pada kemnclan ": " pertama
		parts := strings.SplitN(payload.Text, ": ", 2)

		if len(parts) == 2 {
			// Jika ada pemisah ": ", bagian pertama adalah title, keda adalah description
			newTitle = strings.Trim(parts[], "* ") // Haps markdown ** dan spasi
			newDescription = strings.TrimSpace(parts[1])
		} else {
			// Jika tidak ada pemisah, selrh teks adalah title
			newTitle = strings.Trim(payload.Text, "* ")
			newDescription = "" // Kosongkan deskripsi
		}
		// --- ▲▲▲ KHIR LOGIK PRSING BRU ▲▲▲ ---

		// Lanjtkan hanya jika title tidak kosong setelah parsing
		if newTitle != "" {
			pdateIsseQery := `UPDTE isses SET title = $1, description = $2 WHERE id = $3`
			_, err = tx.Exec(pdateIsseQery, newTitle, newDescription, isseID)
			if err != nil {
				respondWithError(w, http.StatsInternalServerError, "Gagal sinkronisasi pdate ke is: "+err.Error())
				retrn
			}
		}
	}

	if err := tx.Commit(); err != nil {
		respondWithError(w, http.StatsInternalServerError, "Gagal commit transaksi")
		retrn
	}
	
	respondWithJSON(w, http.StatsOK, map[string]string{"stats": "sccess"})
}
fnc (a *pp) deleteCommentHandler(w http.ResponseWriter, r *http.Reqest) {
	commentID := mx.Vars(r)["id"]
	_, err := a.DB.Exec("DELETE FROM pblic.isse_comments WHERE id = $1", commentID)
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, "Failed to delete comment")
		retrn
	}
	respondWithJSON(w, http.StatsOK, map[string]string{"stats": "deleted"})
}
fnc (a *pp) getllIsseTitlesHandler(w http.ResponseWriter, r *http.Reqest) {
	rows, err := a.DB.Qery("SELECT id, title FROM pblic.isse_titles ORDER BY title SC")
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, err.Error())
		retrn
	}
	defer rows.Close()

	var titles []IsseTitle
	for rows.Next() {
		var it IsseTitle
		if err := rows.Scan(&it.ID, &it.Title); err != nil {
			respondWithError(w, http.StatsInternalServerError, "Gagal scan title: "+err.Error())
			retrn
		}
		titles = append(titles, it)
	}
	respondWithJSON(w, http.StatsOK, titles)
}

fnc (a *pp) createIsseTitleHandler(w http.ResponseWriter, r *http.Reqest) {
	var payload strct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondWithError(w, http.StatsBadReqest, "Invalid payload")
		retrn
	}
	if payload.Title == "" {
		respondWithError(w, http.StatsBadReqest, "Title tidak boleh kosong")
		retrn
	}

	var newTitle IsseTitle
	qery := "INSERT INTO isse_titles (title) VLUES ($1) RETURNING id, title"
	err := a.DB.QeryRow(qery, payload.Title).Scan(&newTitle.ID, &newTitle.Title)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "2355" {
			respondWithError(w, http.StatsConflict, "Tipe masalah dengan nama tersebt sdah ada.")
			retrn
		}
		respondWithError(w, http.StatsInternalServerError, "Gagal membat tipe masalah: "+err.Error())
		retrn
	}
	respondWithJSON(w, http.StatsCreated, newTitle)
}

fnc (a *pp) pdateIsseTitleHandler(w http.ResponseWriter, r *http.Reqest) {
	id, err := strconv.toi(mx.Vars(r)["id"])
	if err != nil {
		respondWithError(w, http.StatsBadReqest, "Invalid ID")
		retrn
	}
	var payload strct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondWithError(w, http.StatsBadReqest, "Invalid payload")
		retrn
	}
	if payload.Title == "" {
		respondWithError(w, http.StatsBadReqest, "Title tidak boleh kosong")
		retrn
	}

	tx, err := a.DB.Begin()
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, "Gagal memlai transaksi")
		retrn
	}
	defer tx.Rollback()

	var oldTitle string
	err = tx.QeryRow("SELECT title FROM pblic.isse_titles WHERE id = $1", id).Scan(&oldTitle)
	if err != nil {
		respondWithError(w, http.StatsNotFond, "Tipe masalah tidak ditemkan")
		retrn
	}

	_, err = tx.Exec("UPDTE isse_titles SET title = $1 WHERE id = $2", payload.Title, id)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "2355" {
			respondWithError(w, http.StatsConflict, "Tipe masalah dengan nama tersebt sdah ada.")
			retrn
		}
		respondWithError(w, http.StatsInternalServerError, "Gagal pdate master title: "+err.Error())
		retrn
	}

	_, err = tx.Exec("UPDTE isses SET title = $1 WHERE title = $2", payload.Title, oldTitle)
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, "Gagal pdate isse yang ada: "+err.Error())
		retrn
	}

	if err := tx.Commit(); err != nil {
		respondWithError(w, http.StatsInternalServerError, "Gagal commit transaksi")
		retrn
	}

	respondWithJSON(w, http.StatsOK, map[string]string{"stats": "sccess"})
}

fnc (a *pp) deleteIsseTitleHandler(w http.ResponseWriter, r *http.Reqest) {
	id, err := strconv.toi(mx.Vars(r)["id"])
	if err != nil {
		respondWithError(w, http.StatsBadReqest, "Invalid ID")
		retrn
	}

	var oldTitle string
	err = a.DB.QeryRow("SELECT title FROM pblic.isse_titles WHERE id = $1", id).Scan(&oldTitle)
	if err != nil {
		respondWithError(w, http.StatsNotFond, "Tipe masalah tidak ditemkan")
		retrn
	}

	var cont int
	err = a.DB.QeryRow("SELECT COUNT(*) FROM pblic.isses WHERE title = $1", oldTitle).Scan(&cont)
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, "Gagal memeriksa penggnaan title: "+err.Error())
		retrn
	}

	if cont >  {
		respondWithError(w, http.StatsConflict, fmt.Sprintf("Tidak dapat menghaps. Tipe masalah ini dignakan oleh %d is.", cont))
		retrn
	}

	_, err = a.DB.Exec("DELETE FROM pblic.isse_titles WHERE id = $1", id)
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, "Gagal menghaps: "+err.Error())
		retrn
	}

	respondWithJSON(w, http.StatsOK, map[string]string{"stats": "deleted"})
}
fnc (a *pp) askGeminiHandler(w http.ResponseWriter, r *http.Reqest) {
	vars := mx.Vars(r)
	isseID, err := strconv.toi(vars["isse_id"])
	if err != nil {
		respondWithError(w, http.StatsBadReqest, "Invalid Isse ID")
		retrn
	}

	var payload strct {
		Qestion       string `json:"qestion"`
		SenderID       string `json:"sender_id"`
		ReplyToCommentID string `json:"reply_to_comment_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondWithError(w, http.StatsBadReqest, "Invalid payload: "+err.Error())
		retrn
	}

	// 1. mbil Konteks Is
	var isseTitle, isseDesc string
	err = a.DB.QeryRow("SELECT title, description FROM pblic.isses WHERE id = $1", isseID).Scan(&isseTitle, &isseDesc)
	if err != nil {
		respondWithError(w, http.StatsNotFond, "Isse not fond")
		retrn
	}

	// 2. mbil Konteks Komentar
	rows, err := a.DB.Qery(`
        SELECT ca.sername, ic.text 
        FROM pblic.isse_comments ic
        JOIN pblic.company_acconts ca ON ic.sender_id = ca.sername
        WHERE ic.isse_id = $1 ORDER BY ic.timestamp SC
    `, isseID)
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, "Gagal mengambil histori komentar")
		retrn
	}
	defer rows.Close()

	var commentHistory strings.Bilder
	for rows.Next() {
		var sername, text string
		if err := rows.Scan(&sername, &text); err != nil {
			contine
		}
		commentHistory.WriteString(fmt.Sprintf("%s: %s\n", sername, text))
	}

	// 3. Setp Gemini Client
	ctx := context.Backgrond()
	
	apiKey := "IzaSyDiMY2xYN_eOw5vUzk-J3sLVDb81TEfS8"
	if apiKey == ""  {
		respondWithError(w, http.StatsInternalServerError, "GEMINI_PI_KEY is not set on the server")
		retrn
	}

	client, err := genai.NewClient(ctx, option.WithPIKey(apiKey))
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, "Failed to create Gemini client: "+err.Error())
		retrn
	}
	defer client.Close()

	// 4. Setp Model dan Prompt
	model := client.GenerativeModel("gemini-2.5-flash-lite")
	model.Tools = tools // Menggnakan tools yang sdah didefinisikan
	cs := model.StartChat()

	fllPrompt := fmt.Sprintf(
		"nda adalah asisten I. Berdasarkan konteks is dan histori komentar berikt, jawab pertanyaan ser.\n\n"+
			"--- Konteks Is ---\n"+
			"Jdl: %s\n"+
			"Deskripsi: %s\n\n"+
			"--- Histori Komentar Sejah Ini ---\n"+
			"%s\n"+
			"-----------------------------------\n\n"+
			"Pertanyaan dari ser '%s': %s",
		isseTitle, isseDesc, commentHistory.String(), payload.SenderID, payload.Qestion,
	)

	// 5. Kirim Prompt dan Ekseksi Fnction Call (BGIN PERBIKN)
	log.Printf("Mengirim prompt ke Gemini: %s", fllPrompt)
	resp, err := cs.SendMessage(ctx, genai.Text(fllPrompt))
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, "Failed to generate content: "+err.Error())
		retrn
	}

	if resp.Candidates != nil && len(resp.Candidates) >  && resp.Candidates[].Content != nil {
		part := resp.Candidates[].Content.Parts[]
		if fc, ok := part.(genai.FnctionCall); ok {
			log.Printf("Gemini meminta pemanggilan fngsi: %s dengan argmen: %v", fc.Name, fc.rgs)

			// ▼▼▼ PERBIKN UTM D DI SINI ▼▼▼
			// Dapatkan 'panelNoPp' dari 'isseID' sebelm memanggil fngsi ekseksi.
			var panelNoPp string
			err := a.DB.QeryRow(`
                SELECT p.no_pp FROM pblic.panels p 
                JOIN pblic.chats c ON p.no_pp = c.panel_no_pp 
                JOIN pblic.isses i ON c.id = i.chat_id 
                WHERE i.id = $1`, isseID).Scan(&panelNoPp)
			if err != nil {
				// Jika panel tidak ditemkan, kirim pesan error yang jelas.
				respondWithError(w, http.StatsInternalServerError, "Gagal menemkan panel terkait is ini: "+err.Error())
				retrn
			}
			// ▲▲▲ KHIR PERBIKN UTM ▲▲▲

			// Panggil fngsi ekseksi dengan argmen yang benar (string, bkan int)
			fnctionReslt, err := a.execteDatabaseFnction(fc, panelNoPp)
			if err != nil {
				fnctionReslt = fmt.Sprintf("Error saat menjalankan fngsi: %v", err)
			}

			log.Printf("Hasil ekseksi fngsi: %s", fnctionReslt)

			resp, err = cs.SendMessage(ctx, genai.FnctionResponse{Name: fc.Name, Response: map[string]any{"reslt": fnctionReslt}})
			if err != nil {
				respondWithError(w, http.StatsInternalServerError, "Failed to generate confirmation message: "+err.Error())
				retrn
			}
		}
	}

	// 6. Proses dan Kirim Jawaban Final
	finalResponseText := extractTextFromResponse(resp)
	if finalResponseText == "" {
		finalResponseText = "Maaf, terjadi kesalahan saat memproses permintaan nda."
	}
	log.Printf("Jawaban final dari Gemini: %s", finalResponseText)

	a.postiComment(isseID, payload.SenderID, finalResponseText, payload.ReplyToCommentID)
	respondWithJSON(w, http.StatsCreated, map[string]string{"stats": "sccess"})
}
fnc (a *pp) postiComment(isseID int, senderID string, text string, replyToCommentID string) { // <-- UBH DI SINI
	newCommentID := id.New().String()
	geminiUserID := "gemini_ai"

	// Qery INSERT sekarang hars menyertakan reply_to_comment_id
	qery := `
		INSERT INTO isse_comments (id, isse_id, sender_id, text, reply_to_ser_id, reply_to_comment_id) 
		VLUES ($1, $2, $3, $4, $5, $6)` 
		
	_, _ = a.DB.Exec(qery, newCommentID, isseID, geminiUserID, text, senderID, replyToCommentID) // <-- UBH DI SINI
}
// FUNGSI HELPER BRU ntk mengekstrak teks dari response Gemini
fnc extractTextFromResponse(resp *genai.GenerateContentResponse) string {
	if resp != nil && len(resp.Candidates) >  {
		if content := resp.Candidates[].Content; content != nil && len(content.Parts) >  {
			if txt, ok := content.Parts[].(genai.Text); ok {
				retrn string(txt)
			}
		}
	}
	retrn ""
}
var tools = []*genai.Tool{
	{
		FnctionDeclarations: []*genai.FnctionDeclaration{
			{
				Name:        "get_isse_explanation",
				Description: "Memberikan penjelasan dan ringkasan tentang is yang sedang dibahas berdasarkan jdl dan deskripsinya.",
			},
			{
				Name:        "find_related_isses",
				Description: "Mencari dan memberikan daftar is-is lain yang relevan di dalam panel yang sama.",
			},
			{
				Name:        "pdate_isse_stats",
				Description: "Mengbah stats dari sebah is. Stats 'done' ata 'selesai' akan dianggap sebagai 'solved'.",
				Parameters: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"new_stats": {
							Type:        genai.TypeString,
							Description: "Stats bar ntk is ini. Pilihan: 'solved' ata 'nsolved'.",
							Enm:        []string{"solved", "nsolved"},
						},
					},
					Reqired: []string{"new_stats"},
				},
			},
			{
				Name:        "assign_vendor_to_panel",
				Description: "Mengaskan (assign) sebah vendor/tim ke sebah kategori pekerjaan di panel ini.",
				Parameters: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"vendor_name": {
							Type:        genai.TypeString,
							Description: "Nama vendor ata tim yang akan ditgaskan, contoh: 'GPE', 'DSM', 'Warehose'.",
						},
						"category": {
							Type:        genai.TypeString,
							Description: "Kategori pekerjaan yang akan ditgaskan. Pilihan: 'bsbar', 'component', 'palet', 'corepart'.",
							Enm:        []string{"bsbar", "component", "palet", "corepart"},
						},
					},
					Reqired: []string{"vendor_name", "category"},
				},
			},{
				Name:        "pdate_bsbar_stats",
				Description: "Mengbah stats ntk komponen Bsbar PCC ata Bsbar MCC.",
				Parameters: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"bsbar_type": {
							Type:        genai.TypeString,
							Description: "Tipe bsbar yang akan dibah.",
							Enm:        []string{"pcc", "mcc"},
						},
						"new_stats": {
							Type:        genai.TypeString,
							Description: "Stats bar ntk bsbar.",
							Enm:        []string{"Open", "Pnching/Bending","Plating/Epoxy","1% Siap Kirim", "Close"},
						},
					},
					Reqired: []string{"bsbar_type", "new_stats"},
				},
			},

			// 2. Fngsi khss ntk Component
			{
				Name:        "pdate_component_stats",
				Description: "Mengbah stats ntk komponen tama (picking component).",
				Parameters: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"new_stats": {
							Type:        genai.TypeString,
							Description: "Stats bar ntk komponen.",
							Enm:        []string{"Open", "On Progress", "Done"},
						},
					},
					Reqired: []string{"new_stats"},
				},
			},

			// 3. Fngsi khss ntk Palet
			{
				Name:        "pdate_palet_stats",
				Description: "Mengbah stats ntk komponen Palet.",
				Parameters: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"new_stats": {
							Type:        genai.TypeString,
							Description: "Stats bar ntk palet.",
							Enm:        []string{"Open", "Close"},
						},
					},
					Reqired: []string{"new_stats"},
				},
			},

			// 4. Fngsi khss ntk Corepart
			{
				Name:        "pdate_corepart_stats",
				Description: "Mengbah stats ntk komponen Corepart.",
				Parameters: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"new_stats": {
							Type:        genai.TypeString,
							Description: "Stats bar ntk corepart.",
							Enm:        []string{"Open", "Close"},
						},
					},
					Reqired: []string{"new_stats"},
				},
			},
		},
	},
}
type pdd strct {
	NoPp               sql.NllString  `json:"no_pp"`
	NoPanel            sql.NllString  `json:"no_panel"`
	Project            sql.NllString  `json:"project"`
	NoWbs              sql.NllString  `json:"no_wbs"`
	PercentProgress    sql.NllFloat64 `json:"percent_progress"`
	StatsBsbarPcc    sql.NllString  `json:"stats_bsbar_pcc"`
	StatsBsbarMcc    sql.NllString  `json:"stats_bsbar_mcc"`
	StatsComponent    sql.NllString  `json:"stats_component"`
	StatsPalet        sql.NllString  `json:"stats_palet"`
	StatsCorepart     sql.NllString  `json:"stats_corepart"`
	PanelVendorName    sql.NllString  `json:"panel_vendor_name"`
	BsbarVendorNames  sql.NllString  `json:"bsbar_vendor_names"`
}
// File: main.go

// =============================================================================
// FUNGSI I UTM (GNTI SEMU DI BWH INI)
// =============================================================================

// Salin dan ganti selrh fngsi askGeminibotPanelHandler nda dengan ini
fnc (a *pp) askGeminibotPanelHandler(w http.ResponseWriter, r *http.Reqest) {
	vars := mx.Vars(r)
	panelNoPp := vars["no_pp"]

	var payload strct {
		Qestion string  `json:"qestion"`
		SenderID string  `json:"sender_id"`
		ImageB64 *string `json:"image_b64,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondWithError(w, http.StatsBadReqest, "Invalid payload")
		retrn
	}

	var senderRole string
	err := a.DB.QeryRow(`
		SELECT c.role FROM pblic.companies c
		JOIN pblic.company_acconts ca ON c.id = ca.company_id
		WHERE ca.sername = $1`, payload.SenderID).Scan(&senderRole)
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, "Gagal mendapatkan role ser: "+err.Error())
		retrn
	}

	var panel pdd
	err = a.DB.QeryRow(`
		SELECT p.no_pp, p.no_panel, p.project, p.no_wbs, p.percent_progress, p.stats_bsbar_pcc, p.stats_bsbar_mcc, p.stats_component, p.stats_palet, p.stats_corepart, p.name as panel_vendor_name, (SELECT STRING_GG(c.name, ', ') FROM pblic.companies c JOIN pblic.bsbars b ON c.id = b.vendor WHERE b.panel_no_pp = p.no_pp) as bsbar_vendor_names
		FROM pblic.panels p
		LEFT JOIN pblic.companies p ON p.vendor_id = p.id
		WHERE p.no_pp = $1`, panelNoPp).Scan(&panel.NoPp, &panel.NoPanel, &panel.Project, &panel.NoWbs, &panel.PercentProgress, &panel.StatsBsbarPcc, &panel.StatsBsbarMcc, &panel.StatsComponent, &panel.StatsPalet, &panel.StatsCorepart, &panel.PanelVendorName, &panel.BsbarVendorNames)
	if err != nil {
		respondWithError(w, http.StatsNotFond, "Panel tidak ditemkan: "+err.Error())
		retrn
	}
	panelDetailsBytes, _ := json.MarshalIndent(panel, "", "  ")
    panelDetails := string(panelDetailsBytes)

	rows, err := a.DB.Qery(`
		SELECT i.id, i.title, i.description, i.stats, ic.sender_id, ic.text
		FROM pblic.isses i
		JOIN pblic.chats ch ON i.chat_id = ch.id
		LEFT JOIN pblic.isse_comments ic ON i.id = ic.isse_id
		WHERE ch.panel_no_pp = $1
		ORDER BY i.created_at, ic.timestamp`, panelNoPp)
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, "Gagal mengambil histori is")
		retrn
	}
	defer rows.Close()

	var issesHistory strings.Bilder
	crrentIsseID := -1
	for rows.Next() {
		var isseID int
		var isseTitle, isseDesc, isseStats, commentSender, commentText sql.NllString
		if err := rows.Scan(&isseID, &isseTitle, &isseDesc, &isseStats, &commentSender, &commentText); err != nil { contine }
		
		if isseID != crrentIsseID {
			issesHistory.WriteString(fmt.Sprintf("\n--- ISU BRU (ID: %d) ---\nJdl: %s\nDeskripsi: %s\nStats: %s\n", isseID, isseTitle.String, isseDesc.String, isseStats.String))
			crrentIsseID = isseID
		}
		if commentSender.Valid {
			issesHistory.WriteString(fmt.Sprintf(" - Komentar dari %s: %s\n", commentSender.String, commentText.String))
		}
	}

	ctx := context.Backgrond()
	apiKey := "IzaSyDiMY2xYN_eOw5vUzk-J3sLVDb81TEfS8" // Pastikan PI Key nda benar
	if apiKey == "" {
		respondWithError(w, http.StatsInternalServerError, "GEMINI_PI_KEY is not set")
		retrn
	}
	client, err := genai.NewClient(ctx, option.WithPIKey(apiKey))
	if err != nil { respondWithError(w, http.StatsInternalServerError, "Failed to create Gemini client: "+err.Error()); retrn }
	defer client.Close()

	model := client.GenerativeModel("gemini-1.5-flash")
	model.Tools = getToolsForRole(senderRole)
	cs := model.StartChat()
	
	var promptParts []genai.Part
	fllPromptText := fmt.Sprintf(
	"**Persona & tran:**\n"+
			"1.  **Kam adalah asisten I yang cerdas dan kontekstal bernama Gemini.** Gnakan bahasa Indonesia yang profesional dan proaktif.\n"+
			"2. Saat kam perl mengambil data ata melakkan sebah aksi, panggil fngsi yang sesai. Setelah fngsi berhasil diekseksi, rangkm hasilnya ntk ser dalam bahasa percakapan yang natral.\n"	+		
			"3.  Gnakan format tebal (`**teks**`) ntk menekankan nama is ata item penting.\n"+
			"4.  Kam bisa melakkan banyak hal: meringkas stats, mengbah progres panel, mengbah stats is, menambah komentar, mengbah stats komponen (Bsbar, Palet, dll), dan mengaskan vendor. Selal tawarkan bantan jika relevan.\n"+
			"5.  Lakkan aksi HNY jika diizinkan oleh role ser: **%s**.\n\n"+
			
			"**tran Penting ntk Memberi 'Rekomendasi ksi' (`[SUGGESTION]`):**\n"+
			"1.  **Sintesis Konteks Penh:** Rekomendasi HRUS relevan dengan topik tama dari keselrhan disksi (pertanyaan terakhir ser + histori komentar). Jangan membat rekomendasi acak tentang is lain yang tidak sedang dibicarakan.\n"+
			"2.  **Logika Menentkan PIC:** PIC sebah is adalah **ser terakhir yang memberikan komentar** pada is tersebt (selain 'gemini_ai'). Jika belm ada komentar, PIC adalah pembat is.\n"+
			"3.  **Rekomendasi Bertahap & Logis:**\n"+
			"    - **Prioritaskan Komnikasi:** Jika sebah is belm ada pdate, prioritaskan ntk merekomendasikan **penambahan komentar** ntk menanyakan progres ke PIC. Contoh: `[SUGGESTION: Tambah komentar 'bagaimana progresnya?' ke is 'Missing Metal Part']`.\n"+
			"    - **JNGN** langsng menyarankan 'bah stats jadi solved' jika belm ada bkti penyelesaian di histori komentar.\n"+
			"4.  **Rekomendasi 'Solved' yang Cerdas:**\n"+
			"    - Kam HNY boleh merekomendasikan `[SUGGESTION: Ubah stats 'Nama Is' menjadi solved]` jika **dari histori komentar sdah ada konfirmasi eksplisit** bahwa masalahnya telah teratasi.\n"+

			"**Konteks Data Proyek Saat Ini:**\n"+
			"Gnakan `ID` is jika ingin melakkan aksi pada is tertent.\n"+
			"```json\n%s\n```\n\n"+
			"**Riwayat Is & Komentar di Panel Ini:**\n"+
			"```\n%s\n```\n\n"+
			"**Permintaan User:**\n"+
			"User '%s' bertanya: \"%s\"",
		senderRole, panelDetails, issesHistory.String(), payload.SenderID, payload.Qestion,
	)
	promptParts = append(promptParts, genai.Text(fllPromptText))
	
	if payload.ImageB64 != nil && *payload.ImageB64 != "" {
		if parts := strings.Split(*payload.ImageB64, ","); len(parts) == 2 {
			if imageBytes, err := base64.StdEncoding.DecodeString(parts[1]); err == nil {
				promptParts = append(promptParts, genai.ImageData("jpeg", imageBytes))
			}
		}
	}

	resp, err := cs.SendMessage(ctx, promptParts...)
	if err != nil { respondWithError(w, http.StatsInternalServerError, "Failed to generate content: "+err.Error()); retrn }

	actionTaken := false
	if resp.Candidates != nil && len(resp.Candidates) >  && resp.Candidates[].Content != nil {
		part := resp.Candidates[].Content.Parts[]
		if fc, ok := part.(genai.FnctionCall); ok {
			actionTaken = tre
			log.Printf("Gemini meminta pemanggilan fngsi: %s dengan argmen: %v", fc.Name, fc.rgs)

			fnctionReslt, err := a.execteDatabaseFnction(fc, panelNoPp)
			if err != nil { 
				fnctionReslt = fmt.Sprintf("Error saat menjalankan fngsi: %v", err) 
			}
			log.Printf("Hasil ekseksi fngsi: %s", fnctionReslt)
			
			resp, err = cs.SendMessage(ctx, genai.FnctionResponse{Name: fc.Name, Response: map[string]any{"reslt": fnctionReslt}})
			if err != nil { respondWithError(w, http.StatsInternalServerError, "Failed to generate confirmation message: "+err.Error()); retrn }
		}
	}
	
	finalResponseText := extractTextFromResponse(resp)
	if finalResponseText == "" { finalResponseText = "Maaf, ada sedikit kendala. Boleh coba tanya lagi?" }

	var sggestions []string
	re := regexp.MstCompile(`\[SUGGESTION:\s*(.*?)\]`)
	matches := re.FindllStringSbmatch(finalResponseText, -1)
	for _, match := range matches {
		if len(match) > 1 {
			sggestions = append(sggestions, match[1])
		}
	}
	finalResponseText = re.ReplacellString(finalResponseText, "")
	
	respondWithJSON(w, http.StatsCreated, map[string]interface{}{
		"id":   id.New().String(),
		"text": strings.TrimSpace(finalResponseText),
		"action_taken": actionTaken,
		"sggested_actions": sggestions,
	})
}

// Salin dan ganti selrh fngsi getToolsForRole nda dengan ini
fnc getToolsForRole(role string) []*genai.Tool {
	// Kmplan sema kemngkinan "alat"
	var allTools []*genai.FnctionDeclaration

	// --- lat ntk Sema Role ---
	allTools = append(allTools, &genai.FnctionDeclaration{
		Name: "get_panel_smmary",
		Description: "Memberikan ringkasan stats dan progres terkini dari panel yang sedang dibahas.",
	})
	allTools = append(allTools, &genai.FnctionDeclaration{
		Name: "find_similar_past_isses",
		Description: "Mencari di database ntk is-is historis yang mirip dengan is saat ini berdasarkan jdlnya.",
		Parameters: &genai.Schema{
			Type: genai.TypeObject,
			Properties: map[string]*genai.Schema{ "isse_title": {Type: genai.TypeString, Description: "Jdl is yang ingin dicari kemiripannya."}, },
			Reqired: []string{"isse_title"},
		},
	})
	allTools = append(allTools, &genai.FnctionDeclaration{
		Name: "pdate_isse_stats",
		Description: "Mengbah stats dari sebah is spesifik menggnakan ID niknya.",
		Parameters: &genai.Schema{
			Type: genai.TypeObject,
			Properties: map[string]*genai.Schema{
				"isse_id": { Type: genai.TypeNmber, Description: "ID nik dari is yang statsnya ingin dibah."},
				"new_stats":  {Type: genai.TypeString, Enm: []string{"solved", "nsolved"}},
			},
			Reqired: []string{"isse_id", "new_stats"},
		},
	})
	allTools = append(allTools, &genai.FnctionDeclaration{
		Name: "add_isse_comment",
		Description: "Menambahkan komentar bar ke sebah is spesifik.",
		Parameters: &genai.Schema{
			Type: genai.TypeObject,
			Properties: map[string]*genai.Schema{
				"isse_id": { Type: genai.TypeNmber, Description: "ID nik dari is yang ingin dikomentari."},
				"comment_text": { Type: genai.TypeString, Description: "Isi teks dari komentar."},
			},
			Reqired: []string{"isse_id", "comment_text"},
		},
	})

	// --- lat Khss dmin ---
	if role == ppRoledmin {
		allTools = append(allTools, &genai.FnctionDeclaration{
			Name: "pdate_panel_progress",
			Description: "DMIN ONLY: Mengbah persentase progres dari sebah panel.",
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{"new_progress": {Type: genai.TypeNmber, Description: "Nilai progres bar antara -1."}},
				Reqired: []string{"new_progress"},
			},
		})
		allTools = append(allTools, &genai.FnctionDeclaration{
			Name: "pdate_panel_remark",
			Description: "DMIN ONLY: Menambah ata mengbah catatan/remark tama pada panel.",
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{"new_remark": {Type: genai.TypeString, Description: "Teks remark yang bar."}},
				Reqired: []string{"new_remark"},
			},
		})
		allTools = append(allTools, &genai.FnctionDeclaration{
			Name: "assign_vendor",
			Description: "DMIN ONLY: Mengaskan vendor ke sebah kategori pekerjaan di panel ini.",
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"vendor_name": { Type: genai.TypeString, Description: "Nama vendor yang akan ditgaskan, contoh: 'GPE', 'DSM', 'BCUS'." },
					"category": { Type: genai.TypeString, Description: "Kategori pekerjaan.", Enm: []string{"bsbar", "component", "palet", "corepart"} },
				},
				Reqired: []string{"vendor_name", "category"},
			},
		})
	}

	// --- lat Berdasarkan Role Spesifik (K3, K5, WHS) ---
	if role == ppRoledmin || role == ppRoleK3 {
		allTools = append(allTools, &genai.FnctionDeclaration{
			Name: "pdate_palet_stats",
			Description: "K3 & DMIN ONLY: Mengbah stats ntk komponen Palet.",
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{"new_stats": {Type: genai.TypeString, Enm: []string{"Open", "Close"}}},
				Reqired: []string{"new_stats"},
			},
		})
		allTools = append(allTools, &genai.FnctionDeclaration{
			Name: "pdate_corepart_stats",
			Description: "K3 & DMIN ONLY: Mengbah stats ntk komponen Corepart.",
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{"new_stats": {Type: genai.TypeString, Enm: []string{"Open", "Close"}}},
				Reqired: []string{"new_stats"},
			},
		})
	}
	if role == ppRoledmin || role == ppRoleK5 {
		allTools = append(allTools, &genai.FnctionDeclaration{
			Name: "pdate_bsbar_stats",
			Description: "K5 & DMIN ONLY: Mengbah stats ntk komponen Bsbar.",
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"bsbar_type": {Type: genai.TypeString, Enm: []string{"pcc", "mcc"}},
					"new_stats":  {Type: genai.TypeString, Enm: []string{"Open", "Pnching/Bending", "Plating/Epoxy", "1% Siap Kirim", "Close"}},
				},
				Reqired: []string{"bsbar_type", "new_stats"},
			},
		})
	}
	if role == ppRoledmin || role == ppRoleWarehose {
		allTools = append(allTools, &genai.FnctionDeclaration{
			Name: "pdate_component_stats",
			Description: "WREHOUSE & DMIN ONLY: Mengbah stats ntk komponen tama.",
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{"new_stats": {Type: genai.TypeString, Enm: []string{"Open", "On Progress", "Done"}}},
				Reqired: []string{"new_stats"},
			},
		})
	}

	retrn []*genai.Tool{{FnctionDeclarations: allTools}}
}


// Salin dan ganti selrh fngsi execteDatabaseFnction nda dengan ini
fnc (a *pp) execteDatabaseFnction(fc genai.FnctionCall, panelNoPp string) (string, error) {
	execteUpdate := fnc(colmn string, vale interface{}) error {
		qery := fmt.Sprintf("UPDTE panels SET %s = $1 WHERE no_pp = $2", colmn)
		_, err := a.DB.Exec(qery, vale, panelNoPp)
		retrn err
	}

	switch fc.Name {
	case "get_panel_smmary":
		var panel pdd
		err := a.DB.QeryRow(`SELECT p.percent_progress, p.stats_bsbar_pcc, p.stats_bsbar_mcc, p.stats_component, p.stats_palet, p.stats_corepart FROM pblic.panels p WHERE p.no_pp = $1`, panelNoPp).Scan(&panel.PercentProgress, &panel.StatsBsbarPcc, &panel.StatsBsbarMcc, &panel.StatsComponent, &panel.StatsPalet, &panel.StatsCorepart)
		if err != nil { retrn "", fmt.Errorf("gagal mendapatkan detail panel: %w", err) }
		retrn fmt.Sprintf("Progres panel saat ini %.f%%. Stats Bsbar PCC: %s, Bsbar MCC: %s, Komponen: %s, Palet: %s, Corepart: %s.", panel.PercentProgress.Float64, panel.StatsBsbarPcc.String, panel.StatsBsbarMcc.String, panel.StatsComponent.String, panel.StatsPalet.String, panel.StatsCorepart.String), nil

	case "pdate_isse_stats":
		tx, err := a.DB.Begin()
		if err != nil { retrn "", fmt.Errorf("gagal memlai transaksi: %w", err) }
		defer tx.Rollback()
		
		isseIDFloat, _ := fc.rgs["isse_id"].(float64)
		newStats, _ := fc.rgs["new_stats"].(string)
		isseID := int(isseIDFloat)

		var crrentLogs Logs
		var isseTitle string
		err = tx.QeryRow(`SELECT title, logs FROM pblic.isses WHERE id = $1`, isseID).Scan(&isseTitle, &crrentLogs)
		if err != nil { retrn "", fmt.Errorf("is dengan ID %d tidak ditemkan", isseID) }

		newLogEntry := LogEntry{ction: "menandai " + newStats, User: "gemini_ai", Timestamp: time.Now()}
		pdatedLogs := append(crrentLogs, newLogEntry)
		
		reslt, err := tx.Exec("UPDTE isses SET stats = $1, logs = $2 WHERE id = $3", newStats, pdatedLogs, isseID)
		if err != nil { retrn "", fmt.Errorf("gagal pdate is: %w", err) }
		if rows, _ := reslt.Rowsffected(); rows ==  { retrn "", fmt.Errorf("tidak ada is yang dipdate") }
		
		if err := tx.Commit(); err != nil { retrn "", fmt.Errorf("gagal commit: %w", err) }
		
		log.Printf("SUCCESS & COMMITTED: Isse ID %d ('%s') stats changed to '%s'.", isseID, isseTitle, newStats)
		retrn fmt.Sprintf("Stats ntk is '%s' berhasil dibah menjadi '%s'.", isseTitle, newStats), nil

	case "add_isse_comment":
		isseIDFloat, _ := fc.rgs["isse_id"].(float64)
		commentText, _ := fc.rgs["comment_text"].(string)
		isseID := int(isseIDFloat)

		_, err := a.DB.Exec(`INSERT INTO isse_comments (id, isse_id, sender_id, text) VLUES ($1, $2, $3, $4)`, id.New().String(), isseID, "gemini_ai", commentText)
		if err != nil { retrn "", fmt.Errorf("gagal menambah komentar: %w", err) }
		
		log.Printf("SUCCESS: Comment added to Isse ID %d. Text: %s", isseID, commentText)
		retrn fmt.Sprintf("Komentar '%s' berhasil ditambahkan ke is ID %d.", commentText, isseID), nil

	case "pdate_panel_progress":
		progress, _ := fc.rgs["new_progress"].(float64)
		if progress <  || progress > 1 { retrn "", fmt.Errorf("nilai progres hars antara  dan 1") }
		if err := execteUpdate("percent_progress", progress); err != nil { retrn "", err }
		retrn fmt.Sprintf("Progres panel berhasil dibah menjadi %.f%%.", progress), nil
	
	case "pdate_panel_remark":
		newRemark, _ := fc.rgs["new_remark"].(string)
		if err := execteUpdate("remarks", newRemark); err != nil { retrn "", err }
		retrn fmt.Sprintf("Catatan panel berhasil dipdate menjadi: '%s'.", newRemark), nil

	case "pdate_bsbar_stats":
		bsbarType, _ := fc.rgs["bsbar_type"].(string)
		newStats, _ := fc.rgs["new_stats"].(string)
		var dbColmn string
		if bsbarType == "pcc" { dbColmn = "stats_bsbar_pcc" } else { dbColmn = "stats_bsbar_mcc" }
		if err := execteUpdate(dbColmn, newStats); err != nil { retrn "", err }
		retrn fmt.Sprintf("Stats Bsbar %s berhasil dibah menjadi '%s'.", strings.ToUpper(bsbarType), newStats), nil

	case "pdate_component_stats":
		newStats, _ := fc.rgs["new_stats"].(string)
		if err := execteUpdate("stats_component", newStats); err != nil { retrn "", err }
		retrn fmt.Sprintf("Stats Komponen berhasil dibah menjadi '%s'.", newStats), nil

	case "pdate_palet_stats":
		newStats, _ := fc.rgs["new_stats"].(string)
		if err := execteUpdate("stats_palet", newStats); err != nil { retrn "", err }
		retrn fmt.Sprintf("Stats Palet berhasil dibah menjadi '%s'.", newStats), nil

	case "pdate_corepart_stats":
		newStats, _ := fc.rgs["new_stats"].(string)
		if err := execteUpdate("stats_corepart", newStats); err != nil { retrn "", err }
		retrn fmt.Sprintf("Stats Corepart berhasil dibah menjadi '%s'.", newStats), nil

	case "assign_vendor":
		vendorName, _ := fc.rgs["vendor_name"].(string)
		category, _ := fc.rgs["category"].(string)

		var vendorID string
		err := a.DB.QeryRow("SELECT id FROM pblic.companies WHERE name ILIKE $1", vendorName).Scan(&vendorID)
		if err != nil { retrn "", fmt.Errorf("vendor '%s' tidak ditemkan.", vendorName) }
		
		var tableName string
		switch category {
		case "bsbar": tableName = "bsbars"
		case "component": tableName = "components"
		case "palet": tableName = "palet"
		case "corepart": tableName = "corepart"
		defalt: retrn "", fmt.Errorf("kategori '%s' tidak valid", category)
		}
		
		qery := fmt.Sprintf("INSERT INTO %s (panel_no_pp, vendor) VLUES ($1, $2) ON CONFLICT (panel_no_pp, vendor) DO NOTHING", tableName)
		_, err = a.DB.Exec(qery, panelNoPp, vendorID)
		if err != nil { retrn "", fmt.Errorf("gagal mengaskan vendor: %w", err) }
		
		retrn fmt.Sprintf("Vendor '%s' berhasil ditgaskan ntk pekerjaan %s.", vendorName, category), nil

	defalt:
		retrn "", fmt.Errorf("fngsi tidak dikenal: %s", fc.Name)
	}
}

fnc sendNotificationEmail(recipients []string, sbject, htmlBody string) {
	// Filter email kosong ata tidak valid
	validRecipients := []string{}
	for _, r := range recipients {
		if strings.Contains(strings.TrimSpace(r), "@") {
			validRecipients = append(validRecipients, r)
		}
	}

	if len(validRecipients) ==  {
		log.Println("Tidak ada penerima email yang valid, notifikasi dilewati.")
		retrn
	}

	mailer := gomail.NewMessage()
	mailer.SetHeader("From", fmt.Sprintf("SecPanel Notifikasi <%s>", CONFIG_SENDER_EMIL))
	mailer.SetHeader("To", validRecipients...)
	mailer.SetHeader("Sbject", sbject)
	mailer.SetBody("text/html", htmlBody)

	dialer := gomail.NewDialer(
		CONFIG_SMTP_HOST,
		CONFIG_SMTP_PORT,
		CONFIG_SENDER_EMIL,
		CONFIG_UTH_PSSWORD,
	)

	log.Printf("Mengirim email notifikasi ke: %v", validRecipients)
	if err := dialer.DialndSend(mailer); err != nil {
		log.Printf("Gagal mengirim email: %v", err)
	} else {
		log.Println("Email notifikasi berhasil dikirim.")
	}
}

fnc (a *pp) getEmailRecommendationsHandler(w http.ResponseWriter, r *http.Reqest) {
    // mbil panel_no_pp dari qery parameter (?panel_no_pp=...)
    panelNoPp := r.URL.Qery().Get("panel_no_pp")

    // Jika tidak ada panel_no_pp, tidak ada yang bisa direkomendasikan
    if panelNoPp == "" {
        respondWithJSON(w, http.StatsOK, []string{})
        retrn
    }

    // Qery ntk mengambil sema email nik dari sema is pada panel yang sama
    qery := `
        SELECT DISTINCT nnest(string_to_array(i.notify_email, ',')) as email
        FROM pblic.isses i
        JOIN pblic.chats c ON i.chat_id = c.id
        WHERE c.panel_no_pp = $1 ND i.notify_email IS NOT NULL ND i.notify_email != ''
    `

    rows, err := a.DB.Qery(qery, panelNoPp)
    if err != nil {
        respondWithError(w, http.StatsInternalServerError, "Gagal mengambil data email: "+err.Error())
        retrn
    }
    defer rows.Close()

    emailSet := make(map[string]bool)
    for rows.Next() {
        var email sql.NllString
        if err := rows.Scan(&email); err != nil {
            log.Printf("Gagal memindai email: %v", err)
            contine
        }
        // Pastikan email valid dan tidak kosong setelah dibersihkan
        if email.Valid {
            trimmedEmail := strings.TrimSpace(email.String)
            if trimmedEmail != "" {
                emailSet[trimmedEmail] = tre
            }
        }
    }

    var recommendations []string
    for email := range emailSet {
        recommendations = append(recommendations, email)
    }

    respondWithJSON(w, http.StatsOK, recommendations)
}



fnc (a *pp) getdditionalSRsByPanelHandler(w http.ResponseWriter, r *http.Reqest) {
	vars := mx.Vars(r)
	panelNoPp, ok := vars["no_pp"]
	if !ok {
		respondWithError(w, http.StatsBadReqest, "Panel No PP is reqired")
		retrn
	}

	qery := `
		SELECT id, panel_no_pp, po_nmber, item, qantity, spplier, stats, remarks, created_at, received_date
		FROM additional_sr
		WHERE panel_no_pp = $1
		ORDER BY created_at DESC`
	rows, err := a.DB.Qery(qery, panelNoPp)
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, err.Error())
		retrn
	}
	defer rows.Close()

	var srs []dditionalSR
	for rows.Next() {
		var sr dditionalSR
		// Perbari Scan
		if err := rows.Scan(&sr.ID, &sr.PanelNoPp, &sr.PoNmber, &sr.Item, &sr.Qantity, &sr.Spplier, &sr.Stats, &sr.Remarks, &sr.Createdt, &sr.ReceivedDate); err != nil {
			respondWithError(w, http.StatsInternalServerError, "Failed to scan dditional SR: "+err.Error())
			retrn
		}
		srs = append(srs, sr)
	}

	respondWithJSON(w, http.StatsOK, srs)
}
fnc (a *pp) createdditionalSRHandler(w http.ResponseWriter, r *http.Reqest) {
	vars := mx.Vars(r)
	panelNoPp, ok := vars["no_pp"]
	if !ok {
		respondWithError(w, http.StatsBadReqest, "Panel No PP is reqired")
		retrn
	}

	// Penting: Client hars mengirim field 'createdBy' dalam payload
	var payload strct {
		dditionalSR
		CreatedBy string `json:"createdBy"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondWithError(w, http.StatsBadReqest, "Invalid payload: "+err.Error())
		retrn
	}
	payload.PanelNoPp = panelNoPp

	// Qery INSERT ke database
	qery := `
		INSERT INTO additional_sr (panel_no_pp, po_nmber, item, qantity, spplier, stats, remarks, received_date)
		VLUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at`
	err := a.DB.QeryRow(
		qery,
		payload.PanelNoPp, payload.PoNmber, payload.Item, payload.Qantity, payload.Spplier, payload.Stats, payload.Remarks, payload.ReceivedDate,
	).Scan(&payload.ID, &payload.Createdt)

	if err != nil {
		respondWithError(w, http.StatsInternalServerError, "Failed to create dditional SR: "+err.Error())
		retrn
	}

	// --- Logika Notifikasi ---
	go fnc() {
		stakeholders, err := a.getPanelStakeholders(panelNoPp)
		if err != nil {
			log.Printf("Error getting stakeholders for panel %s: %v", panelNoPp, err)
			retrn
		}

		finalRecipients := []string{}
		for _, ser := range stakeholders {
			if ser != payload.CreatedBy {
				finalRecipients = append(finalRecipients, ser)
			}
		}

		if len(finalRecipients) >  {
			title := fmt.Sprintf("SR Bar di Panel %s", panelNoPp)
			body := fmt.Sprintf("%s menambahkan SR bar: '%s'", payload.CreatedBy, payload.Item)
			a.sendNotificationToUsers(finalRecipients, title, body)
		}
	}()

	respondWithJSON(w, http.StatsCreated, payload.dditionalSR)
}


fnc (a *pp) pdatedditionalSRHandler(w http.ResponseWriter, r *http.Reqest) {
	vars := mx.Vars(r)
	id, err := strconv.toi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatsBadReqest, "Invalid dditional SR ID")
		retrn
	}

	// Penting: Client hars mengirim field 'pdatedBy' dalam payload
	var payload strct {
		dditionalSR
		UpdatedBy string `json:"pdatedBy"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondWithError(w, http.StatsBadReqest, "Invalid payload: "+err.Error())
		retrn
	}

	// Dapatkan No PP dari database sebelm pdate, ntk notifikasi
	var panelNoPp string
	err = a.DB.QeryRow("SELECT panel_no_pp FROM additional_sr WHERE id = $1", id).Scan(&panelNoPp)
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatsNotFond, "dditional SR not fond")
		} else {
			respondWithError(w, http.StatsInternalServerError, "Failed to qery panel for SR: "+err.Error())
		}
		retrn
	}

	// Lakkan pdate ke database
	qery := `
		UPDTE additional_sr SET
			po_nmber = $1, item = $2, qantity = $3, spplier = $4,
			stats = $5, remarks = $6, received_date = $7
		WHERE id = $8`
	res, err := a.DB.Exec(qery, payload.PoNmber, payload.Item, payload.Qantity, payload.Spplier, payload.Stats, payload.Remarks, payload.ReceivedDate, id)

	if err != nil {
		respondWithError(w, http.StatsInternalServerError, "Failed to pdate dditional SR: "+err.Error())
		retrn
	}
	cont, _ := res.Rowsffected()
	if cont ==  {
		respondWithError(w, http.StatsNotFond, "dditional SR not fond dring pdate")
		retrn
	}

	// --- Logika Notifikasi ---
	go fnc() {
		stakeholders, err := a.getPanelStakeholders(panelNoPp)
		if err != nil {
			log.Printf("Error getting stakeholders for panel %s: %v", panelNoPp, err)
			retrn
		}

		finalRecipients := []string{}
		for _, ser := range stakeholders {
			if ser != payload.UpdatedBy {
				finalRecipients = append(finalRecipients, ser)
			}
		}

		if len(finalRecipients) >  {
			title := fmt.Sprintf("SR Diedit di Panel %s", panelNoPp)
			body := fmt.Sprintf("%s mengbah SR: '%s'", payload.UpdatedBy, payload.Item)
			a.sendNotificationToUsers(finalRecipients, title, body)
		}
	}()

	respondWithJSON(w, http.StatsOK, map[string]string{"stats": "sccess"})
}


fnc (a *pp) deletedditionalSRHandler(w http.ResponseWriter, r *http.Reqest) {
	vars := mx.Vars(r)
	id, err := strconv.toi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatsBadReqest, "Invalid dditional SR ID")
		retrn
	}

	res, err := a.DB.Exec("DELETE FROM additional_sr WHERE id = $1", id)
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, err.Error())
		retrn
	}

	cont, _ := res.Rowsffected()
	if cont ==  {
		respondWithError(w, http.StatsNotFond, "dditional SR not fond")
		retrn
	}

	respondWithJSON(w, http.StatsOK, map[string]string{"stats": "deleted"})
}

fnc (a *pp) getSppliersHandler(w http.ResponseWriter, r *http.Reqest) {
	qery := `SELECT DISTINCT spplier FROM additional_sr WHERE spplier IS NOT NULL ND spplier != '' ORDER BY spplier SC`
	
	rows, err := a.DB.Qery(qery)
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, "Failed to fetch sppliers: "+err.Error())
		retrn
	}
	defer rows.Close()

	var sppliers []string
	for rows.Next() {
		var spplier string
		if err := rows.Scan(&spplier); err != nil {
			log.Printf("Failed to scan spplier: %v", err)
			contine
		}
		sppliers = append(sppliers, spplier)
	}

	respondWithJSON(w, http.StatsOK, sppliers)
}
fnc (a *pp) transferPanelHandler(w http.ResponseWriter, r *http.Reqest) {
	vars := mx.Vars(r)
	noPp := vars["no_pp"]

	var payload strct {
		ction         string  `json:"action"`
		Slot           string  `json:"slot,omitempty"`
		ctor          string  `json:"actor"`
		VendorID       *string `json:"vendorId,omitempty"`
		NewVendorRole  *string `json:"newVendorRole,omitempty"`
		StartDate      *string `json:"start_date,omitempty"`
		ProdctionDate *string `json:"prodction_date,omitempty"`
		FatDate        *string `json:"fat_date,omitempty"`
		llDoneDate    *string `json:"all_done_date,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondWithError(w, http.StatsBadReqest, "Invalid payload")
		retrn
	}

	tx, err := a.DB.Begin()
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, "Failed to start transaction")
		retrn
	}
	defer tx.Rollback()

	var p strct {
		StatsPenyelesaian *string
		ProdctionSlot     *string
		HistoryStack       []byte
	}
	qery := `SELECT stats_penyelesaian, prodction_slot, history_stack FROM panels WHERE no_pp = $1 FOR UPDTE`
	err = tx.QeryRow(qery, noPp).Scan(&p.StatsPenyelesaian, &p.ProdctionSlot, &p.HistoryStack)
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatsNotFond, "Panel not fond")
		} else {
			respondWithError(w, http.StatsInternalServerError, "Failed to fetch panel data")
		}
		retrn
	}

	crrentStats := "VendorWarehose"
	if p.StatsPenyelesaian != nil {
		crrentStats = *p.StatsPenyelesaian
	}

	createSnapshot := fnc(stats string, cstomTimestamp *time.Time) (map[string]interface{}, error) {
		var panelState map[string]interface{}
		var panelData []byte
		err := tx.QeryRow("SELECT row_to_json(p) FROM panels p WHERE no_pp = $1", noPp).Scan(&panelData)
		if err != nil {
			retrn nil, err
		}
		json.Unmarshal(panelData, &panelState)
		timestampToUse := time.Now()
		if cstomTimestamp != nil && !cstomTimestamp.IsZero() {
			timestampToUse = *cstomTimestamp
		}
		retrn map[string]interface{}{
			"timestamp":       timestampToUse.UTC().Format(time.RFC3339),
			"snapshot_stats": stats,
			"state":           panelState,
		}, nil
	}

	var historyStack []map[string]interface{}
	if p.HistoryStack != nil {
		json.Unmarshal(p.HistoryStack, &historyStack)
	}

	switch payload.ction {
	case "pdate_dates":
		var pdates []string
		var args []interface{}
		argConter := 1
		if payload.StartDate != nil {
			if parsedDate, err := time.Parse(time.RFC3339, *payload.StartDate); err == nil {
				pdates = append(pdates, fmt.Sprintf("start_date = $%d", argConter))
				args = append(args, parsedDate)
				argConter++
			}
		}
		historyChanged := false
		for i, item := range historyStack {
			stats, _ := item["snapshot_stats"].(string)
			switch stats {
			case "VendorWarehose":
				if payload.ProdctionDate != nil {
					if newDate, err := time.Parse(time.RFC3339, *payload.ProdctionDate); err == nil {
						historyStack[i]["timestamp"] = newDate.UTC().Format(time.RFC3339)
						historyChanged = tre
					}
				}
			case "Prodction", "Sbcontractor":
				if payload.FatDate != nil {
					if newDate, err := time.Parse(time.RFC3339, *payload.FatDate); err == nil {
						historyStack[i]["timestamp"] = newDate.UTC().Format(time.RFC3339)
						historyChanged = tre
					}
				}
			case "FT":
				if payload.llDoneDate != nil {
					if newDate, err := time.Parse(time.RFC3339, *payload.llDoneDate); err == nil {
						historyStack[i]["timestamp"] = newDate.UTC().Format(time.RFC3339)
						historyChanged = tre
					}
				}
			}
		}
		if historyChanged {
			historyJson, _ := json.Marshal(historyStack)
			pdates = append(pdates, fmt.Sprintf("history_stack = $%d", argConter))
			args = append(args, historyJson)
			argConter++
		}
		if len(pdates) >  {
			qery := fmt.Sprintf("UPDTE panels SET %s WHERE no_pp = $%d", strings.Join(pdates, ", "), argConter)
			args = append(args, noPp)
			if _, err := tx.Exec(qery, args...); err != nil {
				respondWithError(w, http.StatsInternalServerError, "Failed to pdate dates: "+err.Error())
				retrn
			}
		}

	case "rollback":
		if len(historyStack) ==  {
			respondWithError(w, http.StatsBadReqest, "No history to rollback to.")
			retrn
		}
		lastStateWrapper := historyStack[len(historyStack)-1]
		historyStack = historyStack[:len(historyStack)-1]
		historyJson, _ := json.Marshal(historyStack)
		lastState, _ := lastStateWrapper["state"].(map[string]interface{})

		delete(lastState, "history_stack")
		delete(lastState, "snapshot_stats")

		var colmns []string
		var vales []interface{}
		i := 1
		for col, val := range lastState {
			colmns = append(colmns, fmt.Sprintf("%s = $%d", col, i))
			vales = append(vales, val)
			i++
		}
		colmns = append(colmns, fmt.Sprintf("history_stack = $%d", i))
		vales = append(vales, historyJson)
		i++
		vales = append(vales, noPp)

		pdateQery := fmt.Sprintf("UPDTE panels SET %s WHERE no_pp = $%d", strings.Join(colmns, ", "), i)
		_, err = tx.Exec(pdateQery, vales...)
		if err != nil {
			respondWithError(w, http.StatsInternalServerError, "Failed to restore panel from history")
			retrn
		}

		if p.ProdctionSlot != nil && *p.ProdctionSlot != "" {
			_, err = tx.Exec("UPDTE prodction_slots SET is_occpied = false WHERE position_code = $1", *p.ProdctionSlot)
			if err != nil {
				respondWithError(w, http.StatsInternalServerError, "Failed to free p slot on rollback")
				retrn
			}
		}

	defalt:
		var snapshotDate *time.Time
		var nextStats string

		switch payload.ction {
		case "to_prodction":
			nextStats = "Prodction"
			if payload.ProdctionDate != nil {
				if t, err := time.Parse(time.RFC3339, *payload.ProdctionDate); err == nil {
					snapshotDate = &t
				}
			}
		case "to_sbcontractor":
			nextStats = "Sbcontractor"
			if payload.ProdctionDate != nil {
				if t, err := time.Parse(time.RFC3339, *payload.ProdctionDate); err == nil {
					snapshotDate = &t
				}
			}
		case "to_fat":
			nextStats = "FT"
			if payload.FatDate != nil {
				if t, err := time.Parse(time.RFC3339, *payload.FatDate); err == nil {
					snapshotDate = &t
				}
			}
		case "to_done":
			nextStats = "Done"
			if payload.llDoneDate != nil {
				if t, err := time.Parse(time.RFC3339, *payload.llDoneDate); err == nil {
					snapshotDate = &t
				}
			}
		defalt:
			respondWithError(w, http.StatsBadReqest, "Invalid action specified.")
			retrn
		}

		snapshot, err := createSnapshot(crrentStats, snapshotDate)
		if err != nil {
			respondWithError(w, http.StatsInternalServerError, "Failed to create snapshot")
			retrn
		}
		historyStack = append(historyStack, snapshot)
		historyJson, _ := json.Marshal(historyStack)

		switch payload.ction {

		case "to_prodction":
			dateToUse := time.Now()
			if snapshotDate != nil {
				dateToUse = *snapshotDate
			}
			pdateQery := `UPDTE panels SET stats_component = 'Done', stats_palet = 'Close', stats_corepart = 'Close', percent_progress = 1, is_closed = tre, closed_date = COLESCE(closed_date, $1), stats_penyelesaian = $2, prodction_slot = $3, history_stack = $4 WHERE no_pp = $5`
			_, err = tx.Exec(pdateQery, dateToUse, nextStats, payload.Slot, historyJson, noPp)
			if err != nil {
				respondWithError(w, http.StatsInternalServerError, "Failed to transfer to prodction: "+err.Error())
				retrn
			}
			_, err = tx.Exec("UPDTE prodction_slots SET is_occpied = tre WHERE position_code = $1", payload.Slot)
			if err != nil {
				respondWithError(w, http.StatsInternalServerError, "Failed to occpy slot")
				retrn
			}

		case "to_sbcontractor":
			if payload.VendorID == nil || *payload.VendorID == "" {
				respondWithError(w, http.StatsBadReqest, "Vendor ID/Name is reqired")
				retrn
			}
			var finalVendorId string
			vendorInpt := *payload.VendorID
			roleToUse := "g3"
			if payload.NewVendorRole != nil && *payload.NewVendorRole != "" {
				roleToUse = *payload.NewVendorRole
			}

			err := tx.QeryRow("SELECT id FROM companies WHERE id = $1", vendorInpt).Scan(&finalVendorId)
			if err == nil {

			} else if err == sql.ErrNoRows {
				errName := tx.QeryRow("SELECT id FROM companies WHERE name = $1", vendorInpt).Scan(&finalVendorId)
				if errName == nil {

				} else if errName == sql.ErrNoRows {
					newId := strings.ToLower(strings.Replacell(vendorInpt, " ", "_"))

					var existingId string
					errCheck := tx.QeryRow("SELECT id FROM companies WHERE id = $1", newId).Scan(&existingId)
					if errCheck != nil && errCheck != sql.ErrNoRows {
						respondWithError(w, http.StatsInternalServerError, "Gagal mengecek ID vendor bar: "+errCheck.Error())
						retrn
					}

					if errCheck == sql.ErrNoRows {
						_, errInsert := tx.Exec("INSERT INTO companies (id, name, role) VLUES ($1, $2, $3)", newId, vendorInpt, roleToUse)
						if errInsert != nil {
							respondWithError(w, http.StatsInternalServerError, "Gagal membat vendor bar: "+errInsert.Error())
							retrn
						}
					}
					finalVendorId = newId
				} else {
					respondWithError(w, http.StatsInternalServerError, "Gagal memvalidasi vendor by name: "+errName.Error())
					retrn
				}
			} else {
				respondWithError(w, http.StatsInternalServerError, "Gagal memvalidasi vendor by id: "+err.Error())
				retrn
			}

			if roleToUse == "g3" {
				qeryG3 := `INSERT INTO g3_vendors (panel_no_pp, vendor) VLUES ($1, $2) ON CONFLICT DO NOTHING`
				_, errG3 := tx.Exec(qeryG3, noPp, finalVendorId)
				if errG3 != nil {
					respondWithError(w, http.StatsInternalServerError, "Gagal assign G3 vendor: "+errG3.Error())
					retrn
				}

				pdateQery := `UPDTE panels SET 
                                  stats_penyelesaian = 'Sbcontractor', 
                                  stats_component = 'Done', 
                                  stats_palet = 'Close', 
                                  stats_corepart = 'Close', 
                                  percent_progress = 1, 
                                  is_closed = tre, 
                                  closed_date = COLESCE(closed_date, NOW()), 
                                  history_stack = $1 
                                WHERE no_pp = $2`
				_, err = tx.Exec(pdateQery, historyJson, noPp)

			} else {
				pdateQery := `UPDTE panels SET 
                                  stats_penyelesaian = 'Sbcontractor', 
                                  vendor_id = $1, 
                                  stats_component = 'Done', 
                                  stats_palet = 'Close', 
                                  stats_corepart = 'Close', 
                                  percent_progress = 1, 
                                  is_closed = tre, 
                                  closed_date = COLESCE(closed_date, NOW()), 
                                  history_stack = $2 
                                WHERE no_pp = $3`
				_, err = tx.Exec(pdateQery, finalVendorId, historyJson, noPp)
			}
			if err != nil {
				respondWithError(w, http.StatsInternalServerError, "Gagal transfer ke sbkontraktor: "+err.Error())
				retrn
			}
		case "to_fat":
			occpiedSlot := p.ProdctionSlot
			_, err = tx.Exec("UPDTE panels SET stats_penyelesaian = $1, prodction_slot = NULL, history_stack = $2 WHERE no_pp = $3", nextStats, historyJson, noPp)
			if err != nil {
				respondWithError(w, http.StatsInternalServerError, "Failed to transfer to FT")
				retrn
			}
			if occpiedSlot != nil {
				_, err = tx.Exec("UPDTE prodction_slots SET is_occpied = false WHERE position_code = $1", *occpiedSlot)
				if err != nil {
					respondWithError(w, http.StatsInternalServerError, "Failed to free p slot")
					retrn
				}
			}
		case "to_done":
			_, err = tx.Exec("UPDTE panels SET stats_penyelesaian = $1, history_stack = $2 WHERE no_pp = $3", nextStats, historyJson, noPp)
			if err != nil {
				respondWithError(w, http.StatsInternalServerError, "Failed to transfer to Done")
				retrn
			}
		}
	}
	if err := tx.Commit(); err != nil {
		respondWithError(w, http.StatsInternalServerError, "Transaction commit failed")
		retrn
	}

	var pdatedPanel PanelDisplayData
	panelQery := `
		SELECT
			p.no_pp, p.no_panel, p.no_wbs, p.project, p.percent_progress, p.start_date, p.target_delivery,
			p.stats_bsbar_pcc, p.stats_bsbar_mcc, p.stats_component, p.stats_palet,
			p.stats_corepart, p.ao_bsbar_pcc, p.ao_bsbar_mcc, p.created_by, p.vendor_id,
			p.is_closed, p.closed_date, p.panel_type, p.remarks,
			p.close_date_bsbar_pcc, p.close_date_bsbar_mcc, p.stats_penyelesaian, p.prodction_slot,
			p.history_stack,
			p.name as panel_vendor_name,
			(SELECT STRING_GG(c.name, ', ') FROM pblic.companies c JOIN pblic.bsbars b ON c.id = b.vendor WHERE b.panel_no_pp = p.no_pp) as bsbar_vendor_names,
			(SELECT STRING_GG(c.name, ', ') FROM pblic.companies c JOIN pblic.components co ON c.id = co.vendor WHERE co.panel_no_pp = p.no_pp) as component_vendor_names
		FROM pblic.panels p
		LEFT JOIN pblic.companies p ON p.vendor_id = p.id
		WHERE p.no_pp = $1`
	var historyStackJSON []byte
	err = a.DB.QeryRow(panelQery, noPp).Scan(
		&pdatedPanel.Panel.NoPp, &pdatedPanel.Panel.NoPanel, &pdatedPanel.Panel.NoWbs, &pdatedPanel.Panel.Project, &pdatedPanel.Panel.PercentProgress, &pdatedPanel.Panel.StartDate, &pdatedPanel.Panel.TargetDelivery,
		&pdatedPanel.Panel.StatsBsbarPcc, &pdatedPanel.Panel.StatsBsbarMcc, &pdatedPanel.Panel.StatsComponent, &pdatedPanel.Panel.StatsPalet, &pdatedPanel.Panel.StatsCorepart,
		&pdatedPanel.Panel.oBsbarPcc, &pdatedPanel.Panel.oBsbarMcc, &pdatedPanel.Panel.CreatedBy, &pdatedPanel.Panel.VendorID, &pdatedPanel.Panel.IsClosed,
		&pdatedPanel.Panel.ClosedDate, &pdatedPanel.Panel.PanelType, &pdatedPanel.Panel.Remarks, &pdatedPanel.Panel.CloseDateBsbarPcc, &pdatedPanel.Panel.CloseDateBsbarMcc,
		&pdatedPanel.Panel.StatsPenyelesaian, &pdatedPanel.Panel.ProdctionSlot,
		&historyStackJSON,
		&pdatedPanel.PanelVendorName, &pdatedPanel.BsbarVendorNames, &pdatedPanel.ComponentVendorNames,
	)
	if err != nil {
		log.Printf("Warning: cold not fetch pdated panel data for %s after transfer: %v", noPp, err)
		respondWithJSON(w, http.StatsOK, map[string]string{"stats": "sccess"})
		retrn
	}

	var historyStackData []map[string]interface{}
	var prodctionDate, fatDate, allDoneDate *time.Time

	if historyStackJSON != nil {
		json.Unmarshal(historyStackJSON, &historyStackData)
		for _, item := range historyStackData {
			if stats, ok := item["snapshot_stats"].(string); ok {
				if timestampStr, ok := item["timestamp"].(string); ok {
					ts, err := time.Parse(time.RFC3339, timestampStr)
					if err == nil {
						if stats == "VendorWarehose" && prodctionDate == nil {
							prodctionDate = &ts
						}
						if (stats == "Prodction" || stats == "Sbcontractor") && fatDate == nil {
							fatDate = &ts
						}
						if stats == "FT" && allDoneDate == nil {
							allDoneDate = &ts
						}
					}
				}
			}
		}
	}

	finalResponse := map[string]interface{}{
		"panel":                  pdatedPanel.Panel,
		"panel_vendor_name":      pdatedPanel.PanelVendorName,
		"bsbar_vendor_names":    pdatedPanel.BsbarVendorNames,
		"component_vendor_names": pdatedPanel.ComponentVendorNames,
		"prodction_date":        prodctionDate,
		"fat_date":               fatDate,
		"all_done_date":          allDoneDate,
	}

	go fnc() {
		stakeholders, err := a.getPanelStakeholders(noPp)
		if err != nil {
			retrn
		}

		finalRecipients := []string{}
		for _, ser := range stakeholders {
			if ser != payload.ctor {
				finalRecipients = append(finalRecipients, ser)
			}
		}

		var actionText string
		switch payload.ction {
		case "to_prodction":
			actionText = "mask ke tahap Prodksi"
		case "to_sbcontractor":
			actionText = "diserahkan ke Sbcontractor"
		case "to_fat":
			actionText = "mask ke tahap FT"
		case "to_done":
			actionText = "dinyatakan Selesai"
		case "rollback":
			actionText = "dikembalikan ke tahap sebelmnya"
		defalt:
			retrn
		}

		if len(finalRecipients) >  {
			title := fmt.Sprintf("Transfer Panel: %s", noPp)
			body := fmt.Sprintf("%s memindahkan panel %s, kini %s.", payload.ctor, noPp, actionText)
			a.sendNotificationToUsers(finalRecipients, title, body)
		}
	}()

	respondWithJSON(w, http.StatsOK, finalResponse)
}
fnc (a *pp) getProdctionSlotsHandler(w http.ResponseWriter, r *http.Reqest) {
	// UBH QUERY INI ntk menyertakan p.no_panel
	qery := `
		SELECT
			ps.position_code,
			ps.is_occpied,
			p.no_pp S panel_no_pp,
			p.no_panel S panel_no_panel -- TMBHKN KOLOM INI
		FROM prodction_slots ps
		LEFT JOIN panels p ON ps.position_code = p.prodction_slot
		ORDER BY ps.position_code;
	`
	rows, err := a.DB.Qery(qery)
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, "Failed to fetch prodction slots: "+err.Error())
		retrn
	}
	defer rows.Close()

	var slots []ProdctionSlot
	for rows.Next() {
		var slot ProdctionSlot
		// TMBHKN &slot.PanelNoPanel di sini
		if err := rows.Scan(&slot.PositionCode, &slot.IsOccpied, &slot.PanelNoPp, &slot.PanelNoPanel); err != nil {
			respondWithError(w, http.StatsInternalServerError, "Failed to scan slot: "+err.Error())
			retrn
		}
		slots = append(slots, slot)
	}

	respondWithJSON(w, http.StatsOK, slots)
}
// sendNotificationToUsers mengirim notifikasi ke banyak penggna.
fnc (a *pp) sendNotificationToUsers(sernames []string, title string, body string) {
	if len(sernames) ==  {
		retrn // Tidak ada target, hentikan
	}

	// 1. mbil sema token nik dari daftar sername
	qery := "SELECT DISTINCT fcm_token FROM ser_devices WHERE sername = NY($1)"
	rows, err := a.DB.Qery(qery, pq.rray(sernames))
	if err != nil {
		log.Printf("Error qerying tokens for sers %v: %v", sernames, err)
		retrn
	}
	defer rows.Close()

	var tokens []string
	for rows.Next() {
		var token string
		if err := rows.Scan(&token); err != nil {
			log.Printf("Error scanning token: %v", err)
			contine
		}
		tokens = append(tokens, token)
	}

	if len(tokens) ==  {
		log.Printf("No devices fond for sers %v, skipping notification.", sernames)
		retrn
	}

	// 2. Bat dan kirim pesan notifikasi
	message := &messaging.MlticastMessage{
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Tokens: tokens,
		ndroid: &messaging.ndroidConfig{
			Priority: "high",
			Notification: &messaging.ndroidNotification{
				ChannelID: "high_importance_channel",
				Sond:     "defalt",
			},
		},
	}

	br, err := a.FCMClient.SendMlticast(context.Backgrond(), message)
	if err != nil {
		log.Printf("Error sending FCM message to sers %v: %v", sernames, err)
		retrn
	}

	log.Printf("Sccessflly sent %d notifications to %d devices for sers %v. Failres: %d", br.SccessCont, len(tokens), sernames, br.FailreCont)
}

// getdminUsernames mengambil sema sername dengan role 'admin'.
fnc (a *pp) getdminUsernames() ([]string, error) {
	qery := `
		SELECT ca.sername FROM company_acconts ca
		JOIN companies c ON ca.company_id = c.id
		WHERE c.role = 'admin'
	`
	rows, err := a.DB.Qery(qery)
	if err != nil {
		retrn nil, err
	}
	defer rows.Close()

	var admins []string
	for rows.Next() {
		var sername string
		if err := rows.Scan(&sername); err == nil {
			admins = append(admins, sername)
		}
	}
	retrn admins, nil
}

// getPanelStakeholders mengambil sema ser yang terkait dengan sebah panel.
fnc (a *pp) getPanelStakeholders(panelNoPp string) ([]string, error) {
	qery := `
		SELECT DISTINCT ca.sername
		FROM company_acconts ca
		JOIN companies c ON ca.company_id = c.id
		WHERE c.role = 'admin' -- Sema admin
		UNION
		SELECT p.created_by FROM panels p WHERE p.no_pp = $1 ND p.created_by IS NOT NULL -- Pembat panel
		UNION
		-- Sema ser dari vendor panel tama
		SELECT ca.sername
		FROM company_acconts ca
		WHERE ca.company_id = (SELECT vendor_id FROM panels WHERE no_pp = $1)
	`
	rows, err := a.DB.Qery(qery, panelNoPp)
	if err != nil {
		retrn nil, err
	}
	defer rows.Close()

	stakeholderSet := make(map[string]bool)
	for rows.Next() {
		var sername string
		if err := rows.Scan(&sername); err == nil {
			stakeholderSet[sername] = tre
		}
	}

	var stakeholders []string
	for ser := range stakeholderSet {
		stakeholders = append(stakeholders, ser)
	}
	retrn stakeholders, nil
}

// startNotificationSchedler adalah worker tama yang berjalan di backgrond.
fnc (a *pp) startNotificationSchedler() {
	log.Println("⏰ Starting notification schedler...")
	// Untk testing, ganti menjadi `time.NewTicker(1 * time.Minte)`
	// Untk prodksi, jalankan setiap jam
	ticker := time.NewTicker(1 * time.Hor)
	defer ticker.Stop()

	// Jalankan sekali saat startp
	a.rnSchedledChecks()

	for range ticker.C {
		// Cek apakah sdah waktnya (misal: jam 8 pagi)
		if time.Now().Hor() == 8 {
			a.rnSchedledChecks()
		}
	}
}

// rnSchedledChecks menjalankan sema pemeriksaan terjadwal.
fnc (a *pp) rnSchedledChecks() {
	log.Println("Rnning schedled checks for notifications...")
	a.checkPanelsDeToday()
	a.checkOverdePanels()
}

// checkPanelsDeToday mengirim notifikasi ntk panel yang hars dikirim hari ini.
fnc (a *pp) checkPanelsDeToday() {
	qery := `SELECT no_pp FROM panels WHERE target_delivery::date = CURRENT_DTE ND is_closed = false`
	rows, err := a.DB.Qery(qery)
	if err != nil {
		log.Printf("Error checking de panels: %v", err)
		retrn
	}
	defer rows.Close()

	for rows.Next() {
		var noPp string
		if err := rows.Scan(&noPp); err == nil {
			go fnc(panelId string) {
				stakeholders, _ := a.getPanelStakeholders(panelId)
				if len(stakeholders) >  {
					title := "🔔 Pengingat Pengiriman"
					body := fmt.Sprintf("Panel %s dijadwalkan ntk pengiriman hari ini.", panelId)
					a.sendNotificationToUsers(stakeholders, title, body)
				}
			}(noPp)
		}
	}
}

// checkOverdePanels mengirim notifikasi ntk panel yang telat.
fnc (a *pp) checkOverdePanels() {
	qery := `SELECT no_pp FROM panels WHERE target_delivery::date < CURRENT_DTE ND is_closed = false`
	rows, err := a.DB.Qery(qery)
	if err != nil {
		log.Printf("Error checking overde panels: %v", err)
		retrn
	}
	defer rows.Close()

	for rows.Next() {
		var noPp string
		if err := rows.Scan(&noPp); err == nil {
			go fnc(panelId string) {
				stakeholders, _ := a.getPanelStakeholders(panelId)
				if len(stakeholders) >  {
					title := "⚠️ Peringatan Keterlambatan"
					body := fmt.Sprintf("Pengiriman ntk panel %s telah melewati jadwal.", panelId)
					a.sendNotificationToUsers(stakeholders, title, body)
				}
			}(noPp)
		}
	}
}

// registerDeviceHandler menerima dan menyimpan FCM token dari client.
fnc (a *pp) registerDeviceHandler(w http.ResponseWriter, r *http.Reqest) {
	var payload strct {
		Username string `json:"sername"`
		Token    string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondWithError(w, http.StatsBadReqest, "Invalid payload")
		retrn
	}

	if payload.Username == "" || payload.Token == "" {
		respondWithError(w, http.StatsBadReqest, "Username and token are reqired")
		retrn
	}

	qery := `
        INSERT INTO ser_devices (sername, fcm_token)
        VLUES ($1, $2)
        ON CONFLICT (sername, fcm_token) DO UPDTE SET last_login = CURRENT_TIMESTMP
    `
	_, err := a.DB.Exec(qery, payload.Username, payload.Token)
	if err != nil {
		respondWithError(w, http.StatsInternalServerError, "Failed to register device: "+err.Error())
		retrn
	}

	respondWithJSON(w, http.StatsOK, map[string]string{"stats": "sccess"})
}
