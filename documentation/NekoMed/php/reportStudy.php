<?php
  // report a study
  $entry = "";
  if (isset($_GET['entry'])) {
     $entry = $_GET['entry'];
  }

  $PatientID = "";
  if (isset($_GET['PatientID'])) {
     $PatientID = $_GET['PatientID'];
  }
  $StudyInstanceUID = "";
  if (isset($_GET['StudyInstanceUID'])) {
     $StudyInstanceUID = $_GET['StudyInstanceUID'];
  }
  $SeriesInstanceUID = "";
  if (isset($_GET['SeriesInstanceUID'])) {
     $SeriesInstanceUID = $_GET['SeriesInstanceUID'];
  }

  if ($PatientID === "" || $StudyInstanceUID === "" || $SeriesInstanceUID === "" || $entry === "") {
     echo(json_encode(array("message" => "Error: insufficient arguments.")));
     exit(0);
  }
  // check if those variables exist in our data
  $study_cache_file = "../crontab/study_cache.json";
  $study_cache = array();
  if (file_exists($study_cache_file)) {
     $study_cache = json_decode(file_get_contents($study_cache_file), TRUE);
  }
  $found = false;
  foreach($study_cache as $study) {
     if ($study["PatientID"] === $PatientID && $study["StudyInstanceUID"] === $StudyInstanceUID && $study["SeriesInstanceUID"] === $SeriesInstanceUID) {
        $found = true;
	break;
     }    
  }
  if (!$found) {
     echo(json_encode(array("message" => "Error: unknown study.")));
     exit(0);
  }  

  // add points for this study
  if ($found) {
     // add to connection.json and
     $con_file = "../php/connections.json";
     if (is_file($con_file)) {
        $con = json_decode(file_get_contents($con_file), TRUE);
	// find our connection based on the IP?
	$changed = FALSE;
	foreach($con as &$c) {
	  $k = $c["IP"].$c["AETitle"].$c["Port"];
	  if ($k !== $entry) {
	     continue;
	  }
	  // we are now in our study
	  // already in there?
	  $existingStudies = $c["ReceivedStudy"];
	  $alreadyAdded = FALSE;
	  foreach ($existingStudies as $existingStudy) {
	     if ($existingStudy["PatientID"] === $PatientID && $existingStudy["StudyInstanceUID"] === $StudyInstanceUID && $existingStudy["SeriesInstanceUID"] == $SeriesInstanceUID) {
	        $alreadyAdded = TRUE;
		break;
	     }
	  }
	  if ($alreadyAdded) {
	       echo(json_encode(array("message" => "Error: This study/series has already been added.")));
     	       exit(0);
	  }

  	  $c["ReceivedStudy"][] = array("PatientID" => $PatientID, "StudyInstanceUID" => $StudyInstanceUID, "SeriesInstanceUID" => $SeriesInstanceUID);
	  file_put_contents($con_file, json_encode($con));
          echo(json_encode(array("report" => "Done! Check your scores!")));	  
	  exit(0);
	}
	echo(json_encode(array("message" => "Error: this connection does not exist.")));
	exit(0);	
     } else {
	echo(json_encode(array("message" => "Error: no connection file found.")));
	exit(0);	
     }    
  }
?>