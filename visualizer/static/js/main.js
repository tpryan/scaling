document.addEventListener('DOMContentLoaded', function() {
    document.querySelector(".send").addEventListener("click", distribute);  

    setInterval(pollLoad, 1000);
});


function distribute() {
    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
         if (this.readyState == 4 && this.status == 200) {
             alert(this.responseText);
         }
    };
    xhttp.open("GET", "/api/distribute?n=100&c=1&url=http://docker.for.mac.localhost:8081/", true);
    xhttp.setRequestHeader("Content-type", "application/json");
    xhttp.send();
}

function pollLoad() {
    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
         if (this.readyState == 4 && this.status == 200) {
            reportLoad(this.responseText);
         }
    };
    xhttp.open("GET", "/api/index", true);
    xhttp.setRequestHeader("Content-type", "application/json");
    xhttp.send();
}

function reportLoad(content){
    var loadIndex = JSON.parse(content);


    for (var instance in loadIndex) {
        if (loadIndex.hasOwnProperty(instance)) {
            updateInstance(loadIndex[instance]);
        }
    };


}

function updateInstance(instance){

    var id = "#instance-" + instance.id

    var ui = document.querySelector(id);

    if (ui != null) {
        document.querySelector(id + " .count").innerHTML = instance.count;  
    } else {
        var instanceDiv = document.createElement("div");
        instanceDiv.id = "instance-" + instance.id;
        instanceDiv.classList.add(instance.env.toLowerCase())
        var countDiv = document.createElement("div");
        countDiv.innerHTML = instance.count;  
        countDiv.classList.add("count");
        instanceDiv.appendChild(countDiv);
        document.querySelector(".load-info").appendChild(instanceDiv);

    }

}