from http.server import BaseHTTPRequestHandler, HTTPServer
from functools import partial
from zipfile import ZipFile
from io import BytesIO

import os
import json

hostName = "localhost"
serverPort = 8080

#
# Put your source code here. One function to load the model. One function to run it.
#
class Model():
    def __init__(self, *args, **kwargs):
        """Load a model."""
        print("We initialize our model here")
        pass

    def predict(self, input, output):
        """Predict on the input folder, write to the output folder."""
        print("We predict something")
        pass


class MyServer(BaseHTTPRequestHandler):
    def __init__(self, model, *args, **kwargs):
        self.model = model
        # BaseHTTPRequestHandler calls do_GET **inside** __init__ !!!
        # So we have to call super().__init__ after setting attributes.
        super().__init__(*args, **kwargs)

    def do_PUT(self):
        """Save a file following a HTTP PUT request"""
        filename = os.path.basename(self.path)

        # Don't overwrite files
        if os.path.exists(filename):
            self.send_response(409, 'Conflict')
            self.end_headers()
            reply_body = '"%s" already exists\n' % filename
            self.wfile.write(reply_body.encode('utf-8'))
            return

        file_length = int(self.headers['Content-Length'])
        # create input folder

        # create output folder

        # we don't want to save the zip file to disk but unpack
        myzip = ZipFile(BytesIO(self.rfile.read(file_length)))
        for contained_file in myzip.namelist():
            # save this file to disk
            with open(contained_file, 'wb') as f:
                f.write(myzip.open(contained_file).read())
            pass

        #with open(filename, 'wb') as output_file:
        #    output_file.write(self.rfile.read(file_length))
        self.send_response(201, 'Created')
        self.end_headers()
        reply_body = 'Saved "%s"\n' % filename
        self.wfile.write(reply_body.encode('utf-8'))
        # and predict on this folder
        input = ""
        output = ""
        self.model.predict(input, output)

if __name__ == "__main__":

    model = Model()
    handler = partial(MyServer, model)

    webServer = HTTPServer((hostName, serverPort), handler)
    print("Server started http://%s:%s" % (hostName, serverPort))
    print(" We receive a file like this: curl -X PUT --upload-file Dockerfile http://localhost:8080")

    try:
        webServer.serve_forever()
    except KeyboardInterrupt:
        pass

    webServer.server_close()
    print("Server stopped.")
