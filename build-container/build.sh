#!/bin/sh

if [ "$#" -ne 1 ]; then
    echo "build.sh: construit l'application goup dans une image docker"
    echo "usage: build.sh branch"
    echo "exemple: ./build.sh master"
    exit 255
fi

if [ -d workspace ]; then
    echo "supprimer le répertoire workspace avant de commencer"
    exit 1
fi

# Checkout git
mkdir workspace
cd workspace
curl -LOs "https://github.com/signaux-faibles/goup/archive/$1.zip"

if [ $(openssl dgst -md5 $1.zip |awk '{print $2}') = '3be7b8b182ccd96e48989b4e57311193' ]; then
   echo "sources manquantes, branche probablement inexistante"
   exit
fi

# Unzip des sources et build
unzip "$1.zip"
cd "goup-$1"

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build

# Build docker
cd ../..
docker build -t goup --build-arg path="./workspace/goup-$1" . 
docker save goup | gzip > goup.tar.gz

# Cleanup
rm -rf workspace
