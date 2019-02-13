VCAP_SERVICES_NEW_RELIC_LICENSE_KEY=$(echo "${VCAP_SERVICES-}" | jq -r .newrelic[0].credentials.licenseKey)
if [ -z "${VCAP_SERVICES_NEW_RELIC_LICENSE_KEY-}" ] || [ "$VCAP_SERVICES_NEW_RELIC_LICENSE_KEY" == "null" ];
then
  VCAP_SERVICES_NEW_RELIC_LICENSE_KEY=$(echo "${VCAP_SERVICES-}" | jq -r '[.[][] | select(.name | contains("newrelic"))][0] | .credentials | .["licenseKey"]');
fi

VCAP_APPLICATION_GUID=$(echo $VCAP_APPLICATION | jq -r .application_id)
VCAP_APPLICATION_NAME=$(echo $VCAP_APPLICATION | jq -r .application_name)

if [ ! -z "${VCAP_SERVICES_NEW_RELIC_LICENSE_KEY-}" ] && [ "$VCAP_SERVICES_NEW_RELIC_LICENSE_KEY" != "null" ];
then
  if [ -z "${NEW_RELIC_LICENSE_KEY-}" ]; then
    export NEW_RELIC_LICENSE_KEY=$VCAP_SERVICES_NEW_RELIC_LICENSE_KEY
  fi
  if [ -z "${NEW_RELIC_APP_NAME-}" ]; then
    export NEW_RELIC_APP_NAME="$VCAP_APPLICATION_NAME"_"$VCAP_APPLICATION_GUID"
  fi
fi
