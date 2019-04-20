# goup
Serveur minimaliste pour téléverser les fichiers bruts du projet Signaux Faibles

## Présentation
### goup
Goup est un microservice permettant l'upload de fichiers aux utilisateurs authentifiés sur présentation d'un jeton disposant d'une signature valide et des informations nécessaires au traitement du fichier.
Ce microservice respecte le protocol TUS, ce qui permet les fonctionnalités suivantes:
- pas de limite de taille du fichier
- l'envoi peut être interrompu et reprendre là où il s'était arrété
- l'envoi peut comporter des métadonnées
- on peut configurer pendant l'envoi la visibilité du fichier à l'ensemble des partenaires de la convention Signaux-Faibles
- l'envoi peut être segmenté en morceau de taille arbitraire pour respecter d'éventuelles limitations administratives sur la taille des requêtes HTTP
- il est possible d'envoyer plusieurs fichiers en parallèle


Goup est basé sur https://github.com/tus/tusd mais certaines fonctionnalités ont été arrangées:
- il n'est plus possible de télécharger un fichier depuis le serveur
- une fois l'envoi terminé, le fichier est déplacé dans un espace de stockage spécifique à l'utilisateur
- les droits utilisateurs sont alors définis sur le fichier 

### présentation du protocol
1. Le client teste si le fichier est déjà présent avec une requête `HEAD`
2. Si le fichier n'existe pas, une requête `POST` comportant les métadonnées permet la création du fichier à vide
3. Des requêtes `PATCH` fournissent les données que le service inscrit dans le répertoire de stockage

### stockage
L'implémentation actuelle se base sur un stockage fichier POSIX, toutefois l'implémentation du serveur tusd permet d'exploiter de nombreux types de stockage.

## Utilisation
### Client
Il est conseillé d'utiliser un client tus, il en existe un facile de mise en oeuvre en python:

Si vous souhaitez intégrer l'upload de fichier dans un projet, il existe des librairies qui implémentent le protocol de nombreux langages. Pour n'en citer que quelques-uns:
- javascript: https://github.com/tus/tus-js-client
- python: https://github.com/tus/tus-py-client
- java: https://github.com/tus/tus-java-client
- go: https://github.com/eventials/go-tus
- php: https://github.com/ankitpokhrel/tus-php

### Exemple Javascript
```javascript
input.addEventListener("change", function(e) {
    // Get the selected file from the input element
    var file = e.target.files[0]

    // Create a new tus upload
    var upload = new tus.Upload(file, {
        endpoint: "http://localhost:1080/files/",
        retryDelays: [0, 3000, 5000, 10000, 20000],
        headers: {
            Authorization: 'Bearer ' + currentToken  // currentToken contient le token d'authentification
        },
        metadata: {
            filename: file.name,
            filetype: file.type,
            private: 'true'
        },
        onError: function(error) {
            console.log("Failed because: " + error)
        },
        onProgress: function(bytesUploaded, bytesTotal) {
            var percentage = (bytesUploaded / bytesTotal * 100).toFixed(2)
            console.log(bytesUploaded, bytesTotal, percentage + "%")
        },
        onSuccess: function() {
            console.log("Download %s from %s", upload.file.name, upload.url)
        }
    })

    // Start the upload
    upload.start()
})
```
### Authentification
La version actuelle propose un service d'authentification minimal pour la démonstration.
Pour effectuer cette authentification, il est nécessaire d'attacher le token dans les entêtes des requêtes tus, ce qui est pris en charge par les clients tus, dans l'exemple précédent:
```javascript
...
        headers: {
            Authorization: 'Bearer ' + currentToken  // currentToken contient le token d'authentification
        },
...
```

### Structure du jeton
Le jeton doit être signé avec le même secret que celui déclaré dans la configuration (`jwtSecret`).  
Le chargement du jeton est un objet JSON devant comporter un clé `value.path` avec une chaine de caractère identifiant l'utilisateur POSIX du système cible. En conséquence de quoi:

- La clé `value.path` doit correspondre à un nom d'utilisateur POSIX du serveur sans quoi les téléversements seront supprimés dès leur complétion
- En absence de cette clé, l'utilisateur sera considéré comme non habilité à envoyer des fichiers et se verra opposer des codes de retour `HTTP 403` lors des tentatives de création (`POST`)
- Les uploads en cours sont situés dans `$basePath/tusd`
- Le service déplacera les nouveaux fichiers dans le répertoire `$basePath/users/$value.path` une fois leurs envois terminés
- Les droits POSIX seront fixés sur les nouveaux fichiers de façon à limiter l'accès de ces fichiers utilisateurs POSIX du système

### Metadonnées
Il est possible de fixer des métadonnées de façon arbitraire en suivant ces prérogatives:
- Les métadonnées sont des chaines de caractères
- La métadonnée private est interprétée par goup, si elle vaut `true` alors le serveur traitera le fichier de façon à empêcher son accès aux autres utilisateurs
- La métadonnée goup-path est réservée par le serveur pour traiter le chemin de stockage, **toute valeur fixée pendant l'envoi sera écrasée**

### Envoi en mode privé
Ce mode permet d'envoyer un fichier sur la plateforme sans partager son contenu avec tous les utilisateurs de la plateforme.

Pour l'exploiter, il faut fixer la métadonnée `private` à la valeur `true`

Exemple javascript:
```javascript
...
    metadata: {
        filename: file.name,
        filetype: file.type,
        private: 'true'
    },
...
```

Toute autre valeur sera ignorée.

## Installation
Prérequis: go > 1.8

`go get github.com/signaux-faibles/goup`

### Configuration
La configuration s'effectue avec un fichier au format toml
```
bind = "127.0.0.1:5000" 
jwtSecret = "don't keep this secret"
basePath = "/some/basePath"
```
#### bind
Adresse TCP sur laquelle va écouter le service goup
#### jwtSecret
Clé de signature des token JWT, doit être partagée avec le service qui génère les tokens
#### basePath
Chemin de base où sera réalisé le stockage

### Stockage
#### Fichiers
Les fichiers sont stockés avec l'identifiant créé par le serveur tus dans 2 fichiers:
- un fichier de données avec une extension .bin
- un fichier de métadonnées avec une extension .info

Exemple de fichier
```javascript
{
    "ID": "f72a07d9d12aa070f5a18d7376865795",
    "Size": 646221764,
    "SizeIsDeferred": false,
    "Offset": 0,
    "MetaData": {
        "batch": "1903",
        "filename": "Vacances au Puy du Fou.mkv",
        "filetype": "video/x-matroska",
        "goup-path": "admin",
        "private": "true",
        "type-signauxfaibles": "loisir"
    },
    "IsPartial": false,
    "IsFinal": false,
    "PartialUploads": null
}

```

On retrouve dans ce fichier des informations relatives à l'état du fichier dans son traitement 
#### Répertoires
Les répertoires de stockage doivent être créés au préalable ainsi que les sous-répertoires correspondant aux utilisateurs et leurs espaces privés. Les droits utilisateurs des répertoires doivent être fixés au préalable, le serveur se chargera de fixer les droits des fichiers importés.

Ci-dessous un exemple:

```
chemin                           user:group   mode

.
+-- tusd                         root :root   700
+-- user1                        user1:users  750
|   +-- file1.bin                user1:users  640
|   +-- file1.info               user1:users  640
|   +-- private                  user1:users  700
|   |   +-- file2.bin            user1:users  600
|   |   +-- file2.info           user1:users  600
+-- $value.path2                 user2:users  750
|   +-- file3.bin                user2:users  640
|   +-- file3.info               user2:users  640
|   +-- private                  user2:users  700
|   |   +-- file4.bin            user2:users  600
|   |   +-- file4.info           user2:users  600
```

