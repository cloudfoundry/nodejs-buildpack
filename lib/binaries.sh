needs_resolution() {
  local semver=$1
  if ! [[ "$semver" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    return 0
  else
    return 1
  fi
}

install_nodejs() {
  local requested_version="$1"
  local resolved_version=$requested_version
  local dir="$2"

  if needs_resolution "$requested_version"; then
    BP_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && cd .. && pwd )"
    versions_as_json=$(ruby -e "require 'yaml'; print YAML.load_file('$BP_DIR/manifest.yml')['dependencies'].select {|dep| dep['name'] == 'node' }.map {|dep| dep['version']}")
    default_version=$($BP_DIR/compile-extensions/bin/default_version_for $BP_DIR/manifest.yml node)
    resolved_version=$(ruby $BP_DIR/lib/version_resolver.rb "$requested_version" "$versions_as_json" "$default_version")
  fi

  if [[ "$resolved_version" = "undefined" ]]; then
    echo "Downloading and installing node $requested_version..."
  else
    echo "Downloading and installing node $resolved_version..."
  fi

  local heroku_url="https://s3pository.heroku.com/node/v$resolved_version/node-v$resolved_version-$os-$cpu.tar.gz"
  local exit_code=0
  local filtered_url=""

  filtered_url=$($BP_DIR/compile-extensions/bin/download_dependency $heroku_url /tmp) || exit_code=$?
  if [ $exit_code -ne 0 ]; then
    echo -e "`$BP_DIR/compile-extensions/bin/recommend_dependency $heroku_url`" 1>&2
    exit 22
  fi

  local downloaded_file=$(ls /tmp/node-v*.tar.gz)
  mv $downloaded_file /tmp/node.tar.gz

  echo "Downloaded [$filtered_url]"
  tar xzf /tmp/node.tar.gz -C /tmp
  rm -rf $dir/*
  mv /tmp/node-v$resolved_version-$os-$cpu/* $dir
  chmod +x $dir/bin/*
}

install_iojs() {
  local version="$1"
  local dir="$2"

  if needs_resolution "$version"; then
    echo "Resolving iojs version ${version:-(latest stable)} via semver.io..."
    version=$(curl --silent --get  --retry 5 --retry-max-time 15 --data-urlencode "range=${version}" https://semver.herokuapp.com/iojs/resolve)
  fi

  echo "Downloading and installing iojs $version..."
  local download_url="https://iojs.org/dist/v$version/iojs-v$version-$os-$cpu.tar.gz"
  curl "$download_url" --silent --fail --retry 5 --retry-max-time 15 -o /tmp/node.tar.gz || (echo "Unable to download iojs $version; does it exist?" && false)
  tar xzf /tmp/node.tar.gz -C /tmp
  mv /tmp/iojs-v$version-$os-$cpu/* $dir
  chmod +x $dir/bin/*
}

download_failed() {
  echo "We're unable to download the version of npm you've provided (${1})."
  echo "Please remove the npm version specification in package.json"
  exit 1
}

install_npm() {
  local version="$1"

  if [ "$version" == "" ]; then
    echo "Using default npm version: `npm --version`"
  else
    if needs_resolution "$version"; then
      echo "Resolving npm version ${version} via semver.io..."
      version=$(curl --silent --get --retry 5 --retry-max-time 15 --data-urlencode "range=${version}" https://semver.herokuapp.com/npm/resolve || echo failed)
      if [ "$version" = "failed" ]; then
        download_failed $1
      fi
    fi
    if [[ `npm --version` == "$version" ]]; then
      echo "npm `npm --version` already installed with node"
    else
      echo "Downloading and installing npm $version (replacing version `npm --version`)..."
      npm install --unsafe-perm --quiet -g npm@$version 2>&1 >/dev/null || download_failed $version
    fi
  fi
}
