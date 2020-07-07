# goup
Serveur minimaliste pour téléverser les fichiers bruts du projet Signaux Faibles

## Présentation
### goup
Goup est un microservice permettant l'upload de fichiers aux utilisateurs authentifiés sur présentation d'un jeton Keycloak disposant d'une signature valide et des informations nécessaires au traitement du fichier.
Ce microservice respecte le protocole tus, ce qui permet les fonctionnalités suivantes :
- pas de limite de taille du fichier
- l'envoi peut être interrompu et reprendre là où il s'était arrété
- l'envoi peut comporter des métadonnées
- on peut configurer pendant l'envoi la visibilité du fichier à l'ensemble des partenaires de la convention Signaux-Faibles
- l'envoi peut être segmenté en morceau de taille arbitraire pour respecter d'éventuelles limitations administratives sur la taille des requêtes HTTP
- il est possible d'envoyer plusieurs fichiers en parallèle

Goup est basé sur https://github.com/tus/tusd et incorpore certaines fonctionnalités supplémentaires :
- il n'est plus possible de télécharger un fichier depuis le serveur
- pendant l'envoi le fichier est placé dans un répertoire qui n'est pas accessible aux utilisateurs
- une fois l'envoi terminé, le fichier est disposé avec un lien dur dans un espace de stockage spécifique avec des droits utilisateur
- il est possible d'orienter un upload vers l'espace partagé entre les utilisateurs ou un espace privatif

### Présentation du protocole
1. Le client teste si le fichier est déjà présent avec une requête `HEAD`
2. Si le fichier n'existe pas, une requête `POST` comportant les métadonnées permet la création du fichier à vide
3. Des requêtes `PATCH` ou `POST` selon la configuration du client fournissent les données que le service inscrit dans le répertoire de stockage

### Stockage
L'implémentation actuelle se base sur un stockage en fichiers POSIX, toutefois l'implémentation du serveur tusd permet d'exploiter de nombreux types de stockage.

## Utilisation
### Client
Si vous souhaitez intégrer l'upload de fichier dans un projet, il existe des librairies qui implémentent le protocole dans de nombreux langages. Pour n'en citer que quelques-uns :
- JavaScript : https://github.com/tus/tus-js-client
- Python : https://github.com/tus/tus-py-client
- Java : https://github.com/tus/tus-java-client
- Go : https://github.com/eventials/go-tus
- PHP : https://github.com/ankitpokhrel/tus-php
- .NET : https://github.com/gerdus/tus-dotnet-client

### Exemple de code d'upload en JavaScript
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
Pour authentifier un upload avec goup, il est nécessaire d'attacher un jeton ou token d'authentification Keycloak dans les entêtes des requêtes tus, ce qui est pris en charge par les clients standards. Dans l'exemple précédent réalisé avec le client JavaScript :
```javascript
...
        headers: {
            Authorization: 'Bearer ' + currentToken  // currentToken contient le token d'authentification
        },
...
```

Pour obtenir ce token, il conviendra d'utiliser au préalable un client Keycloak pour se connecter à la plateforme signaux-faibles avec un identifiant d'utilisateur reconnu.
Comme pour tus, il existe des clients Keycloak dans différents langages : https://www.keycloak.org/docs/latest/securing_apps/#what-are-client-adapters.
L'habilitation à verser des fichiers dans goup sera portée dans le chargement du jeton.

Il est à noter que selon la durée de vie du jeton, il peut être nécessaire de mettre à jour le jeton pendant un upload, chaque requête est authentifiée individuellement.

### Structure du jeton
Le chargement du jeton est un objet JSON devant comporter une clé `goup_path` avec une chaîne de caractère identifiant le groupe d'utilisateurs POSIX du système cible. En conséquence de quoi :
- la clé `goup_path` doit correspondre à un nom de groupe d'utilisateurs POSIX du système
- en l'absence de cette clé, l'utilisateur sera considéré comme non habilité à envoyer des fichiers et se verra opposer des codes de retour `HTTP 403` lors des tentatives de création (`POST`)
- les uploads en cours sont situés dans `<basePath>/tusd` où basePath est la valeur fixée dans le fichier de configuration de goup
- une fois leurs envois terminés, le service créera des liens physiques vers ces fichiers dans le répertoire `<basePath>/<goup_path>` ou `<basePath>/public`
- leurs droits POSIX seront fixés de façon à limiter l'accès de ces fichiers au bon groupe d'utilisateurs POSIX du système (`<goup_path>` ou public) selon le niveau de partage choisi lors de l'upload

Keycloak a été configuré spécifiquement afin de pouvoir faire figurer l'information `goup_path` dans son jeton.
Pour ce faire, il a été défini dans la console d'administration pour le client utilisé une correspondance (mapper) de type "User Attribute" qui permet de lier l'attribut utilisateur `goup_path` au champ (claim) du jeton portant le même nom.
Au niveau de chaque utilisateur Keycloak, cet attribut est renseigné ou non selon les droits d'upload qu'on souhaite accorder.

### Metadonnées
Il est possible de fixer des métadonnées aux fichiers uploadés de façon arbitraire en suivant ces prérogatives :
- les métadonnées sont des chaînes de caractères
- la métadonnée `private` est interprétée par goup, si elle vaut `true` alors le serveur traitera le fichier de façon à empêcher son accès aux autres utilisateurs
- la métadonnée `goup-path` est réservée par le serveur pour traiter le chemin de stockage, elle est fixée à partir de la valeur transmise dans le jeton d'authentification. **Toute valeur fixée dans les métadonnées de l'envoi sera écrasée.**
- les métadonnées filename et filetype sont également réservées

### Envoi en mode privé
Ce mode permet d'envoyer un fichier sur la plateforme sans partager son contenu avec tous les utilisateurs de la plateforme.
Pour l'utiliser, il faut fixer la métadonnée `private` à la valeur `true` ; toute autre valeur sera considérée comme `false`.

Exemple :
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
Prérequis : go > 1.8

`go get github.com/signaux-faibles/goup`

### Configuration
La configuration s'effectue avec un fichier au format toml à nommer config.toml et à placer dans le répertoire de travail de goup. En voici un exemple repris dans config.toml.sample :
```
bind = "1.2.3.4:5678"
basePath = "/var/lib/goup_base"
hostname = "http://foo.bar:8081"
keycloakHostname = "http://foo.bar:8080"
keycloakRealm = "foo-realm"
clamavPath = "/usr/bin/clamscan"
smtpHost = "1.2.3.4:25"
fromEmailAddress = "foobar@foo.bar"
```
#### bind
Adresse TCP sur laquelle va écouter le service goup
#### basePath
Chemin de base où sera réalisé le stockage
#### hostname
URL pour les utilisateurs
#### keycloakHostname
URL de Keycloak
#### keycloakRealm
Domaine (realm) choisi pour les utilisateurs Keycloak
#### clamavPath
Chemin où est installé l'antivirus clamav
#### smtpHost
Serveur SMTP sans authentification
#### fromEmailAddress
Adresse email depuis laquelle les emails seront envoyés

### Stockage
#### Fichiers
Les fichiers sont stockés avec l'identifiant créé par le serveur tus dans 2 fichiers :
- un fichier de données sans extension
- un fichier de métadonnées avec une extension .info

Exemple de fichier .info :
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
    "PartialUploads": null,
    "Storage": {
        "Path": "/goup/tusd/f72a07d9d12aa070f5a18d7376865795",
        "Type": "filestore"
    }
}

```

On retrouve principalement dans ce fichier des informations relatives à l'état du fichier dans son traitement.

#### Répertoires, permissions
Les utilisateurs et répertoires de stockage doivent être créés au préalable ainsi que les sous-répertoires correspondant aux utilisateurs et leurs espaces privés. Les droits utilisateurs des répertoires doivent également être fixés au préalable, le serveur se chargera de fixer les droits des fichiers importés.

Lors qu'un upload est terminé, goup se charge de créer les liens physiques nécessaires pour que le fichier soit disponible dans le répertoire de l'utilisateur ou dans le répertoire partagé (public). Le fichier d'origine reste disponible au travers d'un lien dur dans le but de permettre au serveur tusd d'identifier le chargement de fichiers identiques.

Ci-dessous un exemple d'arborescence avec des groupes d'utilisateurs user1 et user2 :

```
chemin                               user:group            mode 
.
+-- <basePath>                       goup:goup             750
|   +-- tusd                         goup:goup             750
|   |   +-- file1                    goup:public           640
|   |   +-- file1.info               goup:public           640
|   |   +-- file2                    goup:user1            640
|   |   +-- file2.info               goup:user1            640
|   |   +-- file3                    goup:public           640
|   |   +-- file3.info               goup:public           640
|   |   +-- file4                    goup:user2            640
|   |   +-- file4.info               goup:user2            640
|   +-- public                       goup:public           750
|   |   +-- file1                    goup:public           640
|   |   +-- file1.info               goup:public           640
|   |   +-- file3                    goup:public           640
|   |   +-- file3.info               goup:public           640
|   +-- user1                        goup:user1            750
|   |   +-- file2                    goup:user1            640
|   |   +-- file2.info               goup:user1            640
|   +-- user2                        goup:user2            750
|   |   +-- file4                    goup:user2            640
|   |   +-- file4.info               goup:user2            640
```

avec :
- public/file1 lien de tusd/file1
- user1/file2 lien de tusd/file2
- public/file3 lien de tusd/file3
- user2/file4 lien de tusd/file4
