package server

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/shellfly/rest/pkg/database"
)

const (
	setupSQL = `
DROP TABLE IF EXISTS "customers";
CREATE TABLE IF NOT EXISTS "customers"
(
    [CustomerId] INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
    [FirstName] NVARCHAR(40)  NOT NULL,
    [LastName] NVARCHAR(20)  NOT NULL,
    [Email] NVARCHAR(60)  NOT NULL,
	[Active] BOOL NOT NULL
);
DROP TABLE IF EXISTS "invoices";
CREATE TABLE IF NOT EXISTS "invoices"
(
    [InvoiceId] INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
    [CustomerId] INTEGER  NOT NULL,
    [InvoiceDate] DATETIME  NOT NULL,
    [BillingAddress] NVARCHAR(70),
    [Total] NUMERIC(10,2)  NOT NULL,
	[Data] JSON NOT NULL,
    FOREIGN KEY ([CustomerId]) REFERENCES "customers" ([CustomerId])
                ON DELETE NO ACTION ON UPDATE NO ACTION
);
CREATE INDEX [IFK_InvoiceCustomerId] ON "invoices" ([CustomerId]);
`
)

var testServer *Server

func TestMain(m *testing.M) {
	testServer = NewServer("sqlite://ci.db")
	if err := setupData(); err != nil {
		log.Fatal("setupData error: ", err)
	}
	os.Exit(m.Run())
}

func setupData() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if _, err := database.ExecQuery(ctx, testServer.db, setupSQL); err != nil {
		return err
	}
	body := strings.NewReader(`{
		"CustomerId": 1,
		"FirstName": "first name",
		"LastName": "last_name",
		"Email": "a@b.com", 
		"Active":true
	}`)
	if _, err := request(http.MethodPost, "/customers", body); err != nil {
		return err
	}

	body = strings.NewReader(`[
			{
				"InvoiceID": 1,
				"CustomerId":1,
				"InvoiceDate": "2023-01-02 03:04:05",
				"BillingAddress": "I'm an address",
				"Total":3.1415926,
				"Data": "{\"Country\": \"I'm an country\", \"PostalCode\":1234}"
			},
			{
				"InvoiceID": 2,
				"CustomerId":1,
				"InvoiceDate": "2023-01-02 03:04:05",
				"BillingAddress": "I'm an address",
				"Total":1.141421,
				"Data": "{\"Country\": \"I'm an country\", \"PostalCode\":1234}"
			}
		]`)
	_, err := request(http.MethodPost, "/invoices", body)
	return err
}

func request(method, target string, body io.Reader) (*Response, error) {
	req := httptest.NewRequest(method, target, body)
	w := httptest.NewRecorder()
	testServer.ServeHTTP(w, req)
	res := w.Result()
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var response Response
	err = json.Unmarshal(data, &response)
	return &response, err
}