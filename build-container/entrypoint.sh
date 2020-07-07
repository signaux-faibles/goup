#!/bin/sh

set -e

if [[ -z "$GOUP_UID" ]]; then
  echo "$GOUP_UID is mandatory, please provide goup user's uid"
  exit 1
fi

CONTAINER_ALREADY_STARTED="/app/container_already_started"

if [ ! -e $CONTAINER_ALREADY_STARTED ]; then
    echo "-- first startup, provisionning users and groups"
    
    adduser -u $GOUP_UID --disabled-password goup

    if [ -e /app/groups ]; then
      while read gid group
      do 
        addgroup -g "${gid}" "${group}"
        addgroup goup "${group}" 
        echo "group ${group} with gid ${gid} created and goup user added"
        
      done < /app/groups
    else
      echo "no groups created, that's weird but ok, are you sure ?"
      echo "in order to create groups, provide /app/groups in this form"
      echo "> $ cat /app/groups"
      echo "> 500 group1"
      echo "> 501 group2"
    fi

    echo "successful init, just running"
    touch $CONTAINER_ALREADY_STARTED
else
    echo "-- Not first container startup, just running"
fi

sudo -i -u goup GIN_MODE=release /app/goup