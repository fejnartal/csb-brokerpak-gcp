package app

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"regexp"

	"github.com/gorilla/mux"
	_ "github.com/jackc/pgx/v4/stdlib"
)

const (
	tableName   = "test"
	keyColumn   = "keyname"
	valueColumn = "valuedata"
)

func App(uri string) *mux.Router {
	db, err := connect(uri)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Connection succeeded")

	r := mux.NewRouter()
	r.HandleFunc("/", aliveness).Methods(http.MethodHead, http.MethodGet)
	r.HandleFunc("/{schema}", handleCreateSchema(db)).Methods(http.MethodPut)
	r.HandleFunc("/{schema}", handleDropSchema(db)).Methods(http.MethodDelete)
	r.HandleFunc("/{schema}/{key}", handleSet(db)).Methods(http.MethodPut)
	r.HandleFunc("/{schema}/{key}", handleGet(db)).Methods(http.MethodGet)
	r.HandleFunc("/schemas/{schema}/{table}", handleAlterTable(db)).Methods(http.MethodPut)
	r.HandleFunc("/", handleDeleteTestTable(db)).Methods(http.MethodDelete)

	return r
}

func aliveness(w http.ResponseWriter, _ *http.Request) {
	log.Printf("Handled aliveness test.")
	w.WriteHeader(http.StatusNoContent)
}

func connect(uri string) (*sql.DB, error) {
	db, err := sql.Open("pgx", uri)

	if err != nil {
		return nil, fmt.Errorf("%w: failed to connect to database", err)
	}
	db.SetMaxIdleConns(0)

	_, err = db.Exec(fmt.Sprintf(`CREATE TABLE IF NOT EXISTS public.%s (%s VARCHAR(255) NOT NULL, %s VARCHAR(255) NOT NULL)`, tableName, keyColumn, valueColumn))
	if err != nil {
		return nil, fmt.Errorf("%w: error creating table", err)
	}

	_, err = db.Exec(fmt.Sprintf(`GRANT ALL ON TABLE public.%s TO PUBLIC`, tableName))
	if err != nil {
		return nil, fmt.Errorf("%w: error granting table permissions", err)
	}

	return db, nil
}

func fail(w http.ResponseWriter, code int, format string, a ...any) {
	msg := fmt.Sprintf(format, a...)
	log.Println(msg)
	http.Error(w, msg, code)
}

func schemaName(r *http.Request) (string, error) {
	schema, ok := mux.Vars(r)["schema"]

	switch {
	case !ok:
		return "", fmt.Errorf("schema missing")
	case len(schema) > 50:
		return "", fmt.Errorf("schema name too long")
	case len(schema) == 0:
		return "", fmt.Errorf("schema name cannot be zero length")
	case !regexp.MustCompile(`^[a-zA-Z0-9]+$`).MatchString(schema):
		return "", fmt.Errorf("schema name contains invalid characters")
	default:
		return schema, nil
	}
}
