package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/simple-datasource-backend/pkg/auth"
)

type Column struct {
	Text string `json:"text"`
}

type TablePanel struct {
	ID           int             `json:"id"`
	From         string          `json:"from"`
	To           string          `json:"to"`
	Title        string          `json:"title"`
	RawSql       string          `json:"rawSql"`
	Rows         [][]interface{} `json:"rows"`
	Columns      []Column        `json:"columns"`
	Variables    TemplateList    `json:"variables"`
	DatasourceID int             `json:"DatasourceID"`
}

func NewTablePanel(id int, title string, rawSql string, from string, to string, datasourceID int) *TablePanel {
	return &TablePanel{ID: id, Title: title, RawSql: rawSql, From: from, To: to, DatasourceID: datasourceID}
}

func (panel *TablePanel) usesVariable(variable TemplateVariable) bool {
	return strings.Contains(panel.RawSql, "${"+variable.Name)
}

func (panel *TablePanel) injectVariable(variable TemplateVariable, storeIDs string, contentVariables string) {

	var vars map[string][]string
	err := json.Unmarshal([]byte(contentVariables), &vars)
	if err != nil {
		log.DefaultLogger.Error("injectVariable: Decoding JSON: "+contentVariables, err.Error())
	}

	if (variable.Name) == "store" {
		csv := ""
		split := strings.Split(storeIDs, ",")
		for i, substr := range split {

			if i == len(split)-1 {
				csv = csv + "'" + substr + "'"
			} else {
				csv = csv + "'" + substr + "'" + ", "
			}
		}

		panel.RawSql = strings.Replace(panel.RawSql, "${"+variable.Name+"}", csv, -1)
		panel.RawSql = strings.Replace(panel.RawSql, "${"+variable.Name+":sqlstring}", csv, -1)

	} else {

		if variable.Type == "custom" {
			var joined string
			for i, str := range vars["Test"] {
				if i == len(vars[variable.Name])-1 {
					joined = joined + "'" + str + "'"
				} else {
					joined = joined + "'" + str + "'" + ", "
				}
			}

			panel.RawSql = strings.Replace(panel.RawSql, "${"+variable.Name+"}", joined, -1)
			panel.RawSql = strings.Replace(panel.RawSql, "${"+variable.Name+":sqlstring}", joined, -1)

		} else {
			panel.RawSql = strings.Replace(panel.RawSql, "${"+variable.Name+"}", variable.Definition, -1)
			panel.RawSql = strings.Replace(panel.RawSql, "${"+variable.Name+":sqlstring}", variable.Definition, -1)
		}

	}

	log.DefaultLogger.Info("panel.RawSql: " + panel.RawSql)
}

func (panel *TablePanel) PrepSql(variables TemplateList, storeIDs string, contentVariables string) {
	log.DefaultLogger.Info(contentVariables)
	for _, variable := range variables.List {
		if panel.usesVariable(variable) {
			panel.injectVariable(variable, storeIDs, contentVariables)
		}
	}
}

func (panel *TablePanel) GetData(authConfig auth.AuthConfig) error {
	body, err := NewQueryRequest(panel.RawSql, panel.From, panel.To, panel.DatasourceID).ToRequestBody()
	if err != nil {
		log.DefaultLogger.Error("GetData: NewQueryRequest: " + err.Error())
		return err
	}

	url := "http://" + authConfig.AuthString() + authConfig.URL + "/api/tsdb/query"
	response, err := http.Post(url, "application/json", body)
	if err != nil {
		log.DefaultLogger.Error("GetData: http.Post: " + err.Error())
		return err
	}

	qr, err := NewQueryResponse(response)
	if err != nil {
		log.DefaultLogger.Error("GetData: NewQueryResponse: " + err.Error())
		return err
	}

	panel.SetRows(qr.Rows())
	panel.SetColumns(qr.Columns())

	return nil
}

func (panel *TablePanel) SetRows(rows [][]interface{}) {
	panel.Rows = rows
}

func (panel *TablePanel) SetColumns(columns []Column) {
	panel.Columns = columns
}
