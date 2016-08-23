#!/usr/bin/env bash

set -e
set -o pipefail
set -o nounset

SCRIPT_PATH="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
source $SCRIPT_PATH/../../json.sh

if [ ! -z "${VCAP_SERVICES-}" ]; then
  VCAP_SERVICES_NEW_RELIC_LICENSE_KEY=$(echo $VCAP_SERVICES | $JQ --raw-output .newrelic[0].credentials.licenseKey)
  VCAP_APPLICATION_GUID=$(echo $VCAP_APPLICATION | $JQ --raw-output .application_id)
  VCAP_APPLICATION_NAME=$(echo $VCAP_APPLICATION | $JQ --raw-output .application_name)

  if [ ! -z "${VCAP_SERVICES_NEW_RELIC_LICENSE_KEY-}" ];
  then
    if [ -z "${NEW_RELIC_LICENSE_KEY-}" ]; then
      export NEW_RELIC_LICENSE_KEY=$VCAP_SERVICES_NEW_RELIC_LICENSE_KEY
    fi
    if [ -z "${NEW_RELIC_APP_NAME-}" ]; then
      export NEW_RELIC_APP_NAME=$VCAP_APPLICATION_NAME"_"$VCAP_APPLICATION_GUID
    fi
  fi
fi
