package freshbooks

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/tambet/oauthplain"
)

type (
	Api struct {
		apiUrl     string
		apiToken   string
		oAuthToken *oauthplain.Token
		perPage    int
		users      []User
		tasks      []Task
		clients    []Client
		projects   []Project
	}
	Request struct {
		XMLName xml.Name `xml:"request"`
		Method  string   `xml:"method,attr"`
		PerPage int      `xml:"per_page"`
		Page    int      `xml:"page"`

		// optional filters used by various requests
		Email      string     `xml:"email,omitempty"`
		Username   string     `xml:"username,omitempty"`
		DateFrom   *Date      `xml:"date_from,omitempty"`
		DateTo     *Date      `xml:"date_to,omitempty"`
		UpdateFrom *Date      `xml:"update_from,omitempty"`
		UpdateTo   *Date      `xml:"update_to,omitempty"`
		TaskId     string     `xml:"task_id,omitempty"`
		ProjectId  string     `xml:"project_id,omitempty"`
		ClientId   string     `xml:"client_id,omitempty"`
		InvoiceId  string     `xml:"invoice_id,omitempty"`
		TimeEntry  *TimeEntry `xml:"time_entry,omitempty"`
	}
	Response struct {
		Error       string          `xml:"error"`
		Clients     ClientList      `xml:"clients"`
		Projects    ProjectList     `xml:"projects"`
		Tasks       TaskList        `xml:"tasks"`
		Users       UserList        `xml:"staff_members"`
		TimeEntries TimeEntriesList `xml:"time_entries"`
		Contractors ContractorList  `xml:"contractors"`
		Invoices    InvoiceList     `xml:"invoices"`
		// Payments    PaymentList     `xml:"payments"`
	}
	TimeEntryResponse struct {
		Status      string `xml:"status,attr"`
		Error       string `xml:"error"`
		Code        string `xml:"code"`
		Field       string `xml:"field"`
		TimeEntryId int    `xml:"time_entry_id"`
	}
	Pagination struct {
		Page    int `xml:"page,attr"`
		Total   int `xml:"total,attr"`
		PerPage int `xml:"per_page,attr"`
	}
	ClientList struct {
		Pagination
		Clients []Client `xml:"client"`
	}
	ProjectList struct {
		Pagination
		Projects []Project `xml:"project"`
	}
	TaskList struct {
		Pagination
		Tasks []Task `xml:"task"`
	}
	UserList struct {
		Pagination
		Users []User `xml:"member"`
	}
	TimeEntriesList struct {
		Pagination
		TimeEntries []TimeEntry `xml:"time_entry"`
	}
	ContractorList struct {
		Pagination
		Contractors []Contractor `xml:"contractor"`
	}
	InvoiceList struct {
		Pagination
		Invoices []Invoice `xml:"invoice"`
	}
	// PaymentList struct {
	// 	Pagination
	// 	Payments []Payment `xml:"payments"`
	// }

	Client struct {
		ClientId string `xml:"client_id"`
		Name     string `xml:"organization"`
	}
	Project struct {
		ProjectId string `xml:"project_id"`
		ClientId  string `xml:"client_id"`
		Name      string `xml:"name"`
		TaskIds   []int  `xml:"tasks>task>task_id"`
		UserIds   []int  `xml:"staff>staff>staff_id"`
	}
	Task struct {
		TaskId string `xml:"task_id"`
		Name   string `xml:"name"`
	}
	User struct {
		UserId    string `xml:"staff_id"`
		Email     string `xml:"email"`
		FirstName string `xml:"first_name"`
		LastName  string `xml:"last_name"`
	}
	TimeEntry struct {
		TimeEntryId int     `xml:"time_entry_id"`
		ProjectId   int     `xml:"project_id"` // Required
		TaskId      int     `xml:"task_id"`    // Required
		StaffId     string  `xml:"staff_id"`   // Required
		Date        string  `xml:"date"`       // Required
		Notes       string  `xml:"notes"`
		Hours       float64 `xml:"hours"`
	}
	Contractor struct {
		// XMLName      xml.Name `xml:"contractor"`
		ContractorId string    `xml:"contractor_id"`
		Name         string    `xml:"name"`
		Email        string    `xml:"email"`
		Rate         float64   `xml:rate`
		TaskId       string    `xml:task_id`
		Projects     []Project `xml:projects>project`
	}
	Invoice struct {
		InvoiceId         int        `xml:"invoice_id"`
		ClientId          int        `xml:"client_id"`
		Number            string     `xml:"number"`
		Amount            string     `xml:"amount"`
		CurrencyCode      string     `xml:"currency_code"`
		AmountOutstanding string     `xml:"amount_outstanding"`
		Status            string     `xml:"paid"`
		Date              fbTime     `xml:"date"`
		Updated           fbTime     `xml:"updated"`
		Orgnization       string     `xml:"organization"`
		LineItems         []LineItem `xml:"lines"`
	}
	LineItem struct {
		LineId   int    `xml:"line_id"`
		Amount   string `xml:"amount"`
		Name     string `xml:"name"`
		UnitCost string `xml:"unit_cost"`
		Quantity int    `xml:"quantity"`
		Type     string `xml:"type"`
	}
	// Payment struct {
	// 	PaymentId    int    `xml:"payment_id"`
	// 	InvoiceId    int    `xml:"invoice_id"`
	// 	Date         fbTime `xml:"date"`
	// 	Updated      fbTime `xml:"updated"`
	// 	ClientId     int    `xml:"client_id"`
	// 	CurrencyCode int    `xml:"currency_code"`
	// 	Amount       string `xml:"amount"`
	// }
	fbTime time.Time
)

func NewApi(account string, token interface{}) *Api {
	url := fmt.Sprintf("https://%s.freshbooks.com/api/2.1/xml-in", account)
	fb := Api{apiUrl: url, perPage: 25}

	switch token.(type) {
	case string:
		fb.apiToken = token.(string)
	case *oauthplain.Token:
		fb.oAuthToken = token.(*oauthplain.Token)
	}
	return &fb
}

func (r *Request) setDefaults(api *Api, method string) {
	if r.PerPage < 1 {
		r.PerPage = api.perPage
	}
	if r.Page < 1 {
		r.Page = 1
	}
	r.Method = method
}

func (api *Api) ListClients(request Request) (*[]Client, error) {
	request.setDefaults(api, "client.list")

	response, err := api.request(request)
	return &response.Clients.Clients, err
}

func (api *Api) ListTimeEntries(request Request) (*[]TimeEntry, *Pagination, error) {
	request.setDefaults(api, "time_entry.list")

	response, err := api.request(request)
	return &response.TimeEntries.TimeEntries, &response.TimeEntries.Pagination, err
}

func (api *Api) ListContractors(request Request) (*[]Contractor, *Pagination, error) {
	request.setDefaults(api, "contractor.list")

	response, err := api.request(request)
	return &response.Contractors.Contractors, &response.Contractors.Pagination, err
}

func (api *Api) ListInvoices(request Request) (*[]Invoice, *Pagination, error) {
	request.setDefaults(api, "invoice.list")

	response, err := api.request(request)
	return &response.Invoices.Invoices, &response.Invoices.Pagination, err
}

// func (api *Api) ListPayments(request Request) (*[]Payment, *Pagination, error) {
// 	request.setDefaults(api, "payment.list")
//
// 	response, err := api.request(request)
// 	return &response.Payments.Payments, &response.Payments.Pagination, err
// }

func (api *Api) request(request Request) (Response, error) {
	response := Response{}
	// fmt.Printf("%#v", request)

	result, err := api.makeRawRequest(request)
	if err != nil {
		return response, err
	}

	if err := xml.Unmarshal(*result, &response); err != nil {
		return response, err
	}
	if len(response.Error) > 0 {
		return response, errors.New(response.Error)
	}

	return response, nil
}

func (this *Api) makeRawRequest(request interface{}) (*[]byte, error) {
	xmlRequest, err := xml.MarshalIndent(request, "", "  ")
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", this.apiUrl, bytes.NewBuffer(xmlRequest))
	if err != nil {
		return nil, err
	}

	if this.apiToken != "" {
		req.SetBasicAuth(this.apiToken, "X")
	} else if this.oAuthToken != nil {
		header := this.oAuthToken.AuthHeader()
		req.Header.Set("Authorization", header)
	}

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, errors.New(response.Status)
	}

	result, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (t *fbTime) UnmarshalText(b []byte) error {
	result, err := time.Parse("2006-01-02 15:04:05", string(b))
	if err != nil {
		var t2 time.Time
		err = t2.UnmarshalText(b)
		if err != nil {
			return err
		}
		*t = fbTime(t2)
		return nil
	}

	// Save as data
	*t = fbTime(result)
	return nil
}
