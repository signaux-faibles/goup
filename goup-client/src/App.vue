<template>
  <v-app>
      <v-text-field label="endpoint" model="endpoint" value="http://localhost:5000/files/"></v-text-field>
      <v-text-field label="token" model="token" value="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbWFpbCI6ImFkbWluQHRlc3QuY29tIiwiZXhwIjoxNTU1NjgyMjM2LCJvcmlnX2lhdCI6MTU1NTY3ODYzNiwic2NvcGUiOlsiYWRtaW4iLCJhZG1pbkB0ZXN0LmNvbSJdLCJ2YWx1ZSI6eyJ1cGxvYWRfcGF0aCI6ImFkbWluIn19.bxFiPyJfFmPGFbr2hKNyTRASXAzYDOB-8hFzaCrggUo"></v-text-field>
      <UploadButton :fileChangedCallback="setFile"/>
      <v-btn @click="test">test</v-btn>
      <v-btn @click="upload">send</v-btn>
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
    setFile (file) {
      this.file = file
      console.log(file)
    },
    test () {
      client.defaults.headers.common['Authorization'] = `Bearer ` + this.token
      console.log(this.token)
      client.get("http://localhost:5000/list").then(r => {console.log(r)})
    },
    upload () {
      var upload = new tus.Upload(this.file, {
        endpoint: this.endpoint,
        retryDelays: [0, 3000, 5000, 10000, 20000],
        metadata: {
            filename: this.file.name,
            filetype: this.file.type
        },
        origin: "http://localhost:8080",
        headers: {
            Authorization: 'Bearer ' + this.token
        },
        metadata: {
          'type': 'debit',
          'batch': '1903'
        },
        chunkSize: 4000000,
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
      upload.start()
    }
  },
  data () {
    return {
      endpoint: 'http://localhost:5000/files/',
      currentFile: null,
      file: null,
      token: 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbWFpbCI6ImFkbWluQHRlc3QuY29tIiwiZXhwIjoxNTU1NjgyMjM2LCJvcmlnX2lhdCI6MTU1NTY3ODYzNiwic2NvcGUiOlsiYWRtaW4iLCJhZG1pbkB0ZXN0LmNvbSJdLCJ2YWx1ZSI6eyJ1cGxvYWRfcGF0aCI6ImFkbWluIn19.bxFiPyJfFmPGFbr2hKNyTRASXAzYDOB-8hFzaCrggUo'
    }
  }
}
</script>
