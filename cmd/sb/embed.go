package main

const htmlTemplateString = `<!doctype html>
<html lang="en">
  <head>
    <!-- Required meta tags -->
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1, shrink-to-fit=no">
    
    <!-- Bootstrap CSS -->
    <link rel="stylesheet" href="https://stackpath.bootstrapcdn.com/bootstrap/4.5.2/css/bootstrap.min.css" integrity="sha384-JcKb8q3iqJ61gNV9KGb8thSsNjpSL0n8PARn9HuZOnIxN0hoP+VmmDGMN5t9UJ0Z" crossorigin="anonymous">

    <title>{{.Homework.Name}}</title>
    
    <style>
      body {
        font-family: -apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,Oxygen-Sans,Ubuntu,Cantarell,"Helvetica Neue",sans-serif;
        font-weight: normal;
        font-size: 14px;
      }
      th:first-child, td:first-child {
        position: sticky;
        left: 0px;
        background-color: white;
        box-shadow: 0px -0.5px rgb(222,226,230);
      }
      .table-responsive {
        padding: 0;
      }
      tbody tr:hover th {
        background-color: rgb(236,236,236);
      }
      .table td, th {
        text-align: center;
      }
      .table thead tr th {
        border-top-width: 1px;
        border-top: 0px;
      }
      td.penalty::before {
        content: "+";
      }
	  .notation {
        padding: 5px 20px;
        display: inline-block;
      }
    </style>
  </head>
  
  <body>
    <nav class="navbar navbar-light" style="background-color: #FF9800;">
      <a class="navbar-brand" href="#" style="color: white">{{.Homework.Name}} Scoreboard</a>
    </nav>
    <br>
    
    <div style="padding: 0px 20px">
	  <h4>Legend:</h4>
	  <ul>
      <li>
				<span class="notation" style="background-color: rgb(56, 142, 60)">AC</span>
				<span class="notation" style="background-color: rgb(255,193,7)">TLE</span>
				<span class="notation" style="background-color: #FF9800">TLE+</span>
				<span class="notation" style="background-color: rgb(244,67,54)">NA</span>
			</li>
			<li>Each testcase will run for 10s+time limit. If the program runs 10s more than the time limit, it will be
					terminate and show the result with TLE+</li>
			<li>NA is not accepted. It can means wrong answer, segmentation fault or runtime error.</li>
			<li>The rank is based on Time + Penality time</li>
		</ul>
	  <div class="container-flux table-responsive">
        <table id="thetable" class="table table-sm table-hover" data-fixed-columns=true data-fixed-number=1>
          <thead>
            <tr>
              <th scope="col">User</th>
              <th scope="col">Rank</th>
              <th scope="col">Passed</th>
              <th scope="col">Time</th>
              <th scope="col">Penalty</th>
              {{range $name := .Homework.Cases}}
              <th>{{$name}}</th>
              {{end}}
            </tr>
          </thead>
          <tbody>
            {{range $row := .Rows}}
            <tr>
              <th>{{$row.Submission.User}}</th>
              <td class="center">{{$row.Rank}}</td>
              <td class="center">{{$row.NumPassed}}</td>
              <td>
                {{$row.TotalTime | printf "%.2f"}}
              </td>
              {{if gt $row.PenaltyTime 0.0}}<td class="penalty">{{$row.PenaltyTime | printf "%.0f"}}</td>{{else}}<td></td>{{end}}
              {{range $cell := $row.Cells}}
              <td class="{{$cell.Class}}" title="{{$cell.Title}}">{{$cell.Value}}</td>
              {{end}}
            </tr>
            {{end}}
          </tbody>
        </table>
      </div>
    </div>
    
    

    <!-- Optional JavaScript -->
    <!-- jQuery first, then Popper.js, then Bootstrap JS -->
    <script src="https://code.jquery.com/jquery-3.5.1.slim.min.js" integrity="sha384-DfXdz2htPH0lsSSs5nCTpuj/zy4C+OGpamoFVy38MVBnE+IbbVYUew+OrCXaRkfj" crossorigin="anonymous"></script>
    <script src="https://cdn.jsdelivr.net/npm/popper.js@1.16.1/dist/umd/popper.min.js" integrity="sha384-9/reFTGAW83EW2RDu2S0VKaIzap3H66lZH81PoYlFhbGU+6BZp6G7niu735Sk7lN" crossorigin="anonymous"></script>
    <script src="https://stackpath.bootstrapcdn.com/bootstrap/4.5.2/js/bootstrap.min.js" integrity="sha384-B4gt1jrGC7Jh4AgTPSdUtOBvfO8shuf57BaghqFfPlYxofvL8/KUEfYiJOMMV+rV" crossorigin="anonymous"></script>
    
    <script>
      var table = document.getElementById("thetable");
      
      for (var i = 5; i < table.rows[0].cells.length; i++) {
        var minimum = 65536;
        for (var j = 1; j < table.rows.length; j++) {
          if (!table.rows[j].cells[i].classList.contains("failed") && !table.rows[j].cells[i].classList.contains("empty")) {
            var tmp = parseFloat(table.rows[j].cells[i].textContent);
            if (!isNaN(tmp)) minimum = Math.min(minimum, tmp);
          }
        }
        for (var j = 1; j < table.rows.length; j++) {
			if (table.rows[j].cells[i].title == "accepted") {	
          		var time = parseFloat(table.rows[j].cells[i].textContent)
          		var percentage = minimum / time;
          		table.rows[j].cells[i].style.backgroundColor = "rgba(56, 142, 60, " + percentage.toString() + ")";
          		if (time == minimum) table.rows[j].cells[i].style.color = "white";
        	} else if (table.rows[j].cells[i].title == "time limit exceeded") {
          		table.rows[j].cells[i].style.backgroundColor = "rgb(255,193,7)";
        	} else if (table.rows[j].cells[i].title == "time limit exceeded+") {
          		table.rows[j].cells[i].style.backgroundColor = "#FF9800";
			} else {
          		table.rows[j].cells[i].style.backgroundColor = "rgb(244,67,54)";
			}
			/*
        	if (table.rows[j].cells[i].classList.contains("empty") || 
				table.rows[j].cells[i].title == "wrong answer" || 
				table.rows[j].cells[i].title == "runtime error" || 
				table.rows[j].cells[i].title == "internal error") {
          		table.rows[j].cells[i].style.backgroundColor = "rgb(244,67,54)";
			*/

			table.rows[j].cells[i].style.border = 0;
      	}
      }
    </script>

    <!-- Global site tag (gtag.js) - Google Analytics -->
    <script async src="https://www.googletagmanager.com/gtag/js?id=UA-162001246-1"></script>
    <script>
      window.dataLayer = window.dataLayer || [];
      function gtag() { dataLayer.push(arguments); }
      gtag('js', new Date());

      gtag('config', 'UA-162001246-1');
    </script>
  </body>
</html>
`
