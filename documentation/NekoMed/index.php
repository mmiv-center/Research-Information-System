<!DOCTYPE html>
<html lang="en">
    <head>
        <meta charset="utf-8" />
        <meta name="viewport" content="width=device-width, initial-scale=1, shrink-to-fit=no" />
        <meta name="description" content="" />
        <meta name="author" content="" />
        <title>NekoMed</title>
        <link rel="icon" type="image/x-icon" href="assets/favicon.ico" />
        <!-- Font Awesome icons (free version)-->
        <script src="js/fontall.js" crossorigin="anonymous"></script>
        <!-- Google fonts-->
        <!-- <link href="https://fonts.googleapis.com/css?family=Varela+Round" rel="stylesheet" /> 
             <link href="https://fonts.googleapis.com/css?family=Nunito:200,200i,300,300i,400,400i,600,600i,700,700i,800,800i,900,900i" rel="stylesheet" /> -->
	     <link href="XRXX3I6Li01BKofIMNaDRs7nczIH.woff2" ref="stylesheet" />
        <!-- Core theme CSS (includes Bootstrap)-->
             <link href="css/styles.css" rel="stylesheet" />
	     <link href="css/default.min.css" rel="stylesheet" />
    </head>
    <body id="page-top">
        <!-- Navigation-->
        <nav class="navbar navbar-expand-lg navbar-light fixed-top" id="mainNav">
            <div class="container px-4 px-lg-5">
                <a class="navbar-brand" href="#page-top">NekoMed Hospital</a>
                <button class="navbar-toggler navbar-toggler-right" type="button" data-bs-toggle="collapse" data-bs-target="#navbarResponsive" aria-controls="navbarResponsive" aria-expanded="false" aria-label="Toggle navigation">
                    Menu
                    <i class="fas fa-bars"></i>
                </button>
                <div class="collapse navbar-collapse" id="navbarResponsive">
                    <ul class="navbar-nav ms-auto">
                        <li class="nav-item"><a class="nav-link" href="#about">Start</a></li>
                        <!-- <li class="nav-item"><a class="nav-link" href="#projects">Projects</a></li>
                        <li class="nav-item"><a class="nav-link" href="#signup">Contact</a></li> -->
                    </ul>
                </div>
            </div>
        </nav>
        <!-- Masthead-->
        <header class="masthead">
            <div class="container px-4 px-lg-5 d-flex h-100 align-items-center justify-content-center">
                <div class="d-flex justify-content-center">
                    <div class="text-center">
                      <!-- <h1 class="mx-auto my-0 text-uppercase">NekoMed</h1> -->
		      <img class="img-fluid justify-content-center" style="width: 60%;" src="assets/img/NekoMed.png" alt="NekoMed logo" />
                        <h2 class="text-white-50 mx-auto mt-2 mb-5">Workshop on radiology imaging. How to run medical image-based AI in the hospital.</h2>
                        <a class="btn btn-primary" href="#about">Get Started</a>
                    </div>
                </div>
            </div>
        </header>
        <!-- About-->
	

	
        <section class="about-section" id="about">
            <div class="container px-4 px-lg-5">
                <div class="row gx-4 gx-lg-5 justify-content-center">
                    <div class="col-lg-8">
                      <h2 class="text-white mb-4">Leader board</h2>
		      <p class="text-white" style="margin-bottom: 20px;">List of all teams that collected points in the MMIV Connectathon.
                        <div id="leaderboard"></div>
			</p>
                    </div>
                </div>
                <!-- <img class="img-fluid justify-content-center" src="assets/img/NekoMed.png" alt="..." /> -->
            </div>
        </section>

        <section class="about-section" id="about">
            <div class="container px-4 px-lg-5">
                <div class="row gx-4 gx-lg-5 justify-content-center">
                    <div class="col-lg-8">
                        <h2 class="text-white mb-4">Announce your connection (3 points)</h2>
                        <p class="text-white-50" style="margin-bottom: 20px;">
                          We will use our "NekoMed" network - a stand-alone Wifi network (pw: whitehazelnut). You have connected to NekoMed (<a href="http://192.168.0.80:4444">http://192.168.0.87:4444</a>). The IP of your computer appears to be <i style="color: orange;">
<?php if (isset($_SERVER["HTTP_X_FORWARDED_FOR"]) && $_SERVER["HTTP_X_FORWARDED_FOR"] != "") {
  $IP = $_SERVER["HTTP_X_FORWARDED_FOR"];
  $proxy = $_SERVER["REMOTE_ADDR"];
  $host = @gethostbyaddr($_SERVER["HTTP_X_FORWARDED_FOR"]);
} else {
  $IP = $_SERVER["REMOTE_ADDR"];
  $host = @gethostbyaddr($_SERVER["REMOTE_ADDR"]);
  }
echo($IP);
?></i>.</p>
<p class="text-white-50" style="margin-bottom: 20px;">Announce a DICOM connection on your computer. Use your IP and invent an AETitle (your team name), enter a port number and submit. You need to setup a receiver with this information (see next section). If this worked your entry should appear in the list of connections.
                        </p>
			<div>
			  <form class="justify-content-left">
			    <div class="form-group text-white">
			      <label for="exampleIP">IP address</label>
			      <input type="text" class="form-control" id="exampleIP" aria-describedby="ipHelp" placeholder="192.168.0.0">
			      <!-- <small id="emailHelp" class="form-text text-muted">We'll share your IP address</small> -->
			    </div>
			    <div class="form-group text-white">
			      <label for="exampleAE">Application Entity (AE-)Title (Team name)</label>
			      <input type="text" class="form-control" id="exampleAE" placeholder="FREYA">
			    </div>
			    <div class="form-group text-white">
			      <label for="examplePort">Port</label>
			      <input type="number" class="form-control" id="examplePort" placeholder="11112">
			    </div>
			    <button type="submit" id="submitConnection" class="btn btn-primary btn-small" style="margin-top: 5px; margin-bottom: 20px;">Submit</button>
			  </form>
			</div>
			<div style="width: 100%; background-color: orange; color: white; padding-left: 10px; border-top-left-radius: 5px; border-top-right-radius: 5px;">Connections</div>
			<div id="connections"></div>
                    </div>
                </div>
                <!-- <img class="img-fluid justify-content-center" src="assets/img/NekoMed.png" alt="..." /> -->
            </div>
        </section>

	<section class="about-section">
            <div class="container px-4 px-lg-5">
                <div class="row gx-4 gx-lg-5 justify-content-center">
                    <div class="col-lg-8">
                        <h2 class="text-white mb-4">Start a DICOM receiver (3 points)</h2>
                        <p class="text-white-50" style="margin-bottom: 20px;">
			  To receive DICOM data start storescp (part of DCMTK) in a separate terminal window. Replace "&lt;AETitle&gt;", "&lt;output directory&gt;" and &lt;PORT&gt;" with values from your computer. The storescp (DICOM STORE-SCP) program should keep running if the arguments are accepted.
			  <pre><code class="language-bash">
storescp -aet "&lt;AETitle&gt;" \
         --sort-on-study-uid "MYDATA" \
         --eostudy-timeout 16 \
         --exec-sync \
         --exec-on-eostudy "/path/to/process.py #r #a #c #p" \
         -od "&lt;output_directory&gt;" \
         -fe ".dcm" \
         &lt;PORT&gt;
			  </code></pre>
			</p>
			<p class="text-white-50" style="margin-top: -10px;">You will receive studies if storescp is working. Each received study will get you 0.1 points. Studies count even if they are send several times.</p>
		    </div>
		</div>
	    </div>
	</section>

	<section class="about-section">
            <div class="container px-4 px-lg-5">
                <div class="row gx-4 gx-lg-5 justify-content-center">
                    <div class="col-lg-8">
                        <h2 class="text-white mb-4">Report a study (10,000 points each)</h2>
                        <p class="text-white-50" style="margin-bottom: 20px;">
			  Each DICOM file contains information on the patient, study (visit), series (volume) and instance (image). Use python to print information into a log file (see the above call to process.py).
			  <pre><code class="language-python">
#!/usr/bin/env python

import pydicom as dicom
from datetime import datetime
import sys, os

callingIP=sys.argv[1]
callingAETitle=sys.argv[2]
calledAETitle=sys.argv[3]
path=sys.argv[4]
script_directory = os.path.dirname(os.path.abspath(sys.argv[0]))

for root, _, filenames in os.walk(path):
    for filename in filenames:
        try:
            ds = dicom.dcmread(path+"/"+filename, force=True)
        except IOError as e:
            continue
        PatientID=ds['PatientID'].value
        StudyInstanceUID=ds['StudyInstanceUID'].value
        SeriesInstanceUID=ds['SeriesInstanceUID'].value
        break

with open("%s/incoming_data.log" % (script_directory), "a") as log:
    log.write("%s: callingIP=%s, callingAETitle=%s, calledAETitle=%s PatientID=\"%s\" StudyInstanceUID=%s SeriesInstanceUID=%s\n" % (str(datetime.now()), callingIP, callingAETitle, calledAETitle, PatientID, StudyInstanceUID, SeriesInstanceUID))		      
			  </code></pre>			  
			</p>
			
			<p class="text-white-50" style="margin-bottom: 20px;">
			  Tips: Use the full path to process.py in the storescp call. Make your process.py file executable.
			  </p>
			<p class="text-white-50" style="margin-bottom: 20px;">
			  Tell us about a study you received to gain study points. Received studies will be listed in the incoming_data.log file written by the python file above.
			</p>
			<form class="justify-content-left">
			    <div class="form-group text-white">
			      <label for="exampleConnection">Select your DICOM destination</label>
			      <select class="form-control" id="exampleConnection" aria-describedby="ipHelp" placeholder="1.2.300...">
				<option>Select your connection</option>
			      </select>
			    </div>			  
			    <div class="form-group text-white">
			      <label for="exampleStudyInstanceUID">Study Instance UID</label>
			      <input type="text" class="form-control" id="exampleStudyInstanceUID" aria-describedby="ipHelp" placeholder="1.2.300...">
			    </div>
			    <div class="form-group text-white">
			      <label for="examplePatientID">PatientID</label>
			      <input type="text" class="form-control" id="examplePatientID" aria-describedby="examplePatientID" placeholder="FIONA">
			    </div>
			    <div class="form-group text-white">
			      <label for="exampleSeriesInstanceUID">Series Instance UID</label>
			      <input type="text" class="form-control" id="exampleSeriesInstanceUID" placeholder="1.2.3333...">
			    </div>
			    <button type="submit" id="submitStudy" class="btn btn-primary btn-small" style="margin-top: 5px; margin-bottom: 20px;">Submit (10,000 points)</button>
			  </form>
			
		    </div>
		</div>
	    </div>
	</section>

	<section class="about-section">
            <div class="container px-4 px-lg-5">
                <div class="row gx-4 gx-lg-5 justify-content-center">
                    <div class="col-lg-8">
                        <h2 class="text-white mb-4">Boost your point count</h2>
                        <p class="text-white-50" style="margin-bottom: 20px;">
			  You received DICOM files with storescp and you triggered a processing step with process.py. In the next step increase your points by sending data back. Sending data is done by storescu (DCMTK program).
			  <pre><code class="language-bash">
storescu -aet "&lt;calling_AETitle&gt;" \
         -aec "&lt;called_AETitle&gt;" \
         +r +sd -nh \
         &lt;destination_IP&gt; &lt;destination_PORT&gt; \
         "&lt;dicom_directory&gt;
			  </code></pre>
			</p>
                        <p class="text-white-50" style="margin-bottom: 20px;">
			  Notice that you need to specify your own AETitle (calling AETitle) and the AETitle of the receiving system (called AETitle).
			</p>
		    </div>
		</div>
	    </div>
	</section>

	
        <!-- Projects-->
        <section class="projects-section bg-light" id="projects">
            <div class="container px-4 px-lg-5">
                <!-- Featured Project Row-->
                <div class="row gx-0 mb-4 mb-lg-5 align-items-center">
                    <div class="col-xl-8 col-lg-7"><img class="img-fluid mb-3 mb-lg-0" src="assets/img/bg-masthead.jpg" alt="..." /></div>
                    <div class="col-xl-4 col-lg-5">
                        <div class="featured-text text-center text-lg-left">
                            <h4>Workshop</h4>
                            <p class="text-black-50 mb-0">Thank you for participating</p>
                        </div>
                    </div>
                </div>
                <!-- Project One Row-->
                <div class="row gx-0 mb-5 mb-lg-0 justify-content-center">
                    <div class="col-lg-6"><img class="img-fluid" src="assets/img/demo-image-01.jpg" alt="..." /></div>
                    <div class="col-lg-6">
                        <div class="bg-black text-center h-100 project">
                            <div class="d-flex h-100">
                                <div class="project-text w-100 my-auto text-center text-lg-left">
                                    <h4 class="text-white">Next steps</h4>
                                    <p class="mb-0 text-white-50">Create reports as DICOM files. Add your own AI.</p>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
                <!-- Project Row Three-->
                <div class="row gx-0 justify-content-center">
                    <div class="col-lg-6"><img class="img-fluid" src="assets/img/demo-image-02.jpg" alt="..." /></div>
                    <div class="col-lg-6 order-lg-first">
                        <div class="bg-black text-center h-100 project">
                            <div class="d-flex h-100">
                                <div class="project-text w-100 my-auto text-center text-lg-right">
                                    <h4 class="text-white">Publish your work</h4>
                                    <p class="mb-0 text-white-50">Let us know if you do something with this setup.</p>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
                <div class="row gx-0 mb-5 mb-lg-0 justify-content-center">
                    <div class="col-lg-6"><img class="img-fluid" src="assets/img/demo-image-03.jpg" alt="..." /></div>
                    <div class="col-lg-6">
                        <div class="bg-black text-center h-100 project">
                            <div class="d-flex h-100">
                                <div class="project-text w-100 my-auto text-center text-lg-left">
                                    <h4 class="text-white">Learn more abouts</h4>
                                    <p class="mb-0 text-white-50">Multi-modal imaging, image registration, image segmentation, DICOM structured reports, ...</p>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
                <!-- Project Two Row-->
            </div>
        </section>
        <!-- Contact-->
        <section class="contact-section bg-black">
            <div class="container px-4 px-lg-5">
                <div class="row gx-4 gx-lg-5">
                    <div class="col-md-4 mb-3 mb-md-0">
                        <div class="card py-4 h-100">
                            <div class="card-body text-center">
                                <i class="fas fa-map-marked-alt text-primary mb-2"></i>
                                <h4 class="text-uppercase m-0">Contact Institution</h4>
                                <hr class="my-4 mx-auto" />
                                <div class="small text-black-50"><a href="https://mmiv.no">Mohn Medical Imaging and Visualization Centre, Department of Radiology, Haukeland University Hospital, Bergen, Norway</a></div>
                            </div>
                        </div>
                    </div>
                    <div class="col-md-4 mb-3 mb-md-0">
                        <div class="card py-4 h-100">
                            <div class="card-body text-center">
                                <i class="fas fa-envelope text-primary mb-2"></i>
                                <h4 class="text-uppercase m-0">Email</h4>
                                <hr class="my-4 mx-auto" />
                                <div class="small text-black-50"><a href="#!">Hauke.Bartsch@helse-bergen.no</a></div>
                            </div>
                        </div>
                    </div>
                    <div class="col-md-4 mb-3 mb-md-0">
                        <div class="card py-4 h-100">
                            <div class="card-body text-center">
                                <i class="fas fa-mobile-alt text-primary mb-2"></i>
                                <h4 class="text-uppercase m-0">Sources</h4>
                                <hr class="my-4 mx-auto" />
                                <div class="small text-black-50">Logo was created by "Draw Things with AI", all other images have been taken on trips to Tromsø in 2023.</div>
                            </div>
                        </div>
                    </div>
                </div>
   <!--             <div class="social d-flex justify-content-center">
                    <a class="mx-2" href="#!"><i class="fab fa-twitter"></i></a>
                    <a class="mx-2" href="#!"><i class="fab fa-facebook-f"></i></a>
                    <a class="mx-2" href="#!"><i class="fab fa-github"></i></a>
                </div>-->
            </div>
        </section>
        <!-- Footer-->
        <footer class="footer bg-black small text-center text-white-50"><div class="container px-4 px-lg-5">Copyright &copy; MMIV 2023</div></footer>
        <!-- Bootstrap core JS-->
        <script src="js/bootstrap.bundle.min.js"></script>
	<script src="js/highlight.min.js"></script>
	<script src="js/bash.min.js"></script>
	<script src="js/python.min.js"></script>
        <!-- Core theme JS-->
        <script src="js/scripts.js"></script>
        <script src="js/jquery.min.js"></script>
        <script src="js/all.js"></script>
        <!-- * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *-->
        <!-- * *                               SB Forms JS                               * *-->
        <!-- * * Activate your form at https://startbootstrap.com/solution/contact-forms * *-->
        <!-- * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *-->
        <!-- <script src="js/sb-forms-latest.js"></script> -->
    </body>
</html>
