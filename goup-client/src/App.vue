<template>
  <v-app>
    <div class="title">
      <span id="goupRed">Go</span>
      <span id="goupBlue">up</span><br/><br/>
      <span id="subtitle">client</span>
    </div>
    <v-container>
      <v-layout row wrap>
        <v-flex class="pa-2" xs6 >
          <v-card class="pa-4" style="height:400px"> 
            <v-card-title>Login</v-card-title>
            <v-text-field label="Login endpoint" v-model="loginEndpoint"></v-text-field>
            <v-text-field label="user" v-model="email"></v-text-field>
            <v-text-field label="password" v-model="password"></v-text-field>
            <v-btn @click="login">Login</v-btn>
          </v-card>
        </v-flex>
        <v-flex class="pa-2" xs6>
          <v-card class="pa-4" style="height:400px">
            <v-card-title>Upload de fichier</v-card-title>
            <v-text-field label="Service endpoint" v-model="endpoint"></v-text-field>
            <v-text-field label="Auth token" v-model="token"></v-text-field>
            <v-checkbox label="privé" v-model="privateFile"/>
            <div style="display: flex">
              <UploadButton
                color="primary"
                style="width:50%" 
                :fileChangedCallback="setFile" 
                title="fichier" />
              <v-btn @click="upload">send</v-btn>
            </div>
          </v-card>
        </v-flex>
        <v-flex class="pa-2" xs12>
          <v-card class="pa-4">
            <v-textarea
              clearable
              label="Journal"
              v-model="journal"
              rows = 40
              hint="Hint text"
              reverse=true
             ></v-textarea>
          </v-card>
        </v-flex>
      </v-layout>
    </v-container>
  </v-app>
</template>

<script>
import UploadButton from 'vuetify-upload-button';
import tus from 'tus-js-client'
import axios from 'axios'

var client = axios.create(
  {
    headers: {
      'Content-Type': 'application/json'
    },
  }
)

export default {
  name: 'App',
  components: {
      UploadButton
  },
  methods: {
    login () {
      var params = {
        email: this.email,
        password: this.password,
      }
      client.post(this.loginEndpoint, params).then(r => {
        this.token = r.data.token
        this.journal = this.date() + ': Authentification OK\n' + this.journal
      }).catch(() => {
        this.token = null
        this.journal = this.date() + ': Authentification NOK\n' + this.journal
      })
    },
    setFile (file) {
      this.file = file
    },
    date () {
      var today = new Date()
      var dd = today.getDate()
      var mm = today.getMonth()+1 //January is 0!
      var yyyy = today.getFullYear()
      var hh = today.getHours()
      var mn = today.getMinutes()
      var ss = today.getSeconds()
      var ms = today.getMilliseconds()

      if(dd<10) {
        dd = '0'+dd
      } 

      if(mm<10) {
        mm = '0'+mm
      } 

      if(hh<10) {
        hh = '0' + hh
      }
      if(mn<10) {
        mn = '0' + mn
      }

      if(ss<10) {
        mn = '0' + mn
      }

      if(ms<100) {
        ms = '0' + ms
      }
      if(ms<10) {
        ms = '0' + ms
      }
      today = yyyy + '-' + mm + '-' + dd + ' ' + hh + ':' + mn + ':' + ss + '.' + ms  
      return today
    },
    upload () {
      var self = this
      var upload = new tus.Upload(this.file, {
        endpoint: this.endpoint,
        retryDelays: [0, 3000, 5000, 10000, 20000],
        metadata: {
            filename: this.file.name,
            filetype: this.file.type,
            private: self.privateFile?"true":"false",
            type: 'debit',
            batch: '1903'
        },
        origin: "https://goup.signaux-faibles.beta.gouv.fr",
        headers: {
            Authorization: 'Bearer ' + this.token
        },
        chunkSize: 4000000,
        onError: function(error) {
            self.journal = self.date() + ": Echec -> " + error + '\n' + self.journal
        },
        onProgress: function(bytesUploaded, bytesTotal) {
            var percentage = (bytesUploaded / bytesTotal * 100).toFixed(2)
            self.journal = self.date() + ": Envoi en cours -> " + bytesUploaded + ' sur ' + bytesTotal + ', soit ' + percentage + '%\n' + self.journal
        },
        onSuccess: function() {
            self.journal = self.date() + ": Envoi effectué -> " + upload.file.name + '\n' + self.journal
        }
      })
      upload.start()
    }
  },
  data () {
    return {
      privateFile: false,
      loginEndpoint: 'https://goup.signaux-faibles.beta.gouv.fr/login',
      endpoint: 'https://goup.signaux-faibles.beta.gouv.fr/files/',
      email: '',
      password: '',
      currentFile: null,
      file: null,
      token: null,
      journal: this.date() + ': Démarrage\n'
    }
  }
}
</script>

<style scoped>
@import url('https://fonts.googleapis.com/css?family=Pacifico');
div.title {
  display: block;
  width: 100%;
  margin: 30px;
  text-align: center;
}
#goupRed {
  font-family: "Pacifico";
  font-size: 40px;
  color: #003189
}
#goupBlue {
  font-family: "Pacifico";
  font-size: 40px;
  color: #e2011c
}
#subtitle {
  margin-top: 30px;
  color: #666;
  font-weight: 100;
}
</style>