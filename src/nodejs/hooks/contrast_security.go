package hooks

import (
  "bytes"
  "encoding/json"
  "fmt"
  "github.com/cloudfoundry/libbuildpack"
  "io/ioutil"
  "os"
  "path"
  "path/filepath"
  "strings"
)

type ContrastSecurityHook struct {
  libbuildpack.DefaultHook
  Log               *libbuildpack.Logger
}

type ContrastSecurityCredentials struct {
  ApiKey      string
  OrgUuid     string
  ServiceKey  string
  ContrastUrl string          //formerly Teamserver URL
  Username    string
}

func init() {
  logger := libbuildpack.NewLogger(os.Stdout)

  libbuildpack.AddHook(ContrastSecurityHook {
    Log:               logger,
  })
}

func (h ContrastSecurityHook) AfterCompile(stager *libbuildpack.Stager) error {
  h.Log.Debug("Contrast Security after compile hook")

  success, contrastSecurityCredentials := h.GetCredentialsFromEnvironment()

  if(!success) {
    h.Log.Info("Contrast Security no credentials found. Will not write environment files.")
    return nil
  }

  profileDDir := path.Join(stager.BuildDir(), ".profile.d")

  if _, err := os.Stat(profileDDir); os.IsNotExist(err) {
    os.Mkdir(profileDDir, 0777)
  }

  var b bytes.Buffer

  b.WriteString(fmt.Sprintf("export CONTRAST__API__API_KEY=%s\n", contrastSecurityCredentials.ApiKey))
  b.WriteString(fmt.Sprintf("export CONTRAST__API__URL=%s\n", contrastSecurityCredentials.ContrastUrl + "/Contrast/"))
  b.WriteString(fmt.Sprintf("export CONTRAST__API__SERVICE_KEY=%s\n", contrastSecurityCredentials.ServiceKey))
  b.WriteString(fmt.Sprintf("export CONTRAST__API__USER_NAME=%s\n", contrastSecurityCredentials.Username))

  err := ioutil.WriteFile(filepath.Join(profileDDir, "contrast_security"), b.Bytes(), 0666)

  if(err != nil) {
    h.Log.Error(err.Error())
  } else {
    h.Log.Debug("Contrast Security successfully wrote %s", filepath.Join(profileDDir, "contrast_security"));
  }

  return nil
}

func (h ContrastSecurityHook) GetCredentialsFromEnvironment() (bool, ContrastSecurityCredentials) {
 type Service struct {
   Name        string                 `json:"name"`
   Credentials map[string]interface{} `json:"credentials"`
 }

 var vcapServices map[string][]Service

 vcapServicesEnvironment := os.Getenv("VCAP_SERVICES")

 if vcapServicesEnvironment == ""  {
   h.Log.Debug("Contrast Security could not find VCAP_SERVICES in the environment")
   return false, ContrastSecurityCredentials{}
 }

 err := json.Unmarshal([]byte(os.Getenv("VCAP_SERVICES")), &vcapServices)
 if err != nil {
   h.Log.Warning("Contrast Security could not parse VCAP_SERVICES")
   return false, ContrastSecurityCredentials{}
 }

 for key, services := range vcapServices {
   if strings.Contains(key, "contrast-security") {
     h.Log.Debug("Contrast Security found credentials in VCAP_SERVICES")
     for _, service := range services {
       apiKey      := getCredentialString(service.Credentials, "api_key")
       orgUuid     := getCredentialString(service.Credentials, "org_uuid")
       serviceKey  := getCredentialString(service.Credentials, "service_key")
       contrastUrl := getCredentialString(service.Credentials, "teamserver_url")
       username    := getCredentialString(service.Credentials, "username")

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

 return false, ContrastSecurityCredentials{}
}