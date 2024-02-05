package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"html/template"
	"io"
	"net/http"
	"reflect"
	"slices"
	"strings"
)

func runHTTPServer() {
	router := mux.NewRouter()
	router.HandleFunc("/", indexHandler).Methods("GET")
	router.HandleFunc("/save_config", saveConfigHandler).Methods("POST")
	router.HandleFunc("/otp", otpGetHandler).Methods("GET")
	router.HandleFunc("/otp", otpPostHandler).Methods("POST")
	http.ListenAndServe("0.0.0.0:5000", router)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	exportedConf, err := currentConf.Export()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl := template.New("index.html")
	tmpl, err = tmpl.ParseFiles("template/index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, exportedConf)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func saveConfigHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// get conf meta
	confExport, err := currentConf.Export()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// param check
	for _, confSection := range confExport.ConfigSections {
		for _, config := range confSection.Configs {
			formValues := r.Form[confSection.Section+"__"+config.ConfigKey]
			isRequired := (confSection.Section == GlobalConfigSectionName && config.Required) ||
				(slices.Contains(r.Form[GlobalConfigSectionName+"__"+ConfKeyEnabledWebsites], confSection.Section) && config.Required)
			if isRequired && len(formValues) == 0 {
				json.NewEncoder(w).Encode(map[string]interface{}{"success": false,
					"message": fmt.Sprintf(`Config "%s" is required`, config.ConfigName)})
				return
			}
			if config.Type == FieldTypeString && len(formValues) > 0 {
				value := formValues[0]
				if isRequired && value == "" {
					json.NewEncoder(w).Encode(map[string]interface{}{"success": false,
						"message": fmt.Sprintf(`Config "%s" under section "%s" is required`, config.ConfigName, confSection.DisplayName)})
					return
				}
			} else if config.Type == FieldTypeBool && len(formValues) > 0 {
				value := formValues[0]
				if value != TrueLiteral && value != FalseLiteral {
					json.NewEncoder(w).Encode(map[string]interface{}{"success": false,
						"message": fmt.Sprintf(`Invalid value for bool option "%s"`, config.ConfigName)})
					return
				}
			}
		}
	}

	// set values
	for _, confSection := range confExport.ConfigSections {
		for _, config := range confSection.Configs {
			formValues := r.Form[confSection.Section+"__"+config.ConfigKey]
			if len(formValues) == 0 {
				continue
			}
			var structToChange reflect.Value
			if confSection.Section == GlobalConfigSectionName {
				// is global option
				structToChange = reflect.ValueOf(currentConf)
			} else {
				// is agent option
				if currentConf.AgentConfs[confSection.Section] == nil {
					currentConf.AgentConfs[confSection.Section] = &AgentConf{}
				}
				structToChange = reflect.ValueOf(currentConf.AgentConfs[confSection.Section])
			}

			var err error
			if config.Type == FieldTypeString {
				value := formValues[0]
				err = SetValueByJSONTag(structToChange, config.ConfigKey, value)
			} else if config.Type == FieldTypeStringList {
				value := strings.Split(formValues[0], ",")
				if len(value) == 1 && value[0] == "" {
					value = []string{}
				}
				err = SetValueByJSONTag(structToChange, config.ConfigKey, value)
			} else if config.Type == FieldTypeCheckbox {
				value := formValues
				err = SetValueByJSONTag(structToChange, config.ConfigKey, value)
			} else if config.Type == FieldTypeRadio {
				value := formValues[0]
				err = SetValueByJSONTag(structToChange, config.ConfigKey, value)
			} else if config.Type == FieldTypeBool {
				value := formValues[0]
				if value == "True" {
					err = SetValueByJSONTag(structToChange, config.ConfigKey, true)
				} else {
					err = SetValueByJSONTag(structToChange, config.ConfigKey, false)
				}
			} else {
				json.NewEncoder(w).Encode(map[string]interface{}{"success": false,
					"message": fmt.Sprintf("unknown field type %s", config.Type)})
				return
			}
			if err != nil {
				json.NewEncoder(w).Encode(map[string]interface{}{"success": false,
					"message": fmt.Sprintf("set value for %s__%s failed: %s", confSection.Section, config.ConfigKey, err.Error())})
				return
			}
		}
	}

	err = currentConf.WriteDisk()
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false,
			"message": fmt.Sprintf("write config to disk failed: %s", err.Error())})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

func otpGetHandler(w http.ResponseWriter, r *http.Request) {
	select {
	case reqAgent := <-otpRequestChan:
		json.NewEncoder(w).Encode(map[string]interface{}{"prompt": fmt.Sprintf("%s requires OTP. Please check your mailbox and input OTP here:", reqAgent)})
	default:
		json.NewEncoder(w).Encode(map[string]interface{}{"prompt": nil})
	}
}

func otpPostHandler(w http.ResponseWriter, r *http.Request) {
	type Payload struct {
		Input string `json:"input"`
	}

	body, _ := io.ReadAll(r.Body)
	defer r.Body.Close()

	var p Payload
	err := json.Unmarshal(body, &p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	select {
	case otpChan <- p.Input:
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
	default:
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "No website requires OTP"})
	}
}
