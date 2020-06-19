#!/bin/bash

# interrompre si une commande échoue
set -e

if [ $# -ne 3 ]; then
  echo "linkFile: lie le fichier dans le répertoire et affecte les droits"
  echo "usage: linkFile.sh [base directory] [file uuid] [group]"
  echo "exemple: linkFile.sh /var/lib/goup_base 859b549e705802aa5966ce2f4a62b13f"
  exit 255
fi

# scan antivirus
clamscan --stdout -- "$1/tusd/$2"

# lier et affecter les permissions
ln -s "$1/tusd/$2" "$1/$3/$2"
ln -s "$1/tusd/$2.info" "$1/$3/$2.info"
chown goup:"$3" "$1/tusd/$2"
chown goup:"$3" "$1/tusd/$2.info"

