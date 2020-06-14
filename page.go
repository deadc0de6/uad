/*
author: deadc0de6 (https://github.com/deadc0de6)
Copyright (c) 2020, deadc0de6
*/

package main

var Page = `<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <meta http-equiv="X-UA-Compatible" content="ie=edge" />
    <style>
			body {
				background-color: black;
				color: white;
			}
			a {
				color: #2c87f0;
			}
			a:visited {
				color: #636;
			}
			a:hover, a:active, a:focus {
				color:#c33;
			}
      #drophere {
        border: 2px dashed #ccc;
        border-radius: 20px;
        width: 480px;
        font-family: sans-serif;
        margin: 100px auto;
        padding: 20px;
      }

      #drophere.highlight {
        background-color: grey;
      }

			table.center {
				margin-left:auto;
				margin-right:auto;
			}

      th,
      td {
        padding: 10px;
        border: solid 1px;
        text-align: center;
      }
      .wide {
        width: 300px;
      }
    </style>
    <title>{{ .Title }}</title>
  </head>
  <body>
		{{ if .EnableUploads }}
    <div id="drophere">
      <form enctype="multipart/form-data" action="/upload" method="post">
        <input type="file" name="file"/>
        <input type="submit" value="upload"/>
      </form>
    </div>
		{{ end }}
		{{ if .EnableDownloads }}
    <div>
			{{ if .Files }}
      <table class="center">
        <tr>
          <th>Name</th>
          <th>Size</th>
          <th>Uploaded</th>
        </tr>
        {{range .Files}}
        <tr>
          <td class="wide"><a href="{{.Path}}">{{.Name}}</a></td>
          <td>{{.Size}}</td>
          <td>{{.Modified}}</td>
        </tr>
        {{end}}
      </table>
			{{ end }}
    </div>
		{{ end }}
    <!--
    <div>
      <a href="/files/">Uploaded files</a>
    </div>
    -->
    <script>
      let drophere = document.getElementById('drophere');

      // prevent browser from displaying the file itself
      ['dragenter', 'dragover', 'dragleave', 'drop'].forEach(eventName => {
        drophere.addEventListener(eventName, preventDefaults, false);
      });

      function preventDefaults (e) {
        e.preventDefault();
        e.stopPropagation();
      }

      // handle highlighting
      ['dragenter', 'dragover'].forEach(eventName => {
        drophere.addEventListener(eventName, highlight, false);
      });

      ['dragleave', 'drop'].forEach(eventName => {
        drophere.addEventListener(eventName, unhighlight, false);
      });

      function highlight(e) {
        drophere.classList.add('highlight');
      }

      function unhighlight(e) {
        drophere.classList.remove('highlight');
      }

      // handle drops
      drophere.addEventListener('drop', handleDrop, false);

      function handleDrop(e) {
        let dt = e.dataTransfer;
        let files = dt.files;
        handleFiles(files);
      }

      function handleFiles(files) {
        ([...files]).forEach(uploadFile);
      }

      function uploadFile(file) {
        let url = '/upload';
        let formData = new FormData();

        formData.append('file', file);

        fetch(url, {
          method: 'POST',
          body: formData
        })
        .then(() => {
          // succeed
          location.reload();
        })
        .catch(() => {
          // error
        })
      }
    </script>
  </body>
</html>`
