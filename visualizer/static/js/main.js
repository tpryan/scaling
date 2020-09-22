document.addEventListener('DOMContentLoaded', function() {
    document.querySelector(".send").addEventListener("click", distribute);  
    document.querySelector(".clear").addEventListener("click", clear); 

    setInterval(pollLoad, 100);
    setInterval(pollGenerators, 100);
});


function distribute() {
    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
         if (this.readyState == 4 && this.status == 200) {
            console.log("Fireing load - success");
            console.log(this.responseText);
         }
    };
    console.log("Fireing load");
    xhttp.open("GET", "/api/distribute?n=1000&c=1&url=http://docker.for.mac.localhost:8081/", true);
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
        var envType = instance.env.toLowerCase();

        var imagePath = "";

        switch (envType) {
            case "local":
                imagePath = "img/local.svg";
              break;
            case "computeengine":
                imagePath = "img/computeengine.svg";
              break;
            case "appengine":
                imagePath = "img/appengine.svg";
              break;
            default :
                imagePath = "img/local.svg";
        }

        var instanceDiv = document.createElement("div");
        instanceDiv.id = "instance-" + instance.id;
        instanceDiv.classList.add(envType);
        instanceDiv.classList.add("instance");
        
        var countDiv = document.createElement("div");
        countDiv.innerHTML = instance.count;  
        countDiv.classList.add("count");

        var idDiv = document.createElement("div");
        idDiv.innerHTML = instance.id;  
        idDiv.classList.add("id");
        
        var imgEle = document.createElement("img");
        imgEle.src =  imagePath;
        
        instanceDiv.appendChild(imgEle);
        instanceDiv.appendChild(idDiv);
        instanceDiv.appendChild(countDiv);

        document.querySelector(".load-info").appendChild(instanceDiv);

    }

}

function pollGenerators() {
    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
         if (this.readyState == 4 && this.status == 200) {
            reportGenerators(this.responseText);
         }
    };
    xhttp.open("GET", "/api/nodelist", true);
    xhttp.setRequestHeader("Content-type", "application/json");
    xhttp.send();
}

function reportGenerators(content){

    var loadIndex = JSON.parse(content);

    for (var instance in loadIndex) {
        if (loadIndex.hasOwnProperty(instance)) {
            updateLoadGenerator(loadIndex[instance]);
        }
    };
}

function updateLoadGenerator(node){

    var id = "#generator-" + node.id

    var ui = document.querySelector(id);

    if (ui != null) {
        if (node.active){
           ui.classList.add("active");  
        } else{
           ui.classList.remove("active");  
        }
        
    } else {

        var nodeDiv = document.createElement("div");
        nodeDiv.id = "generator-" + node.id;
        nodeDiv.classList.add("node")

        var idDiv = document.createElement("div");
        idDiv.innerHTML = node.id;  
        idDiv.classList.add("id");
        
        var imgEle = document.createElement("img");
        imgEle.src =  "img/computeengine.svg";
        
        nodeDiv.appendChild(imgEle);
        nodeDiv.appendChild(idDiv);
        document.querySelector(".load-generators").appendChild(nodeDiv);

    }

}

function clear() {
    console.log("Clear called")
    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
         if (this.readyState == 4 && this.status == 200) {
            console.log(this.responseText);
            document.querySelector(".load-generators").innerHTML = "";
            document.querySelector(".load-info").innerHTML = "";
         }
    };
    xhttp.open("GET", "/api/clear", true);
    xhttp.setRequestHeader("Content-type", "application/json");
    xhttp.send();
}