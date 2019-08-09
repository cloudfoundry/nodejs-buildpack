package hooks

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/cloudfoundry/libbuildpack"
)

// ContrastSecurityHook implements libbuildpack.Hook to get Contrast credentials from bound Contrast service.
type ContrastSecurityHook struct {
	libbuildpack.DefaultHook
	Log *libbuildpack.Logger
}

// ContrastSecurityCredentials represents the credentials used to authenticate with the ContrastUI (formerly TeamServer)
type ContrastSecurityCredentials struct {
	APIKey      string
	OrgUUID     string
	ServiceKey  string
	ContrastURL string //formerly Teamserver URL
	Username    string
}

func init() {
	logger := libbuildpack.NewLogger(os.Stdout)

	libbuildpack.AddHook(ContrastSecurityHook{
		Log: logger,
	})
}

// AfterCompile gets the Contrast Security credentials from the VCAP_SERVICES environment variable.
func (h ContrastSecurityHook) AfterCompile(stager *libbuildpack.Stager) error {
	h.Log.Debug("Contrast Security after compile hook")

	success, contrastSecurityCredentials := h.GetCredentialsFromEnvironment()

	if !success {
		h.Log.Info("Contrast Security no credentials found. Will not write environment files.")
		return nil
	}

	var contrastSecurityScript = "export CONTRAST__API__API_KEY=" + contrastSecurityCredentials.APIKey + "\n" +
		"export CONTRAST__API__URL=" + contrastSecurityCredentials.ContrastURL + "/Contrast/\n" +
		"export CONTRAST__API__SERVICE_KEY=" + contrastSecurityCredentials.ServiceKey + "\n" +
		"export CONTRAST__API__USER_NAME=" + contrastSecurityCredentials.Username + "\n"

	stager.WriteProfileD("contrast_security", contrastSecurityScript)

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

// GetCredentialsFromEnvironment extracts Contrast Security credentials from VCAP_SERVICES environment variable, if they exist.
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
					orgUUID := getContrastCredentialString(credentials, "org_uuid")
					serviceKey := getContrastCredentialString(credentials, "service_key")
					contrastURL := getContrastCredentialString(credentials, "teamserver_url")
					username := getContrastCredentialString(credentials, "username")

					contrastSecurityCredentials := ContrastSecurityCredentials{
						APIKey:      apiKey,
						OrgUUID:     orgUUID,
						ServiceKey:  serviceKey,
						ContrastURL: contrastURL,
						Username:    username,
					}
					return true, contrastSecurityCredentials
				}
			}
		}
	}
	return false, ContrastSecurityCredentials{}
}
