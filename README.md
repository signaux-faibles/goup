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


Goup est basé sur https://github.com/tus/tusd et incorpore certaines fonctionnalités supplémentaires:
- il n'est plus possible de télécharger un fichier depuis le serveur
- une fois l'envoi terminé, le fichier est déplacé dans un espace de stockage spécifique avec des droits utilisateurs
- il est possible d'orienter un versement vers l'espace partagé entre les utilisateurs ou un espace privatif


### présentation du protocol
1. Le client teste si le fichier est déjà présent avec une requête `HEAD`
2. Si le fichier n'existe pas, une requête `POST` comportant les métadonnées permet la création du fichier à vide
3. Des requêtes `PATCH` fournissent les données que le service inscrit dans le répertoire de stockage

### stockage
L'implémentation actuelle se base sur un stockage fichier POSIX, toutefois l'implémentation du serveur tusd permet d'exploiter de nombreux types de stockage.

## Utilisation
### Client
Il est conseillé d'utiliser un client tus, il en existe un facile de mise en oeuvre en python:

Si vous souhaitez intégrer l'upload de fichier dans un projet, il existe des librairies qui implémentent le protocol de nombreux langages. Pour n'en citer que quelques-uns:
- javascript: https://github.com/tus/tus-js-client
- python: https://github.com/tus/tus-py-client
- java: https://github.com/tus/tus-java-client
- go: https://github.com/eventials/go-tus
- php: https://github.com/ankitpokhrel/tus-php
- .NET: https://github.com/gerdus/tus-dotnet-client

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
            typeSignauxFaibles: 'extreme_cycling_downhill_on_volcano',
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
Pour authentifier un versement avec goup, il est nécessaire d'attacher un jeton JWT dans les entêtes des requêtes tus, ce qui est pris en charge par les clients standards. Dans l'exemple précédent réalisé avec le client javascript:
```javascript
...
        headers: {
            Authorization: 'Bearer ' + currentToken  // currentToken contient le token d'authentification
        },
...
```

Pour obtenir ce token, il conviendra d'adresser une requête sur le service `/login` de la plateforme signaux-faibles avec des identifiants d'utilisateurs reconnus. L'habilitation à verser des fichiers dans goup sera porté dans le chargement du jeton.
Dans la version de démonstration, un service minimal de login est en place, toutefois goup ne proposera pas dans sa version production de service d'authentification, il sera nécessaire pour l'utiliser d'obtenir un jeton JWT auprès d'un autre service. 

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
- La métadonnée goup-path est réservée par le serveur pour traiter le chemin de stockage, elle est fixée à partir de la valeur transmise dans le jeton d'authentification. **Toute valeur fixée dans les métadonnées de l'envoi sera écrasée.**

### Envoi en mode privé
Ce mode permet d'envoyer un fichier sur la plateforme sans partager son contenu avec tous les utilisateurs de la plateforme.
Pour l'utiliser, il faut fixer la métadonnée `'private'` à la valeur `'true'`; toute autre valeur sera considérée comme `'false'`

Exemple:
```javascript
...
    metadata: {
        filename: file.name,
        filetype: file.type,
        typeSignauxFaibles: 'extreme_cycling_downhill_on_volcano',
        private: 'true'
    },
...
```

## Installation
Prérequis: go > 1.8

`go get github.com/signaux-faibles/goup`

### Configuration
La configuration s'effectue avec un fichier au format toml à nommer config.toml et à placer dans le répertoire de travail de goup. En voici un exemple:
```
bind = "127.0.0.1:5000" 
jwtSecret = "don't keep this secret"
basePath = "/some/basePath"
```
#### bind
Adresse TCP sur laquelle va écouter le service goup
#### jwtSecret
Clé de signature des token JWT, doit être partagée avec le service d'authentification qui génère les tokens
#### basePath
Chemin de base où sera réalisé le stockage

### Stockage
#### Fichiers
Les fichiers sont stockés avec l'identifiant créé par le serveur tus dans 2 fichiers:
- un fichier de données avec une extension .bin
- un fichier de métadonnées avec une extension .info

Exemple de fichier
```json
{
    "ID": "f72a07d9d12aa070f5a18d7376865795",
    "Size": 646221764,
    "SizeIsDeferred": false,
    "Offset": 0,
    "MetaData": {
        "batch": "1903",
        "filename": "Vacances au Puy de Dome.mkv",
        "filetype": "video/x-matroska",
        "goup-path": "christophe",
        "private": "true",
        "typeSignauxFaibles": "extreme_cycling_downhill_on_volcano"
    },
    "IsPartial": false,
    "IsFinal": false,
    "PartialUploads": null
}

```

On retrouve dans ce fichier des informations relatives à l'état du fichier dans son traitement 
#### Répertoires, permissions
Les utilisateurs et répertoires de stockage doivent être créés au préalable ainsi que les sous-répertoires correspondant aux utilisateurs et leurs espaces privés. Les droits utilisateurs des répertoires doivent être fixés au préalable, le serveur se chargera de fixer les droits des fichiers importés.

Lors qu'un versement est terminé, goup se charge de créer les liens nécessaires pour que le fichier soit disponible dans le répertoire de l'utilisateur. Le fichier d'origine reste disponible au travers d'un lien dur dans le but de permettre au serveur tusd d'identifier le chargement de fichiers identiques.

Ci-dessous un exemple:

```
chemin                           user:group            mode 

.
+-- tusd                         goup:goup             770
|   +-- file1.bin                user1:users           660
|   +-- file1.info               user1:users           660
|   +-- file2.bin                user1:goup            660
|   +-- file2.info               user1:goup            660
|   +-- file3.bin                user2:users           660
|   +-- file3.info               user2:users           660
|   +-- file4.bin                user2:goup            660
|   +-- file4.info               user2:goup            660
+-- public                       user1:users           770
|   +-- file1.bin                user1:users           660
|   +-- file1.info               user1:users           660
|   +-- file3.bin                user2:users           660
|   +-- file3.info               user2:users           660
+-- user1                        user1:goup            770
|   +-- file2.bin                user1:goup            660
|   +-- file2.info               user1:goup            660
+-- user2                        user2:goup            770
|   +-- file4.bin                user2:goup            660
|   +-- file4.info               user2:goup            660
```

