<template>
  <v-app>
    <div class="title">
      <span id="goupBlue">Go</span>
      <span id="goupRed">up</span>
      <br>
      <br>
      <span id="subtitle">client</span>
    </div>
    <v-container>
      <v-layout row wrap>
        <v-flex class="pa-2" xs6>
          <v-card class="pa-4" style="height:500px; text-align: center">
            <v-card-title>Login</v-card-title>
            <v-text-field label="Login endpoint" v-model="loginEndpoint"></v-text-field>
            <v-text-field label="user" v-model="email"></v-text-field>
            <v-text-field label="password" type="password" v-model="password"></v-text-field>
            <v-btn color="#003189" dark @click="login">Login</v-btn>
            <v-textarea readonly v-model="Jwt" label="JWT Payload"></v-textarea>
          </v-card>
        </v-flex>
        <v-flex class="pa-2" xs6>
          <v-card class="pa-4" style="height:500px">
            <v-card-title>Upload de fichier</v-card-title>
            <v-text-field label="Service endpoint" v-model="endpoint"></v-text-field>
            <v-text-field label="Auth token" v-model="token"></v-text-field>
            <v-textarea
              :label="'metadonnées additionnelles (JSON) ' + ((mderror)?'error':'')"
              rows="5"
              :style="'color:' + (mderror?'\'red\'':'\'black\'')"
              v-model="metadata"
            ></v-textarea>
            <div style="display: flex; justify-content: left; vertical-align: middle;">
              <v-checkbox label="privé" v-model="privateFile" st/>
                <UploadButton
                  color="red"
                  style="width:35%; color: #fff"
                  :fileChangedCallback="setFile"
                  title="fichier"
                />
              <div style="vertical-align: middle;">
                {{(file || {name: "pas de fichier sélectionné"}).name}}
                <br>
                {{ (file || {size: ""}).size}} {{file?"octets":""}}
              </div>
            </div>
            <div style="display: flex; justify-content: center;">
              <v-btn color="#003189" :disabled="fileSet||mderror" dark @click="upload">envoyer</v-btn>
              <v-progress-circular v-if="uploading" :value="progress" color="brown"></v-progress-circular>
            </div>
          </v-card>
        </v-flex>
        <v-flex class="pa-2" xs12>
          <v-card class="pa-4">
            <v-tabs centered color="transparent" slider-color="white">
              <v-tab key="log">Log</v-tab>
              <v-tab key="store">File Store</v-tab>

              <v-tab-item key="log">
                <v-textarea readonly v-model="journal" rows="40" hint="Hint text" :reverse="true"></v-textarea>
              </v-tab-item>
              <v-tab-item key="store">
                <div style="width: 100%">
                  <v-btn color="#003189" dark @click="refreshFiles">Rafraichir</v-btn>
                </div>
                <table width="100%" border="1px solid #333" style="border-collapse:collapse; border: 1px solid #333">
                  <tr>
                    <th>Chemin</th>
                    <th>Taille</th>
                    <th>Propriétaire</th>
                    <th>Groupe</th>
                    <th>Mode</th>
                    <th>Informations TUSD</th>
                  </tr>
                  <tr v-for="f in files" :key="f.filename">
                    <td>{{ f.filename }}</td>
                    <td>{{ f.size }}</td>
                    <td>{{ f.owner }}</td>
                    <td>{{ f.group }}</td>
                    <td>{{ f.mode }}</td>
                    <td><v-textarea rows=1 readonly :value="JSON.stringify((f.tusdInfo||{}).MetaData, null, 2)"></v-textarea></td>
                  </tr>
                </table>
              </v-tab-item>
            </v-tabs>
          </v-card>
        </v-flex>
      </v-layout>
    </v-container>
  </v-app>
</template>

<script>
import UploadButton from "vuetify-upload-button";
import tus from "tus-js-client";
import axios from "axios";

var client = axios.create({
  headers: {
    "Content-Type": "application/json"
  }
});

export default {
  name: "App",
  components: {
    UploadButton
  },
  computed: {
    fileSet() {
      return this.file == null;
    },
    filesRead: {
      get() {
        if (this.files.length > 0) {
          return JSON.stringify(this.files, null, 2);
        } else {
          return "";
        }
      },
      set() {}
    },
    metadata: {
      get() {
        return JSON.stringify(this.localMetadata, null, 2);
      },
      set(m) {
        try {
          this.localMetadata = JSON.parse(m);
          this.mderror = false;
        } catch (error) {
          this.mderror = true;
        }
      }
    },
    Jwt() {
      if (this.token) {
        var base64Url = this.token.split(".")[1];
        var base64 = base64Url.replace(/-/g, "+").replace(/_/g, "/");
        var payload = JSON.parse(window.atob(base64));
        return JSON.stringify(payload, null, 2);
      } else {
        return "";
      }
    }
  },
  mounted() {
    this.refreshFiles();
  },
  methods: {
    refreshFiles() {
      var e = this.endpoint.slice(0, this.endpoint.length - 6) + "list";
      client.get(e).then(r => {
        this.files = r.data;
      });
    },
    login() {
      var params = {
        email: this.email,
        password: this.password
      };
      client
        .post(this.loginEndpoint, params)
        .then(r => {
          this.token = r.data.token;
          this.journal = this.date() + ": Authentification OK\n" + this.journal;
        })
        .catch(() => {
          this.token = null;
          this.journal =
            this.date() + ": Authentification NOK\n" + this.journal;
        });
    },
    setFile(file) {
      this.file = file;
    },
    date() {
      var today = new Date();
      var dd = today.getDate();
      var mm = today.getMonth() + 1; //January is 0!
      var yyyy = today.getFullYear();
      var hh = today.getHours();
      var mn = today.getMinutes();
      var ss = today.getSeconds();
      var ms = today.getMilliseconds();

      if (dd < 10) {
        dd = "0" + dd;
      }

      if (mm < 10) {
        mm = "0" + mm;
      }

      if (hh < 10) {
        hh = "0" + hh;
      }
      if (mn < 10) {
        mn = "0" + mn;
      }

      if (ss < 10) {
        mn = "0" + mn;
      }

      if (ms < 100) {
        ms = "0" + ms;
      }
      if (ms < 10) {
        ms = "0" + ms;
      }
      today =
        yyyy + "-" + mm + "-" + dd + " " + hh + ":" + mn + ":" + ss + "." + ms;
      return today;
    },
    upload() {
      var self = this;
      var metadata = {};
      Object.assign(metadata, this.localMetadata);
      Object.assign(metadata, {
        filename: this.file.name,
        filetype: this.file.type,
        private: self.privateFile ? "true" : "false"
      });

      var upload = new tus.Upload(this.file, {
        endpoint: this.endpoint,
        retryDelays: [0, 3000, 5000, 10000, 20000],
        metadata: metadata,
        headers: {
          Authorization: "Bearer " + this.token
        },
        chunkSize: 4000000,
        onError: function(error) {
          self.journal =
            self.date() + ": Echec -> " + error + "\n" + self.journal;
          self.Progress = 0
          self.uploading = false;
        },

        onProgress: function(bytesUploaded, bytesTotal) {
          var percentage = ((bytesUploaded / bytesTotal) * 100).toFixed(2);
          self.progress = parseInt(percentage);
          self.journal =
            self.date() +
            ": Envoi en cours -> " +
            bytesUploaded +
            " sur " +
            bytesTotal +
            ", soit " +
            percentage +
            "%\n" +
            self.journal;
        },
        onSuccess: function() {
          self.journal =
            self.date() +
            ": Envoi effectué -> " +
            upload.file.name +
            "\n" +
            self.journal;
          self.Progress = 0
          self.uploading = false;
        }
      });

      this.uploading = true;
      upload.start();
    }
  },
  data() {
    return {
      mderror: false,
      localMetadata: {},
      privateFile: false,
      loginEndpoint: "https://goup.signaux-faibles.beta.gouv.fr/login",
      endpoint: "https://goup.signaux-faibles.beta.gouv.fr/files/",
      email: "",
      password: "",
      file: null,
      files: [],
      token: null,
      uploading: false,
      progress: 0,
      journal: this.date() + ": Démarrage\n"
    };
  }
};
</script>

<style scoped>
@import url("https://fonts.googleapis.com/css?family=Pacifico");
div.title {
  display: block;
  width: 100%;
  margin: 30px;
  text-align: center;
}
#goupBlue {
  font-family: "Pacifico";
  font-size: 40px;
  color: #003189;
}
#goupRed {
  font-family: "Pacifico";
  font-size: 40px;
  color: #e2011c;
}
#subtitle {
  margin-top: 30px;
  color: #666;
  font-weight: 100;
}
td {
  padding: 5px;
}
</style>