package hooks

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/cloudfoundry/libbuildpack"
)

type ContrastSecurityHook struct {
	libbuildpack.DefaultHook
	Log *libbuildpack.Logger
}

type ContrastSecurityCredentials struct {
	ApiKey      string
	OrgUuid     string
	ServiceKey  string
	ContrastUrl string //formerly Teamserver URL
	Username    string
}

func init() {
	logger := libbuildpack.NewLogger(os.Stdout)

	libbuildpack.AddHook(ContrastSecurityHook{
		Log: logger,
	})
}

func (h ContrastSecurityHook) AfterCompile(stager *libbuildpack.Stager) error {
	h.Log.Debug("Contrast Security after compile hook")

	success, contrastSecurityCredentials := h.GetCredentialsFromEnvironment()

	if !success {
		h.Log.Info("Contrast Security no credentials found. Will not write environment files.")
		return nil
	}

	h.Log.Info("Contrast Security credentials found. Configuring environment for [%s].", contrastSecurityCredentials.ContrastUrl)

	var contrastSecurityScript = "export CONTRAST__API__API_KEY=" + contrastSecurityCredentials.ApiKey + "\n" +
		"export CONTRAST__API__URL=" + contrastSecurityCredentials.ContrastUrl + "/Contrast/\n" +
		"export CONTRAST__API__SERVICE_KEY=" + contrastSecurityCredentials.ServiceKey + "\n" +
		"export CONTRAST__API__USER_NAME=" + contrastSecurityCredentials.Username + "\n"

	stager.WriteProfileD("contrast_security", contrastSecurityScript)

	h.Log.Debug("Contrast Security successfully wrote to .profile.d")

	return nil
}

func getContrastCredentialString(credentials map[string]interface{}, key string) string {
	if value, exists := credentials[key]; exists {
		return value.(string)
	}
	return ""
}

func containsContrastService(key string, services interface{}, query string) bool {
	var serviceName string
	var serviceLabel string
	var serviceTags []interface{}

	if strings.Contains(key, query) {
		return true
	}
	val := services.([]interface{})
	for serviceIndex := range val {
		service := val[serviceIndex].(map[string]interface{})
		if v, ok := service["name"]; ok {
			serviceName = v.(string)
		}
		if v, ok := service["label"]; ok {
			serviceLabel = v.(string)
		}
		if strings.Contains(serviceName, query) || strings.Contains(serviceLabel, query) {
			return true
		}
		if v, ok := service["tags"]; ok {
			serviceTags = v.([]interface{})
		}
		for _, tagValue := range serviceTags {
			if strings.Contains(tagValue.(string), query) {
				return true
			}
		}
	}
	return false
}

func (h ContrastSecurityHook) GetCredentialsFromEnvironment() (bool, ContrastSecurityCredentials) {

	type rawVcapServicesJSONValue map[string]interface{}

	var vcapServices rawVcapServicesJSONValue

	vcapServicesEnvironment := os.Getenv("VCAP_SERVICES")

	if vcapServicesEnvironment == "" {
		h.Log.Debug("Contrast Security could not find VCAP_SERVICES in the environment")
		return false, ContrastSecurityCredentials{}
	}

	err := json.Unmarshal([]byte(vcapServicesEnvironment), &vcapServices)
	if err != nil {
		h.Log.Warning("Contrast Security could not parse VCAP_SERVICES")
		return false, ContrastSecurityCredentials{}
	}

	for key, services := range vcapServices {
		if containsContrastService(key, services, "contrast-security") {
			h.Log.Debug("Contrast Security found credentials in VCAP_SERVICES")
			val := services.([]interface{})
			for serviceIndex := range val {
				service := val[serviceIndex].(map[string]interface{})
				if credentials, exists := service["credentials"].(map[string]interface{}); exists {
					apiKey := getContrastCredentialString(credentials, "api_key")
					orgUuid := getContrastCredentialString(credentials, "org_uuid")
					serviceKey := getContrastCredentialString(credentials, "service_key")
					contrastUrl := getContrastCredentialString(credentials, "teamserver_url")
					username := getContrastCredentialString(credentials, "username")

					contrastSecurityCredentials := ContrastSecurityCredentials{
						ApiKey:      apiKey,
						OrgUuid:     orgUuid,
						ServiceKey:  serviceKey,
						ContrastUrl: contrastUrl,
						Username:    username,
					}
					return true, contrastSecurityCredentials
				}
			}
		}
	}

	return false, ContrastSecurityCredentials{}
}
